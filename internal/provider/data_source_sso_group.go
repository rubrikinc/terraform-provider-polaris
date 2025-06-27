// Copyright 2025 Rubrik, Inc.
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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceSSOGroupDescription = `
The ´polaris_sso_group´ data source is used to access information about an SSO
group in RSC. An SSO group is looked up using either the ID or the name.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceSSOGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: ssoGroupRead,

		Description: description(dataSourceSSOGroupDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SSO group ID.",
			},
			keyDomainName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The domain name of the SSO group.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keySSOGroupID},
				Description:  "SSO group name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRoles: {
				Type:        schema.TypeSet,
				Elem:        rscRoleResource(),
				Computed:    true,
				Description: "Roles assigned to the SSO group.",
			},
			keySSOGroupID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyName},
				Description:  "SSO group ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyUsers: {
				Type:        schema.TypeSet,
				Elem:        rscUserResource(),
				Computed:    true,
				Description: "Users in the SSO group.",
			},
		},
	}
}

func ssoGroupRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] ssoGroupRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var group gqlaccess.SSOGroup
	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		group, err = access.Wrap(client).SSOGroupByID(ctx, groupID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		group, err = access.Wrap(client).SSOGroupByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyDomainName, group.DomainName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, group.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySSOGroupID, group.ID); err != nil {
		return diag.FromErr(err)
	}

	roles := &schema.Set{F: schema.HashResource(rscRoleResource())}
	for _, role := range group.Roles {
		roles.Add(map[string]any{
			keyID:   role.ID.String(),
			keyName: role.Name,
		})
	}
	if err := d.Set(keyRoles, roles); err != nil {
		return diag.FromErr(err)
	}

	users := &schema.Set{F: schema.HashResource(rscUserResource())}
	for _, user := range group.Users {
		users.Add(map[string]any{
			keyID:    user.ID,
			keyEmail: user.Email,
		})
	}
	if err := d.Set(keyUsers, users); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(group.ID)
	return nil
}

func rscRoleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role ID (UUID).",
			},
			keyName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role name.",
			},
		},
	}
}

func rscUserResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User ID.",
			},
			keyEmail: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User email address.",
			},
		},
	}
}
