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
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

const resourceRoleAssignmentDescription = `
The ´polaris_role_assignment´ resource is used to assign roles to users in RSC.
`

func resourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		CreateContext: createRoleAssignment,
		ReadContext:   readRoleAssignment,
		DeleteContext: deleteRoleAssignment,

		Description: description(resourceRoleAssignmentDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the user email and the role ID.",
			},
			keyRoleID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Role ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyUserEmail: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "User email address. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func createRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createRoleAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	roleID, err := uuid.Parse(d.Get(keyRoleID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get(keyUserEmail).(string)

	if err := access.Wrap(client).AssignRole(ctx, userEmail, roleID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(userEmail+roleID.String()))))

	readCustomRole(ctx, d, m)
	return nil
}

func readRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readRoleAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	roleID, err := uuid.Parse(d.Get(keyRoleID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get(keyUserEmail).(string)

	user, err := access.Wrap(client).User(ctx, userEmail)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if !user.HasRole(roleID) {
		d.Set(keyRoleID, "")
	}

	return nil
}

func deleteRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteRoleAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	roleID, err := uuid.Parse(d.Get(keyRoleID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get(keyUserEmail).(string)

	if err := access.Wrap(client).UnassignRole(ctx, userEmail, roleID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
