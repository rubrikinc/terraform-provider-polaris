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

const resourceUserDescription = `
The ´polaris_user´ resource is used to create and manage local users in RSC.
`

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: createUser,
		ReadContext:   readUser,
		UpdateContext: updateUser,
		DeleteContext: deleteUser,

		Description: description(resourceUserDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User ID (UUID).",
			},
			keyDomain: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User domain. Possible values are `LOCAL` and `SSO`.",
			},
			keyEmail: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "User email address. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyIsAccountOwner: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the user is the account owner.",
			},
			keyRoleIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
				Required:    true,
				Description: "Roles assigned to the user (UUIDs).",
			},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User status.",
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceUserV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceUserStateUpgradeV0,
			Version: 0,
		}},
	}
}

func createUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createUser")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	userEmail := d.Get(keyEmail).(string)

	var roleIDs []uuid.UUID
	for _, roleID := range d.Get(keyRoleIDs).(*schema.Set).List() {
		roleID, err := uuid.Parse(roleID.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	id, err := access.Wrap(client).CreateUser(ctx, userEmail, roleIDs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)
	readUser(ctx, d, m)
	return nil
}

func readUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readUser")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	user, err := access.Wrap(client).UserByID(ctx, d.Id())
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
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

	roleIDs := &schema.Set{F: schema.HashString}
	for _, role := range user.Roles {
		roleIDs.Add(role.ID.String())
	}
	if err := d.Set(keyRoleIDs, roleIDs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] updateUser")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var roleIDs []uuid.UUID
	for _, roleID := range d.Get(keyRoleIDs).(*schema.Set).List() {
		roleID, err := uuid.Parse(roleID.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	if err := access.Wrap(client).ReplaceUserRoles(ctx, d.Id(), roleIDs); err != nil {
		return diag.FromErr(err)
	}

	readUser(ctx, d, m)
	return nil
}

func deleteUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteUser")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if err := access.Wrap(client).DeleteUser(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
