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
The ´polaris_role_assignment´ resource is used to assign a role in RSC.
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
				Description: "SHA-256 hash of the user ID / SSO group ID and the role ID.",
			},
			keyRoleID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Role ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keySSOGroupID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyUserEmail, keyUserID},
				Description:  "SSO group ID. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyUserEmail: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keySSOGroupID, keyUserID},
				Description:  "User email address. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyUserID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keySSOGroupID, keyUserEmail},
				Description:  "User ID. Changing this forces a new resource to be created.",
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

	if userID := d.Get(keyUserID).(string); userID != "" {
		if err := access.Wrap(client).AssignUserRole(ctx, userID, roleID); err != nil {
			return diag.FromErr(err)
		}

		d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(userID+roleID.String()))))
		return nil
	}

	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		if err := access.Wrap(client).AssignSSOGroupRole(ctx, groupID, roleID); err != nil {
			return diag.FromErr(err)
		}

		d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(groupID+roleID.String()))))
		return nil
	}

	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := access.Wrap(client).AssignUserRole(ctx, user.ID, roleID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(user.ID+roleID.String()))))
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

	if userID := d.Get(keyUserID).(string); userID != "" {
		user, err := access.Wrap(client).UserByID(ctx, userID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyUserID, user.ID); err != nil {
			return diag.FromErr(err)
		}
		if !user.HasRole(roleID) {
			d.SetId("")
		}

		return nil
	}

	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		group, err := access.Wrap(client).SSOGroupByID(ctx, groupID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keySSOGroupID, group.ID); err != nil {
			return diag.FromErr(err)
		}
		if !group.HasRole(roleID) {
			d.SetId("")
		}

		return nil
	}

	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string))
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyUserEmail, user.Email); err != nil {
		return diag.FromErr(err)
	}
	if !user.HasRole(roleID) {
		d.SetId("")
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

	if userID := d.Get(keyUserID).(string); userID != "" {
		err := access.Wrap(client).UnassignUserRole(ctx, userID, roleID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		err := access.Wrap(client).UnassignSSOGroupRole(ctx, groupID, roleID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string))
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	err = access.Wrap(client).UnassignUserRole(ctx, user.ID, roleID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
