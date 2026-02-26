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
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const resourceCustomRoleDescription = `
The ´polaris_custom_role´ resource is used to create and manage custom roles in
RSC.
`

func resourceCustomRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: createCustomRole,
		ReadContext:   readCustomRole,
		UpdateContext: updateCustomRole,
		DeleteContext: deleteCustomRole,

		Description:   description(resourceCustomRoleDescription),
		CustomizeDiff: customizeDiffCustomRole,
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role ID (UUID).",
			},
			keyDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Role description.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Role name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermission: {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyHierarchy: {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyObjectIDs: {
										Type: schema.TypeSet,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringIsNotWhiteSpace,
										},
										Required:    true,
										MinItems:    1,
										Description: "Object/workload identifiers.",
									},
									keySnappableType: {
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
						keyOperation: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Operation to allow on object IDs under the snappable hierarchy.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
				Required:    true,
				MinItems:    1,
				Description: "Role permission.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// createCustomRole run the Create operation for the custom role resource.
func createCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "createCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get(keyName).(string)
	description := d.Get(keyDescription).(string)
	permissions := toPermissions(d.Get(keyPermission))

	id, err := access.Wrap(client).CreateRole(ctx, name, description, permissions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	readCustomRole(ctx, d, m)
	return nil
}

// readCustomRole run the Read operation for the custom role resource.
func readCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "readCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := access.Wrap(client).RoleByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyName, role.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, role.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPermission, fromPermissions(role.AssignedPermissions)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// updateCustomRole run the Update operation for the custom role resource.
func updateCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "updateCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges(keyName, keyDescription, keyPermission) {
		name := d.Get(keyName).(string)
		description := d.Get(keyDescription).(string)
		permissions := toPermissions(d.Get(keyPermission))

		if err := access.Wrap(client).UpdateRole(ctx, id, name, description, permissions); err != nil {
			return diag.FromErr(err)
		}
	}

	readCustomRole(ctx, d, m)
	return nil
}

// deleteCustomRole run the Delete operation for the custom role resource.
func deleteCustomRole(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "deleteCustomRole")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := access.Wrap(client).DeleteRole(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// customizeDiffCustomRole validates that no two permission blocks share the
// same operation. Because the permission set is hashed by operation, duplicate
// operations would cause one block to silently overwrite the other. Users
// should instead use multiple hierarchy blocks within a single permission
// block.
func customizeDiffCustomRole(ctx context.Context, diff *schema.ResourceDiff, m any) error {
	tflog.Trace(ctx, "customizeDiffCustomRole")

	permissions := diff.Get(keyPermission).(*schema.Set).List()
	seen := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		operation := p.(map[string]any)[keyOperation].(string)
		if _, ok := seen[operation]; ok {
			return fmt.Errorf(
				"duplicate permission blocks with operation %q: use multiple hierarchy blocks inside a single permission block instead",
				operation,
			)
		}
		seen[operation] = struct{}{}
	}

	return nil
}

func permissionHash(v any) int {
	return schema.HashString(v.(map[string]any)[keyOperation])
}

func fromPermissions(permissions []gqlaccess.Permission) any {
	permissionBlocks := &schema.Set{F: permissionHash}
	for _, permission := range permissions {
		permissionBlocks.Add(fromPermission(permission))
	}

	return permissionBlocks
}

func fromPermission(permission gqlaccess.Permission) any {
	hierarchyBlocks := &schema.Set{F: func(v any) int {
		return schema.HashString(v.(map[string]any)[keySnappableType])
	}}
	for _, hierarchy := range permission.ObjectsForHierarchyTypes {
		hierarchyBlocks.Add(fromSnappableHierarchy(hierarchy))
	}

	return map[string]any{
		keyOperation: permission.Operation,
		keyHierarchy: hierarchyBlocks,
	}
}

func fromSnappableHierarchy(hierarchy gqlaccess.ObjectsForHierarchyType) any {
	objectIDs := &schema.Set{F: schema.HashString}
	for _, objectID := range hierarchy.ObjectIDs {
		objectIDs.Add(objectID)
	}

	return map[string]any{
		keySnappableType: hierarchy.SnappableType,
		keyObjectIDs:     objectIDs,
	}
}

func toPermissions(permissionBlocks any) []gqlaccess.Permission {
	var permissions []gqlaccess.Permission
	for _, permissionBlock := range permissionBlocks.(*schema.Set).List() {
		permissions = append(permissions, toPermission(permissionBlock.(map[string]any)))
	}

	return permissions
}

func toPermission(permissionBlock map[string]any) gqlaccess.Permission {
	var hierarchies []gqlaccess.ObjectsForHierarchyType
	for _, hierarchy := range permissionBlock[keyHierarchy].(*schema.Set).List() {
		hierarchies = append(hierarchies, toSnappableHierarchy(hierarchy.(map[string]any)))
	}

	return gqlaccess.Permission{
		Operation:                permissionBlock[keyOperation].(string),
		ObjectsForHierarchyTypes: hierarchies,
	}
}

func toSnappableHierarchy(hierarchyBlock map[string]any) gqlaccess.ObjectsForHierarchyType {
	var objectIDs []string
	for _, objectID := range hierarchyBlock[keyObjectIDs].(*schema.Set).List() {
		objectIDs = append(objectIDs, objectID.(string))
	}

	return gqlaccess.ObjectsForHierarchyType{
		SnappableType: hierarchyBlock[keySnappableType].(string),
		ObjectIDs:     objectIDs,
	}
}
