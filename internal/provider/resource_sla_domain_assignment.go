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
	"log"
	"time"

	"github.com/google/uuid"
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
				ForceNew:     true,
				Description:  "SLA domain ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
		},
	}
}

func createSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	slaID, err := uuid.Parse(d.Get(keySLADomainID).(string))
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

	if err := sla.Wrap(client).AssignSLADomain(ctx, gqlsla.AssignSLAParams{
		SLAID:               &slaID,
		SLADomainAssignType: gqlsla.ProtectWithSLA,
		ObjectIDs:           objectIDs,
	}); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForSLADomainAssignment(ctx, client, slaID, objectIDs); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(slaID.String())
	return nil
}

func readSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	slaID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Make sure the SLA domain still exists.
	_, err = sla.Wrap(client).GlobalSLADomainByID(ctx, slaID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySLADomainID, slaID.String()); err != nil {
		return diag.FromErr(err)
	}

	objects, err := sla.Wrap(client).GlobalSLADomainProtectedObjects(ctx, slaID, "")
	if err != nil {
		return diag.FromErr(err)
	}

	objectIDs := schema.Set{F: schema.HashString}
	for _, object := range objects {
		objectIDs.Add(object.ID.String())
	}
	if err := d.Set(keyObjectIDs, &objectIDs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] updateSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	slaID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(keyObjectIDs) {
		addObjectIDs, removeObjectIDs, totalObjectIDs, err := diffObjectIDs(d)
		if err != nil {
			return diag.FromErr(err)
		}

		if len(addObjectIDs) > 0 {
			if err := sla.Wrap(client).AssignSLADomain(ctx, gqlsla.AssignSLAParams{
				SLAID:               &slaID,
				SLADomainAssignType: gqlsla.ProtectWithSLA,
				ObjectIDs:           addObjectIDs,
			}); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(removeObjectIDs) > 0 {
			if err := sla.Wrap(client).AssignSLADomain(ctx, gqlsla.AssignSLAParams{
				SLADomainAssignType: gqlsla.NoAssignment,
				ObjectIDs:           removeObjectIDs,
			}); err != nil {
				return diag.FromErr(err)
			}
		}

		if err := waitForSLADomainAssignment(ctx, client, slaID, totalObjectIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func deleteSLADomainAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteSLADomainAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	slaID, err := uuid.Parse(d.Id())
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
		if err := sla.Wrap(client).AssignSLADomain(ctx, gqlsla.AssignSLAParams{
			SLADomainAssignType: gqlsla.NoAssignment,
			ObjectIDs:           objectIDs,
		}); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := waitForSLADomainAssignment(ctx, client, slaID, []uuid.UUID{}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// waitForSLADomainAssignment
func waitForSLADomainAssignment(ctx context.Context, client *polaris.Client, slaID uuid.UUID, objectIDs []uuid.UUID) error {
	log.Print("[DEBUG] waiting for SLA domain assigment")

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		objectIDSet := make(map[uuid.UUID]struct{}, len(objectIDs))
		for _, id := range objectIDs {
			objectIDSet[id] = struct{}{}
		}

		objects, err := sla.Wrap(client).GlobalSLADomainProtectedObjects(waitCtx, slaID, "")
		if errors.Is(err, context.DeadlineExceeded) {
			log.Print("[WARN] abort waiting for SLA domain assignment")
			return nil
		}
		if err != nil {
			return err
		}

		for _, object := range objects {
			if _, ok := objectIDSet[object.ID]; !ok {
				objectIDSet[object.ID] = struct{}{}
			} else {
				delete(objectIDSet, object.ID)
			}
		}
		if len(objectIDSet) == 0 {
			return nil
		}

		log.Printf("[DEBUG] waiting for SLA domain assignment of %d objects", len(objectIDSet))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-waitCtx.Done():
		case <-time.After(5 * time.Second):
		}
	}
}

// diffObjectIDs returns the object IDs to add and remove.
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

	// Total object IDs are the union of object IDs to keep and new object IDs.
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
