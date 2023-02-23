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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
)

// resourceUser defines the schema for the user resource.
func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: createUser,
		ReadContext:   readUser,
		UpdateContext: updateUser,
		DeleteContext: deleteUser,

		Schema: map[string]*schema.Schema{
			"email": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "User email address.",
				ValidateDiagFunc: validateStringIsNotWhiteSpace,
			},
			"is_account_owner": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the user is the account owner.",
			},
			"role_ids": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateStringIsNotWhiteSpace,
				},
				Required:    true,
				Description: "Roles assigned to the user.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User status.",
			},
		},
	}
}

// createUser run the Create operation for the user resource.
func createUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createUser")

	userEmail := d.Get("email").(string)
	roleIDs, err := parseRoleIDs(d.Get("role_ids").(*schema.Set))
	if err != nil {
		return diag.FromErr(err)
	}

	client := m.(*polaris.Client)
	if err := access.Wrap(client).AddUser(ctx, userEmail, roleIDs); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(userEmail)

	readUser(ctx, d, m)
	return nil
}

// readUser run the Read operation for the user resource.
func readUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readUser")

	client := m.(*polaris.Client)
	user, err := access.Wrap(client).User(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("email", user.Email); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("is_account_owner", user.IsAccountOwner); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", user.Status); err != nil {
		return diag.FromErr(err)
	}

	roleIDs := &schema.Set{F: schema.HashString}
	for _, role := range user.Roles {
		roleIDs.Add(role.ID.String())
	}
	if err := d.Set("role_ids", roleIDs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// updateUser run the Update operation for the user resource.
func updateUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	roleIDs, err := parseRoleIDs(d.Get("role_ids").(*schema.Set))
	if err != nil {
		return diag.FromErr(err)
	}

	client := m.(*polaris.Client)
	if err := access.Wrap(client).ReplaceRoles(ctx, d.Id(), roleIDs); err != nil {
		return diag.FromErr(err)
	}

	readUser(ctx, d, m)
	return nil
}

// deleteUser run the Delete operation for the user resource.
func deleteUser(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteUser")

	client := m.(*polaris.Client)
	if err := access.Wrap(client).RemoveUser(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func parseRoleIDs(roleIDs *schema.Set) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, roleIDs.Len())
	for _, roleID := range roleIDs.List() {
		s, ok := roleID.(string)
		if !ok {
			return nil, fmt.Errorf("invalid role id: wrong type")
		}

		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid role id: %w", err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}
