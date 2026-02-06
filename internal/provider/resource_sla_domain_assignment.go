// Copyright 2024 Rubrik, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

// Note: The description uses acute accent marks (´) instead of backticks (`)
// because backticks would terminate the raw string literal. The Terraform
// registry documentation renderer handles acute accents correctly.
const resourceSLADomainAssignmentDescription = `
The ´polaris_sla_domain_assignment´ resource is used to assign SLA domains to
objects.

When an object is removed from the ´polaris_sla_domain_assignment´ resource, it
will inherit the SLA Domain of its parent object. If there is no parent object
or the parent object doesn't have an SLA Domain, the object will be unprotected.
Existing snapshots of the object will be retained according to the SLA Domain
inherited from the parent object. If the parent object doesn't have an SLA
Domain, the existing snapshots will be retained forever.

The ´assignment_type´ attribute controls how the SLA domain is assigned:

  * ´protectWithSlaId´ - Protect objects with the specified SLA domain. Requires ´sla_domain_id´ to be set.

  * ´doNotProtect´ - Do not protect objects. Requires that ´sla_domain_id´ isn't set.

~> **Note:** When importing, ´apply_changes_to_existing_snapshots´ and
´apply_changes_to_non_policy_snapshots´ cannot be retrieved from the API.
These attributes will use their default values after import.
`

// doNotProtectID is the sentinel resource ID used when the assignment type is
// DoNotProtect.
const doNotProtectID = "doNotProtect"

func resourceSLADomainAssignment() *schema.Resource {
	return &schema.Resource{
		CreateContext: createSLADomainAssignment,
		ReadContext:   readSLADomainAssignment,
		UpdateContext: updateSLADomainAssignment,
		DeleteContext: deleteSLADomainAssignment,

		Description: description(resourceSLADomainAssignmentDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SLA domain ID (UUID).",
			},
			keyAssignmentType: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  string(gqlsla.ProtectWithSLA),
				ValidateFunc: validation.StringInSlice([]string{
					string(gqlsla.ProtectWithSLA),
					string(gqlsla.DoNotProtect),
				}, false),
				Description: "SLA domain assignment type. Valid values are `protectWithSlaId` and " +
					"`doNotProtect`. Defaults to `protectWithSlaId`.",
			},
			keyObjectIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
				MinItems:    1,
				Required:    true,
				Description: "Object IDs (UUID).",
			},
			keySLADomainID: {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "SLA domain ID (UUID). Required when `assignment_type` is `protectWithSlaId`.",
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{keyExistingSnapshotRetention},
			},
			keyApplyChangesToExistingSnapshots: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       true,
				Description:   "Apply SLA changes to existing snapshots. Only valid when `assignment_type` is `protectWithSlaId`. Defaults to `true`.",
				ConflictsWith: []string{keyExistingSnapshotRetention},
			},
			keyApplyChangesToNonPolicySnapshots: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				Description:   "Apply SLA changes to non-policy snapshots. Only valid when `assignment_type` is `protectWithSlaId`. Defaults to `false`.",
				ConflictsWith: []string{keyExistingSnapshotRetention},
			},
			keyExistingSnapshotRetention: {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(gqlsla.RetainSnapshots),
					string(gqlsla.KeepForever),
					string(gqlsla.ExpireImmediately),
				}, false),
				Description: "Existing snapshot retention policy. Only valid when `assignment_type` is " +
					"`doNotProtect`. Valid values are `RETAIN_SNAPSHOTS`, `KEEP_FOREVER`, and `EXPIRE_IMMEDIATELY`.",
				ConflictsWith: []string{keySLADomainID, keyApplyChangesToExistingSnapshots, keyApplyChangesToNonPolicySnapshots},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: importSLADomainAssignment,
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceSLADomainAssignmentV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceSLADomainAssignmentStateUpgradeV0,
			Version: 0,
		}},
	}
}

func createSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "createSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var objectIDs []uuid.UUID
	for _, id := range d.Get(keyObjectIDs).(*schema.Set).List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		objectIDs = append(objectIDs, id)
	}

	assignType := gqlsla.AssignmentType(d.Get(keyAssignmentType).(string))
	domainID := d.Get(keySLADomainID).(string)
	snapshotRetention := d.Get(keyExistingSnapshotRetention).(string)
	applyToExisting := d.Get(keyApplyChangesToExistingSnapshots).(bool)
	applyToNonPolicy := d.Get(keyApplyChangesToNonPolicySnapshots).(bool)

	params := gqlsla.AssignDomainParams{
		DomainAssignType: assignType,
		ObjectIDs:        objectIDs,
		// Always set AllSubHierarchyType for cloud account objects that support multiple
		// workload hierarchies. The backend requires this field to be set (not blank) for
		// these object types, despite the GraphQL documentation suggesting it can be left
		// blank. Since we cannot read this value back from the API, we always use this
		// default to ensure assignments work correctly for all object types.
		ApplicableWorkloadType: "AllSubHierarchyType",
	}

	// For protectWithSlaId, sla_domain_id is required.
	// For doNotProtect, sla_domain_id must be empty and existing_snapshot_retention is allowed.
	var expectedDomainID string
	switch assignType {
	case gqlsla.ProtectWithSLA:
		if domainID == "" {
			return diag.Errorf("sla_domain_id is required when assignment_type is %q", gqlsla.ProtectWithSLA)
		}
		if snapshotRetention != "" {
			return diag.Errorf("existing_snapshot_retention is only valid when assignment_type is %q", gqlsla.DoNotProtect)
		}
		if applyToNonPolicy && !applyToExisting {
			return diag.Errorf("apply_changes_to_non_policy_snapshots requires apply_changes_to_existing_snapshots to be true")
		}
		dID, err := uuid.Parse(domainID)
		if err != nil {
			return diag.FromErr(err)
		}
		params.DomainID = &dID
		params.ApplyToExistingSnapshots = ptr(applyToExisting)
		params.ApplyToNonPolicySnapshots = ptr(applyToNonPolicy)
		expectedDomainID = domainID
	case gqlsla.DoNotProtect:
		if domainID != "" {
			return diag.Errorf("sla_domain_id must be empty when assignment_type is %q", gqlsla.DoNotProtect)
		}
		if snapshotRetention != "" {
			params.ExistingSnapshotRetention = gqlsla.ExistingSnapshotRetention(snapshotRetention)
		}
		expectedDomainID = hierarchy.DoNotProtectSLAID
	}

	if err := sla.Wrap(client).AssignDomain(ctx, params); err != nil {
		return diag.FromErr(err)
	}

	// Wait for the assignment to be applied.
	if err := waitForAssignment(ctx, client, expectedDomainID, objectIDs); err != nil {
		return diag.FromErr(err)
	}

	// Set the resource ID based on assignment type.
	if assignType == gqlsla.DoNotProtect {
		d.SetId(doNotProtectID)
	} else {
		d.SetId(params.DomainID.String())
	}

	return nil
}

func readSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "readSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	assignType := gqlsla.AssignmentType(d.Get(keyAssignmentType).(string))
	domainIDStr := d.Get(keySLADomainID).(string)

	// For protectWithSlaId, verify the SLA domain still exists.
	if assignType == gqlsla.ProtectWithSLA && domainIDStr != "" {
		domainID, err := uuid.Parse(domainIDStr)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, err := sla.Wrap(client).DomainByID(ctx, domainID); err != nil {
			if errors.Is(err, graphql.ErrNotFound) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	// Query each object to check if it still has the expected SLA directly assigned.
	objectIDs := d.Get(keyObjectIDs).(*schema.Set)
	remainingIDs := &schema.Set{F: schema.HashString}
	for _, idStr := range objectIDs.List() {
		objectID, err := uuid.Parse(idStr.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		obj, err := sla.Wrap(client).HierarchyObjectByID(ctx, objectID)
		if err != nil {
			if errors.Is(err, graphql.ErrNotFound) {
				// Object no longer exists, skip it.
				continue
			}
			return diag.FromErr(err)
		}

		// Check if the object has the expected SLA directly assigned.
		if obj.SLAAssignment != gqlsla.Direct {
			continue
		}

		switch assignType {
		case gqlsla.DoNotProtect:
			// For doNotProtect, check effectiveSlaDomain because with
			// RETAIN_SNAPSHOTS, configuredSlaDomain keeps the previous SLA
			// (converted to retention type) while effectiveSlaDomain becomes
			// DO_NOT_PROTECT.
			if obj.EffectiveSLADomain.ID == hierarchy.DoNotProtectSLAID {
				remainingIDs.Add(idStr)
			}
		case gqlsla.ProtectWithSLA:
			if obj.ConfiguredSLADomain.ID == domainIDStr {
				remainingIDs.Add(idStr)
			}
		}
	}

	if remainingIDs.Len() == 0 {
		d.SetId("")
		return nil
	}

	if err := d.Set(keyObjectIDs, remainingIDs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "updateSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	addObjectIDs, removeObjectIDs, totalObjectIDs, err := diffObjectIDs(d)
	if err != nil {
		return diag.FromErr(err)
	}

	assignType := gqlsla.AssignmentType(d.Get(keyAssignmentType).(string))
	domainIDStr := d.Get(keySLADomainID).(string)
	snapshotRetention := d.Get(keyExistingSnapshotRetention).(string)
	applyToExisting := d.Get(keyApplyChangesToExistingSnapshots).(bool)
	applyToNonPolicy := d.Get(keyApplyChangesToNonPolicySnapshots).(bool)

	var domainID uuid.UUID
	// Validate based on assignment_type.
	switch assignType {
	case gqlsla.ProtectWithSLA:
		if domainIDStr == "" {
			return diag.Errorf("sla_domain_id is required when assignment_type is %q", gqlsla.ProtectWithSLA)
		}
		domainID, err = uuid.Parse(domainIDStr)
		if err != nil {
			return diag.FromErr(err)
		}
		if snapshotRetention != "" {
			return diag.Errorf("existing_snapshot_retention is only valid when assignment_type is %q", gqlsla.DoNotProtect)
		}
		if applyToNonPolicy && !applyToExisting {
			return diag.Errorf("apply_changes_to_non_policy_snapshots requires apply_changes_to_existing_snapshots to be true")
		}
	case gqlsla.DoNotProtect:
		if domainIDStr != "" {
			return diag.Errorf("sla_domain_id must be empty when assignment_type is %q", gqlsla.DoNotProtect)
		}
	}

	// Handle assignment type changes.
	if d.HasChange(keyAssignmentType) || d.HasChange(keySLADomainID) || d.HasChange(keyExistingSnapshotRetention) ||
		d.HasChange(keyApplyChangesToExistingSnapshots) || d.HasChange(keyApplyChangesToNonPolicySnapshots) {
		// When assignment type, domain ID, snapshot retention, or apply settings change, we need to reassign all objects.
		addObjectIDs = totalObjectIDs
	}

	if len(addObjectIDs) > 0 {
		params := gqlsla.AssignDomainParams{
			DomainAssignType:       assignType,
			ObjectIDs:              addObjectIDs,
			ApplicableWorkloadType: "AllSubHierarchyType",
		}

		// For protectWithSlaId, use apply settings from schema.
		// For doNotProtect, existing_snapshot_retention is allowed.
		var expectedAssignment string
		switch assignType {
		case gqlsla.ProtectWithSLA:
			params.DomainID = &domainID
			params.ApplyToExistingSnapshots = ptr(applyToExisting)
			params.ApplyToNonPolicySnapshots = ptr(applyToNonPolicy)
			expectedAssignment = domainIDStr
		case gqlsla.DoNotProtect:
			if snapshotRetention != "" {
				params.ExistingSnapshotRetention = gqlsla.ExistingSnapshotRetention(snapshotRetention)
			}
			expectedAssignment = hierarchy.DoNotProtectSLAID
		}

		if err := sla.Wrap(client).AssignDomain(ctx, params); err != nil {
			return diag.FromErr(err)
		}

		// Wait for the assignment to be applied.
		if err := waitForAssignment(ctx, client, expectedAssignment, addObjectIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if len(removeObjectIDs) > 0 {
		if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
			DomainAssignType:          gqlsla.NoAssignment,
			ObjectIDs:                 removeObjectIDs,
			ApplyToExistingSnapshots:  ptr(true),
			ApplyToNonPolicySnapshots: ptr(false),
			ApplicableWorkloadType:    "AllSubHierarchyType",
		}); err != nil {
			return diag.FromErr(err)
		}

		// Wait for the unassignment to be applied.
		if err := waitForAssignment(ctx, client, hierarchy.UnprotectedSLAID, removeObjectIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	// Update the resource ID based on assignment type.
	if assignType == gqlsla.DoNotProtect {
		d.SetId(doNotProtectID)
	} else {
		d.SetId(domainID.String())
	}

	return nil
}

func deleteSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "deleteSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the current SLA domain ID to track what we're removing.
	currentSLADomainID := d.Id()
	if currentSLADomainID == doNotProtectID {
		currentSLADomainID = hierarchy.DoNotProtectSLAID
	}

	// Filter objects to only unassign those that are still assigned to this
	// resource's SLA. This prevents issues where concurrent creates might have
	// already reassigned the object to a different SLA.
	var objectsToUnassign []uuid.UUID
	for _, objectIDStr := range d.Get(keyObjectIDs).(*schema.Set).List() {
		objectID, err := uuid.Parse(objectIDStr.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		obj, err := sla.Wrap(client).HierarchyObjectByID(ctx, objectID)
		if err != nil {
			if errors.Is(err, graphql.ErrNotFound) {
				// Object no longer exists, skip.
				continue
			}
			return diag.FromErr(err)
		}

		// Only unassign if the object still has our SLA directly assigned.
		if obj.SLAAssignment != gqlsla.Direct {
			continue
		}

		switch currentSLADomainID {
		case hierarchy.DoNotProtectSLAID:
			// For doNotProtect, check both effective and configured SLA.
			// If configuredSlaDomain is a regular SLA (not DO_NOT_PROTECT or
			// UNPROTECTED), it means a concurrent create has already assigned
			// the object to a new SLA, so we should skip unassignment.
			if obj.EffectiveSLADomain.ID == hierarchy.DoNotProtectSLAID {
				configuredID := obj.ConfiguredSLADomain.ID
				if configuredID == hierarchy.DoNotProtectSLAID || configuredID == hierarchy.UnprotectedSLAID {
					objectsToUnassign = append(objectsToUnassign, objectID)
				}
			}
		default:
			// For regular SLAs, check configured SLA.
			if obj.ConfiguredSLADomain.ID == currentSLADomainID {
				objectsToUnassign = append(objectsToUnassign, objectID)
			}
		}
	}

	if len(objectsToUnassign) > 0 {
		if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
			DomainAssignType:          gqlsla.NoAssignment,
			ObjectIDs:                 objectsToUnassign,
			ApplyToExistingSnapshots:  ptr(true),
			ApplyToNonPolicySnapshots: ptr(false),
			ApplicableWorkloadType:    "AllSubHierarchyType",
		}); err != nil {
			return diag.FromErr(err)
		}

		// Wait for the assignment to be removed.
		if err := waitForAssignmentRemoval(ctx, client, currentSLADomainID, objectsToUnassign); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

// Note, the SLA domain assignment resource is designed to only manage SLA
// domain assignments owned by the resource. An import on the other hand will
// take ownership of all SLA domain assignments for a domain.
//
// Import formats:
//   - <sla_domain_id> - for protectWithSlaId assignments (imports all objects
//     directly assigned to the SLA domain)
//   - doNotProtect:<object_id1>,<object_id2>,... - for doNotProtect assignments
//     (imports the specified objects, verifying they have DO_NOT_PROTECT assigned)
func importSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	tflog.Trace(ctx, "importSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	importID := d.Id()

	// Check if this is a doNotProtect import.
	if strings.HasPrefix(importID, "doNotProtect:") {
		return importDoNotProtectAssignment(ctx, d, client, importID)
	}

	// Otherwise, treat as a protectWithSlaId import.
	return importProtectWithSLAAssignment(ctx, d, client, importID)
}

// importProtectWithSLAAssignment imports a protectWithSlaId assignment by
// querying all objects directly assigned to the specified SLA domain.
func importProtectWithSLAAssignment(ctx context.Context, d *schema.ResourceData, client *polaris.Client, importID string) ([]*schema.ResourceData, error) {
	domainID, err := uuid.Parse(importID)
	if err != nil {
		return nil, fmt.Errorf("invalid SLA domain ID %q: %w", importID, err)
	}

	// Return a human-readable error message if the SLA domain doesn't exist.
	if _, err := sla.Wrap(client).DomainByID(ctx, domainID); err != nil {
		return nil, err
	}

	objects, err := sla.Wrap(client).DomainObjects(ctx, domainID, "")
	if err != nil {
		return nil, err
	}

	objectIDsSet := &schema.Set{F: schema.HashString}
	for _, object := range objects {
		objectIDsSet.Add(object.ID.String())
	}
	if err := d.Set(keyObjectIDs, objectIDsSet); err != nil {
		return nil, err
	}
	if err := d.Set(keySLADomainID, domainID.String()); err != nil {
		return nil, err
	}
	if err := d.Set(keyAssignmentType, string(gqlsla.ProtectWithSLA)); err != nil {
		return nil, err
	}

	// Keep the domain ID as the resource ID (already set from import input).

	return []*schema.ResourceData{d}, nil
}

// importDoNotProtectAssignment imports a doNotProtect assignment by verifying
// that the specified objects have DO_NOT_PROTECT as their effective SLA.
func importDoNotProtectAssignment(ctx context.Context, d *schema.ResourceData, client *polaris.Client, importID string) ([]*schema.ResourceData, error) {
	// Parse the object IDs from the import ID format: doNotProtect:<id1>,<id2>,...
	objectIDsStr := strings.TrimPrefix(importID, "doNotProtect:")
	if objectIDsStr == "" {
		return nil, fmt.Errorf("import format for doNotProtect requires object IDs: 'doNotProtect:<object_id1>,<object_id2>,...'")
	}

	objectIDStrs := strings.Split(objectIDsStr, ",")
	objectIDs := make([]uuid.UUID, 0, len(objectIDStrs))
	for _, idStr := range objectIDStrs {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		objectID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid object ID %q: %w", idStr, err)
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return nil, fmt.Errorf("import format for doNotProtect requires at least one object ID: 'doNotProtect:<object_id1>,<object_id2>,...'")
	}

	// Verify each object has DO_NOT_PROTECT as its effective SLA.
	objectIDsSet := &schema.Set{F: schema.HashString}
	for _, objectID := range objectIDs {
		obj, err := sla.Wrap(client).HierarchyObjectByID(ctx, objectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get object %s: %w", objectID, err)
		}

		if obj.EffectiveSLADomain.ID != hierarchy.DoNotProtectSLAID {
			return nil, fmt.Errorf("object %s does not have DO_NOT_PROTECT assigned (effective SLA: %s)", objectID, obj.EffectiveSLADomain.Name)
		}

		if obj.SLAAssignment != gqlsla.Direct {
			return nil, fmt.Errorf("object %s does not have DO_NOT_PROTECT directly assigned (assignment type: %s)", objectID, obj.SLAAssignment)
		}

		objectIDsSet.Add(objectID.String())
	}

	if err := d.Set(keyObjectIDs, objectIDsSet); err != nil {
		return nil, err
	}
	if err := d.Set(keyAssignmentType, string(gqlsla.DoNotProtect)); err != nil {
		return nil, err
	}

	// Set the resource ID to "doNotProtect" for doNotProtect assignments.
	d.SetId("doNotProtect")

	return []*schema.ResourceData{d}, nil
}

// waitForAssignment waits for all objects to have the expected SLA directly assigned.
// For doNotProtect, expectedDomainID should be hierarchy.DoNotProtectSLAID.
// For protectWithSlaId, expectedDomainID should be the SLA domain ID string.
// For unassignment (noAssignment), expectedDomainID should be hierarchy.UnprotectedSLAID.
func waitForAssignment(ctx context.Context, client *polaris.Client, expectedDomainID string, objectIDs []uuid.UUID) error {
	startTime := time.Now()

	for {
		pending := make([]uuid.UUID, 0, len(objectIDs))

		for _, objectID := range objectIDs {
			obj, err := sla.Wrap(client).HierarchyObjectByID(ctx, objectID)
			if err != nil {
				return err
			}

			switch expectedDomainID {
			case hierarchy.UnprotectedSLAID:
				// For unprotected, we just need to wait until there's no direct assignment.
				if obj.SLAAssignment == gqlsla.Direct || obj.ConfiguredSLADomain.ID != hierarchy.UnprotectedSLAID {
					tflog.Debug(ctx, "waiting for assignment to be removed (sla_assignment != DIRECT)", map[string]any{
						"object_id":      objectID.String(),
						"configured_sla": obj.ConfiguredSLADomain.ID,
						"effective_sla":  obj.EffectiveSLADomain.ID,
						"elapsed":        time.Since(startTime).Truncate(time.Second).String(),
					})
					pending = append(pending, objectID)
				}
			case hierarchy.DoNotProtectSLAID:
				// For doNotProtect, we check effectiveSlaDomain because with
				// RETAIN_SNAPSHOTS, configuredSlaDomain keeps the previous SLA.
				if obj.EffectiveSLADomain.ID != hierarchy.DoNotProtectSLAID {
					tflog.Debug(ctx, "waiting for transition to DO_NOT_PROTECT", map[string]any{
						"object_id":      objectID.String(),
						"sla_assignment": string(obj.SLAAssignment),
						"configured_sla": obj.ConfiguredSLADomain.ID,
						"effective_sla":  obj.EffectiveSLADomain.ID,
						"elapsed":        time.Since(startTime).Truncate(time.Second).String(),
					})
					pending = append(pending, objectID)
				}
			default:
				// For other cases, check if the object has the expected SLA directly assigned.
				if obj.SLAAssignment != gqlsla.Direct || obj.ConfiguredSLADomain.ID != expectedDomainID {
					tflog.Debug(ctx, "waiting for SLA assignment", map[string]any{
						"object_id":      objectID.String(),
						"expected_sla":   expectedDomainID,
						"sla_assignment": string(obj.SLAAssignment),
						"configured_sla": obj.ConfiguredSLADomain.ID,
						"effective_sla":  obj.EffectiveSLADomain.ID,
						"elapsed":        time.Since(startTime).Truncate(time.Second).String(),
					})
					pending = append(pending, objectID)
				}
			}
		}

		if len(pending) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// waitForAssignmentRemoval waits for objects to no longer have the specified SLA
// directly assigned. Unlike waitForAssignment, this function accepts that objects
// may be reassigned to a different SLA (not just unprotected) as a valid completion
// state. This prevents race conditions when Terraform replaces resources by deleting
// the old assignment and creating new ones concurrently.
//
// Parameters:
//   - currentSLADomainID: The SLA domain ID being removed (can be a UUID string,
//     hierarchy.DoNotProtectSLAID, or hierarchy.UnprotectedSLAID)
//   - objectIDs: The objects to check
//
// The function completes successfully when all objects either:
//  1. No longer have a direct assignment (inherited or unprotected), OR
//  2. Have been directly assigned to a different SLA domain
func waitForAssignmentRemoval(ctx context.Context, client *polaris.Client, currentSLADomainID string, objectIDs []uuid.UUID) error {
	startTime := time.Now()

	for {
		pending := make([]uuid.UUID, 0, len(objectIDs))

		for _, objectID := range objectIDs {
			obj, err := sla.Wrap(client).HierarchyObjectByID(ctx, objectID)
			if err != nil {
				return err
			}

			// Check if the object still has the old SLA directly assigned.
			stillHasOldAssignment := false

			switch currentSLADomainID {
			case hierarchy.DoNotProtectSLAID:
				// For doNotProtect, check if effective SLA is still DO_NOT_PROTECT with direct assignment.
				if obj.SLAAssignment == gqlsla.Direct && obj.EffectiveSLADomain.ID == hierarchy.DoNotProtectSLAID {
					stillHasOldAssignment = true
				}
			case hierarchy.UnprotectedSLAID:
				// For unprotected, check if it's still unprotected with direct assignment.
				// Note: This case is unlikely in practice since unprotected means no direct assignment.
				if obj.SLAAssignment == gqlsla.Direct && obj.ConfiguredSLADomain.ID == hierarchy.UnprotectedSLAID {
					stillHasOldAssignment = true
				}
			default:
				// For regular SLA domains, check if the configured SLA matches the old one.
				if obj.SLAAssignment == gqlsla.Direct && obj.ConfiguredSLADomain.ID == currentSLADomainID {
					stillHasOldAssignment = true
				}
			}

			if stillHasOldAssignment {
				tflog.Debug(ctx, "waiting for old SLA assignment to be removed or replaced", map[string]any{
					"object_id":      objectID.String(),
					"old_sla":        currentSLADomainID,
					"sla_assignment": string(obj.SLAAssignment),
					"configured_sla": obj.ConfiguredSLADomain.ID,
					"effective_sla":  obj.EffectiveSLADomain.ID,
					"elapsed":        time.Since(startTime).Truncate(time.Second).String(),
				})
				pending = append(pending, objectID)
			} else {
				// Object no longer has the old assignment - either it's been reassigned
				// to a different SLA or it's now inherited/unprotected. Either way, our
				// delete operation is complete for this object.
				tflog.Debug(ctx, "old SLA assignment removed or replaced", map[string]any{
					"object_id":      objectID.String(),
					"old_sla":        currentSLADomainID,
					"sla_assignment": string(obj.SLAAssignment),
					"configured_sla": obj.ConfiguredSLADomain.ID,
					"effective_sla":  obj.EffectiveSLADomain.ID,
				})
			}
		}

		if len(pending) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// diffObjectIDs returns the object IDs to add, remove and the total which
// should be assigned to the SLA domain after the assignment.
func diffObjectIDs(d *schema.ResourceData) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
	oldObjIDs, newObjIDs := d.GetChange(keyObjectIDs)

	addSet := make(map[uuid.UUID]struct{}, newObjIDs.(*schema.Set).Len())
	for _, id := range newObjIDs.(*schema.Set).List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return nil, nil, nil, err
		}
		addSet[id] = struct{}{}
	}

	// Total object IDs is the union of new object IDs and object IDs to keep.
	totalObjIDs := make([]uuid.UUID, 0, len(addSet))
	for id := range addSet {
		totalObjIDs = append(totalObjIDs, id)
	}

	removeObjIDs := make([]uuid.UUID, 0, oldObjIDs.(*schema.Set).Len())
	for _, id := range oldObjIDs.(*schema.Set).List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return nil, nil, nil, err
		}
		if _, ok := addSet[id]; !ok {
			removeObjIDs = append(removeObjIDs, id)
		} else {
			delete(addSet, id)
		}
	}

	addObjIDs := make([]uuid.UUID, 0, len(addSet))
	for id := range addSet {
		addObjIDs = append(addObjIDs, id)
	}

	return addObjIDs, removeObjIDs, totalObjIDs, nil
}

func ptr[T any](v T) *T {
	return &v
}
