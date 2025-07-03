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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceUserDescription = `
The ´polaris_user´ data source is used to access information about an RSC user.
Information for both local and SSO users can be accessed. A user is looked up
using either the ID or the email address.

-> **Note:** RSC allows the same email address to be used, at the same time, by
   both local and SSO users. Use the ´domain´ field to specify in which domain
   to look for a user.

-> **Note:** The ´status´ field will always be ´UNKNOWN´ for SSO users.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: userRead,

		Description: description(dataSourceUserDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User ID.",
			},
			keyDomain: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{keyUserID},
				Description: "The domain in which to look for a user when an email address is specified. Possible " +
					"values are `LOCAL` and `SSO`.",
				ValidateFunc: validation.StringInSlice([]string{"LOCAL", "SSO"}, false),
			},
			keyEmail: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyUserID},
				Description:  "User email address.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyIsAccountOwner: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the user is the account owner.",
			},
			keyRoles: {
				Type:        schema.TypeSet,
				Elem:        rscRoleResource(),
				Computed:    true,
				Description: "Roles assigned to the user.",
			},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User status.",
			},
			keyUserID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyEmail},
				Description:  "User ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func userRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "userRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var user gqlaccess.User
	if userID := d.Get(keyUserID).(string); userID != "" {
		user, err = access.Wrap(client).UserByID(ctx, userID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		user, err = access.Wrap(client).UserByEmail(ctx, d.Get(keyEmail).(string), gqlaccess.UserDomain(d.Get(keyDomain).(string)))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyDomain, user.Domain); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyEmail, user.Email); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyIsAccountOwner, user.IsAccountOwner); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStatus, user.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyUserID, user.ID); err != nil {
		return diag.FromErr(err)
	}

	roles := &schema.Set{F: schema.HashResource(rscRoleResource())}
	for _, role := range user.Roles {
		roles.Add(map[string]any{
			keyID:   role.ID.String(),
			keyName: role.Name,
		})
	}
	if err := d.Set(keyRoles, roles); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.ID)
	return nil
}
