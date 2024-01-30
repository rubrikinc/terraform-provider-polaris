// Copyright 2023 Rubrik, Inc.
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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

// resourceCustomRole defines the schema for the custom role resource.
func resourceCustomRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: createCustomRole,
		ReadContext:   readCustomRole,
		UpdateContext: updateCustomRole,
		DeleteContext: deleteCustomRole,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Role description.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Role name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hierarchy": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"object_ids": {
										Type: schema.TypeSet,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringIsNotWhiteSpace,
										},
										Required:    true,
										MinItems:    1,
										Description: "Object/workload identifiers.",
									},
									"snappable_type": {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "Snappable/workload type.",
										ValidateFunc: validation.StringIsNotWhiteSpace,
									},
								},
							},
							Required:    true,
							MinItems:    1,
							Description: "Snappable hierarchy.",
						},
						"operation": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Operation to allow on object ids under the snappable hierarchy.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
				Required:    true,
				MinItems:    1,
				Description: "Role permission.",
			},
		},
	}
}

// createCustomRole run the Create operation for the custom role resource.
func createCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	permissions := toPermissions(d.Get("permission"))

	id, err := access.Wrap(client).AddRole(ctx, name, description, permissions, access.NoProtectableClusters)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())

	readCustomRole(ctx, d, m)
	return nil
}

// readCustomRole run the Read operation for the custom role resource.
func readCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := access.Wrap(client).Role(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", role.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", role.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("permission", fromPermissions(role.AssignedPermissions)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// updateCustomRole run the Update operation for the custom role resource.
func updateCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] updateCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "description", "permission") {
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		permissions := toPermissions(d.Get("permission"))

		if err := access.Wrap(client).UpdateRole(ctx, id, name, description, permissions, access.NoProtectableClusters); err != nil {
			return diag.FromErr(err)
		}
	}

	readCustomRole(ctx, d, m)
	return nil
}

// deleteCustomRole run the Delete operation for the custom role resource.
func deleteCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := access.Wrap(client).RemoveRole(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func permissionHash(v any) int {
	return schema.HashString(v.(map[string]any)["operation"])
}

func fromPermissions(permissions []access.Permission) any {
	permissionBlocks := &schema.Set{F: permissionHash}
	for _, permission := range permissions {
		permissionBlocks.Add(fromPermission(permission))
	}

	return permissionBlocks
}

func fromPermission(permission access.Permission) any {
	hierarchyBlocks := &schema.Set{F: func(v any) int {
		return schema.HashString(v.(map[string]any)["snappable_type"])
	}}
	for _, hierarchy := range permission.Hierarchies {
		hierarchyBlocks.Add(fromSnappableHierarchy(hierarchy))
	}

	return map[string]any{
		"operation": permission.Operation,
		"hierarchy": hierarchyBlocks,
	}
}

func fromSnappableHierarchy(hierarchy access.SnappableHierarchy) any {
	objectIDs := &schema.Set{F: schema.HashString}
	for _, objectID := range hierarchy.ObjectIDs {
		objectIDs.Add(objectID)
	}

	return map[string]any{
		"snappable_type": hierarchy.SnappableType,
		"object_ids":     objectIDs,
	}
}

func toPermissions(permissionBlocks any) []access.Permission {
	var permissions []access.Permission
	for _, permissionBlock := range permissionBlocks.(*schema.Set).List() {
		permissions = append(permissions, toPermission(permissionBlock.(map[string]any)))
	}

	return permissions
}

func toPermission(permissionBlock map[string]any) access.Permission {
	var hierarchies []access.SnappableHierarchy
	for _, hierarchy := range permissionBlock["hierarchy"].(*schema.Set).List() {
		hierarchies = append(hierarchies, toSnappableHierarchy(hierarchy.(map[string]any)))
	}

	return access.Permission{
		Operation:   permissionBlock["operation"].(string),
		Hierarchies: hierarchies,
	}
}

func toSnappableHierarchy(hierarchyBlock map[string]any) access.SnappableHierarchy {
	var objectIDs []string
	for _, objectID := range hierarchyBlock["object_ids"].(*schema.Set).List() {
		objectIDs = append(objectIDs, objectID.(string))
	}

	return access.SnappableHierarchy{
		SnappableType: hierarchyBlock["snappable_type"].(string),
		ObjectIDs:     objectIDs,
	}
}
