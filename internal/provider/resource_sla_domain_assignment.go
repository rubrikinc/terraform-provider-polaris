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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const resourceSLADomainAssignmentDescription = `
The ´polaris_sla_domain_assignment´ resource is used to assign SLA domains to
objects.

When an object is removed from the ´polaris_sla_domain_assignment´ resource, it
will inherit the SLA Domain of its parent object. If there is no parent object
or the parent object doesn't have an SLA Domain, the object will be unprotected.
Existing snapshots of the object will be retained according to the SLA Domain
inherited from the parent object. If the parent object doesn't have an SLA
Domain, the existing snapshots will be retained forever.

-> **Note:** As of now, it's not possible to assign objects as Do Not Protect.
`

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
				Type:         schema.TypeString,
				Required:     true,
				Description:  "SLA domain ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: importSLADomainAssignment,
		},
	}
}

func createSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "createSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	domainID, err := uuid.Parse(d.Get(keySLADomainID).(string))
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

	if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
		DomainID:                  &domainID,
		DomainAssignType:          gqlsla.ProtectWithSLA,
		ObjectIDs:                 objectIDs,
		ApplyToExistingSnapshots:  ptr(true),
		ApplyToNonPolicySnapshots: ptr(false),
	}); err != nil {
		return diag.FromErr(err)
	}
	if err := waitForAssignment(ctx, client, domainID, objectIDs); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(domainID.String())
	return nil
}

func readSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "readSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	domainID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Make sure the SLA domain still exists.
	if _, err := sla.Wrap(client).DomainByID(ctx, domainID); err != nil {
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if err := d.Set(keySLADomainID, domainID.String()); err != nil {
		return diag.FromErr(err)
	}

	objectIDs := d.Get(keyObjectIDs).(*schema.Set)
	idSet := make(map[uuid.UUID]struct{}, objectIDs.Len())
	for _, id := range objectIDs.List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		idSet[id] = struct{}{}
	}

	objects, err := sla.Wrap(client).DomainObjects(ctx, domainID, "")
	if err != nil {
		return diag.FromErr(err)
	}
	for _, object := range objects {
		delete(idSet, object.ID)
	}

	for id := range idSet {
		objectIDs.Remove(id.String())
	}
	if err := d.Set(keyObjectIDs, &objectIDs); err != nil {
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

	domainID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	newDomainID := domainID
	if d.HasChange(keySLADomainID) {
		newDomainID, err = uuid.Parse(d.Get(keySLADomainID).(string))
		if err != nil {
			return diag.FromErr(err)
		}

		// If the SLA domain ID has changed, we need to move the new objects
		// and the objects to keep.
		addObjectIDs = totalObjectIDs
	}

	if len(addObjectIDs) > 0 {
		if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
			DomainID:                  &newDomainID,
			DomainAssignType:          gqlsla.ProtectWithSLA,
			ObjectIDs:                 addObjectIDs,
			ApplyToExistingSnapshots:  ptr(true),
			ApplyToNonPolicySnapshots: ptr(false),
		}); err != nil {
			return diag.FromErr(err)
		}
		if err := waitForAssignment(ctx, client, newDomainID, addObjectIDs); err != nil {
			return diag.FromErr(err)
		}
	}
	if len(removeObjectIDs) > 0 {
		if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
			DomainAssignType:          gqlsla.NoAssignment,
			ObjectIDs:                 removeObjectIDs,
			ApplyToExistingSnapshots:  ptr(true),
			ApplyToNonPolicySnapshots: ptr(false),
		}); err != nil {
			return diag.FromErr(err)
		}
		if err := waitForUnassignment(ctx, client, domainID, removeObjectIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(newDomainID.String())
	return nil
}

func deleteSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "deleteSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	domainID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var objectIDs []uuid.UUID
	for _, objectID := range d.Get(keyObjectIDs).(*schema.Set).List() {
		objectID, err := uuid.Parse(objectID.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) > 0 {
		if err := sla.Wrap(client).AssignDomain(ctx, gqlsla.AssignDomainParams{
			DomainAssignType:          gqlsla.NoAssignment,
			ObjectIDs:                 objectIDs,
			ApplyToExistingSnapshots:  ptr(true),
			ApplyToNonPolicySnapshots: ptr(false),
		}); err != nil {
			return diag.FromErr(err)
		}
		if err := waitForUnassignment(ctx, client, domainID, objectIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

// Note, the SLA domain assignment resource is designed to only manage SLA
// domain assignments owned by the resource. An import on the other hand will
// take ownership of all SLA domain assignments for a domain.
func importSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	tflog.Trace(ctx, "importSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	domainID, err := uuid.Parse(d.Id())
	if err != nil {
		return nil, err
	}

	// Return a human-readable error message if the SLA domain doesn't exist.
	if _, err := sla.Wrap(client).DomainByID(ctx, domainID); err != nil {
		return nil, err
	}

	objects, err := sla.Wrap(client).DomainObjects(ctx, domainID, "")
	if err != nil {
		return nil, err
	}
	objectIDs := &schema.Set{F: schema.HashString}
	for _, object := range objects {
		objectIDs.Add(object.ID.String())
	}
	if err := d.Set(keyObjectIDs, &objectIDs); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func waitForAssignment(ctx context.Context, client *polaris.Client, domainID uuid.UUID, objectIDs []uuid.UUID) error {
	tflog.Debug(ctx, "waiting for SLA domain assignment")

	for {
		idSet := make(map[uuid.UUID]struct{}, len(objectIDs))
		for _, id := range objectIDs {
			idSet[id] = struct{}{}
		}

		objects, err := sla.Wrap(client).DomainObjects(ctx, domainID, "")
		if err != nil {
			return err
		}
		for _, object := range objects {
			delete(idSet, object.ID)
		}
		if len(idSet) == 0 {
			return nil
		}

		tflog.Debug(ctx, "waiting for SLA domain assignment", map[string]any{
			"remaining_objects": len(idSet),
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func waitForUnassignment(ctx context.Context, client *polaris.Client, domainID uuid.UUID, objectIDs []uuid.UUID) error {
	tflog.Debug(ctx, "waiting for SLA domain unassignment")

	for {
		idSet := make(map[uuid.UUID]struct{}, len(objectIDs))
		for _, id := range objectIDs {
			idSet[id] = struct{}{}
		}

		objects, err := sla.Wrap(client).DomainObjects(ctx, domainID, "")
		if err != nil {
			return err
		}
		for _, object := range objects {
			delete(idSet, object.ID)
		}
		n := len(objectIDs) - len(idSet)
		if len(idSet) == len(objectIDs) {
			return nil
		}

		tflog.Debug(ctx, "waiting for SLA domain unassignment", map[string]any{
			"remaining_objects": n,
		})
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
