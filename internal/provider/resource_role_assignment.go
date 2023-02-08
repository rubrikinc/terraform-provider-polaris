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
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
)

// resourceRoleAssignment defines the schema for the role assignment resource.
func resourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		CreateContext: createRoleAssignment,
		ReadContext:   readRoleAssignment,
		DeleteContext: deleteRoleAssignment,

		Schema: map[string]*schema.Schema{
			"role_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Role identifier.",
				ValidateDiagFunc: validateStringIsNotWhiteSpace,
			},
			"user_email": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "User email address.",
				ValidateDiagFunc: validateStringIsNotWhiteSpace,
			},
		},
	}
}

// createRoleAssignment run the Create operation for the role assignment
// resource.
func createRoleAssignment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] createRoleAssignment")

	client := m.(*polaris.Client)

	roleID, err := uuid.Parse(d.Get("role_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get("user_email").(string)

	if err := access.Wrap(client).AssignRole(ctx, roleID, userEmail); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(userEmail+roleID.String()))))

	readCustomRole(ctx, d, m)
	return nil
}

// readRoleAssignment run the Read operation for the role assignment resource.
func readRoleAssignment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] readRoleAssignment")

	client := m.(*polaris.Client)

	roleID, err := uuid.Parse(d.Get("role_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get("user_email").(string)

	user, err := access.Wrap(client).User(ctx, userEmail)
	if err != nil {
		return diag.FromErr(err)
	}
	if !user.HasRole(roleID) {
		d.Set("role_id", "")
	}

	return nil
}

// deleteRoleAssignment run the Delete operation for the role assignment
// resource.
func deleteRoleAssignment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] deleteRoleAssignment")

	client := m.(*polaris.Client)

	roleID, err := uuid.Parse(d.Get("role_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	userEmail := d.Get("user_email").(string)

	if err := access.Wrap(client).UnassignRole(ctx, roleID, userEmail); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
