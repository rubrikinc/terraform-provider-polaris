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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceRoleDescription = `
The ´polaris_role´ data source is used to access information about an RSC role.
A role is looked up using either the ID or the name.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: roleRead,

		Description: description(dataSourceRoleDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role ID (UUID).",
			},
			keyDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role description.",
			},
			keyIsOrgAdmin: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the role is the organization administrator.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyRoleID},
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
											Type: schema.TypeString,
										},
										Computed:    true,
										Description: "Object/workload identifiers.",
									},
									keySnappableType: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Snappable/workload type.",
									},
								},
							},
							Computed:    true,
							Description: "Snappable hierarchy.",
						},
						keyOperation: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operation allowed on object IDs under the snappable hierarchy.",
						},
					},
				},
				Computed:    true,
				Description: "Role permission.",
			},
			keyRoleID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyName},
				Description:  "Role ID.",
				ValidateFunc: validation.IsUUID,
			},
		},
	}
}

func roleRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] roleRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var role gqlaccess.Role
	if roleID := d.Get(keyRoleID).(string); roleID != "" {
		roleID, err := uuid.Parse(roleID)
		if err != nil {
			return diag.FromErr(err)
		}
		role, err = access.Wrap(client).RoleByID(ctx, roleID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		role, err = access.Wrap(client).RoleByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyDescription, role.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyIsOrgAdmin, role.IsOrgAdmin); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, role.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPermission, fromPermissions(role.AssignedPermissions)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRoleID, role.ID.String()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(role.ID.String())
	return nil
}
