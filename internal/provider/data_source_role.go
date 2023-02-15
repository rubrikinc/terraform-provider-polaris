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
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
)

// dataSourceRole defines the schema for the role data source.
func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: roleRead,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role description.",
			},
			"is_org_admin": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the role is the organization administrator.",
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Role name.",
				ValidateDiagFunc: validateStringIsNotWhiteSpace,
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
											Type: schema.TypeString,
										},
										Computed:    true,
										Description: "Object/workload identifiers.",
									},
									"snappable_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Snappable/workload type.",
									},
								},
							},
							Computed:    true,
							Description: "Snappable hierarchy.",
						},
						"operation": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operation allowed on object ids under the snappable hierarchy.",
						},
					},
				},
				Computed:    true,
				Description: "Role permission.",
			},
		},
	}
}

// roleRead run the Read operation for the role data source.
func roleRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] roleRead")

	client := m.(*polaris.Client)

	name := d.Get("name").(string)
	roles, err := access.Wrap(client).Roles(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}
	role, err := findNamedRole(roles, name)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("description", role.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("is_org_admin", role.IsOrgAdmin); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("permission", fromPermissions(role.AssignedPermissions)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(role.ID.String())
	return nil
}

// findNamedRole returns the role with a name exactly matching the specified
// name.
func findNamedRole(roles []access.Role, name string) (access.Role, error) {
	if len(roles) == 0 {
		return access.Role{}, fmt.Errorf("no role named %q found", name)
	}

	for _, role := range roles {
		if role.Name == name {
			return role, nil
		}
	}

	return access.Role{}, fmt.Errorf("no role named exactly %q found", name)
}
