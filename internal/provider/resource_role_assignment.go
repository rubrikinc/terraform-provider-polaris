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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const resourceRoleAssignmentDescription = `
The ´polaris_role_assignment´ resource is used to assign roles to a user or SSO
group in RSC.

~> **Warning:** When using multiple ´polaris_role_assignment´ resources to
   assign roles to the same user or SSO group, there is a risk for a race
   condition when the resources are destroyed. This can result in RSC roles
   still being assigned to the user or SSO group. The race condition can be
   avoided by either assigning all roles to the user using a single
   ´polaris_role_assignment´ resource or by using the ´depends_on´ field to make
   sure that the resources are destroyed in a serial fashion.
`

func resourceRoleAssignment() *schema.Resource {
	return &schema.Resource{
		CreateContext: createRoleAssignment,
		ReadContext:   readRoleAssignment,
		UpdateContext: updateRoleAssignment,
		DeleteContext: deleteRoleAssignment,

		Description: description(resourceRoleAssignmentDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User ID.",
			},
			keyRoleID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyRoleIDs},
				Description:  "Role ID (UUID). **Deprecated:** use `role_ids` instead.",
				Deprecated:   "use `role_ids` instead.",
				ValidateFunc: validation.IsUUID,
			},
			keyRoleIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
				Optional:     true,
				ExactlyOneOf: []string{keyRoleID},
				MinItems:     1,
				Description:  "Role IDs (UUID).",
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
				Description: "User email address. Changing this forces a new resource to be created. **Deprecated:** " +
					"use `user_id` with the `polaris_user` data source instead.",
				Deprecated:   "use `user_id` with the `polaris_user` data source instead.",
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
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceRoleAssignmentV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceRoleAssignmentStateUpgradeV0,
			Version: 0,
		}},
	}
}

func createRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "createRoleAssignment")

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
	if roleID := d.Get(keyRoleID).(string); roleID != "" {
		roleID, err := uuid.Parse(roleID)
		if err != nil {
			return diag.FromErr(err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	// Using user ID.
	if userID := d.Get(keyUserID).(string); userID != "" {
		if err := access.Wrap(client).AssignUserRoles(ctx, userID, roleIDs); err != nil {
			return diag.FromErr(err)
		}

		d.SetId(userID)
		return nil
	}

	// Using group ID.
	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		if err := access.Wrap(client).AssignSSOGroupRoles(ctx, groupID, roleIDs); err != nil {
			return diag.FromErr(err)
		}

		d.SetId(groupID)
		return nil
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string), gqlaccess.DomainLocal)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := access.Wrap(client).AssignUserRoles(ctx, user.ID, roleIDs); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.ID)
	return nil
}

func readRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "readRoleAssignment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Using user ID.
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
		if err := setRoleIDs(d, user.Roles); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	// Using group ID.
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
		if err := setRoleIDs(d, group.Roles); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string), gqlaccess.DomainLocal)
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
	if err := setRoleIDs(d, user.Roles); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "updateRoleAssignment")

	if !d.HasChanges(keyRoleID, keyRoleIDs) {
		return nil
	}

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	addRoleIDs, removeRoleIDs, err := diffRoleIDs(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Using user ID.
	if userID := d.Get(keyUserID).(string); userID != "" {
		if len(removeRoleIDs) > 0 {
			err := access.Wrap(client).UnassignUserRoles(ctx, userID, removeRoleIDs)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		if len(addRoleIDs) > 0 {
			err := access.Wrap(client).AssignUserRoles(ctx, userID, addRoleIDs)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return nil
	}

	// Using group ID.
	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		if len(removeRoleIDs) > 0 {
			err := access.Wrap(client).UnassignSSOGroupRoles(ctx, groupID, removeRoleIDs)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		if len(addRoleIDs) > 0 {
			err := access.Wrap(client).AssignSSOGroupRoles(ctx, groupID, addRoleIDs)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return nil
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string), gqlaccess.DomainLocal)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(removeRoleIDs) > 0 {
		err := access.Wrap(client).UnassignUserRoles(ctx, user.ID, removeRoleIDs)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if len(addRoleIDs) > 0 {
		err := access.Wrap(client).AssignUserRoles(ctx, user.ID, addRoleIDs)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func deleteRoleAssignment(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "deleteRoleAssignment")

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

	// Deprecated, provided only for backwards compatibility.
	if roleID := d.Get(keyRoleID).(string); roleID != "" {
		roleID, err := uuid.Parse(roleID)
		if err != nil {
			return diag.FromErr(err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	// Using user ID.
	if userID := d.Get(keyUserID).(string); userID != "" {
		err := access.Wrap(client).UnassignUserRoles(ctx, userID, roleIDs)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	// Using group ID.
	if groupID := d.Get(keySSOGroupID).(string); groupID != "" {
		err := access.Wrap(client).UnassignSSOGroupRoles(ctx, groupID, roleIDs)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(client).UserByEmail(ctx, d.Get(keyUserEmail).(string), gqlaccess.DomainLocal)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	err = access.Wrap(client).UnassignUserRoles(ctx, user.ID, roleIDs)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// diffRoleIDs returns the role IDs to add and remove given the changes to the
// resource data.
func diffRoleIDs(d *schema.ResourceData) ([]uuid.UUID, []uuid.UUID, error) {
	oldRoleIDs, newRoleIDs := d.GetChange(keyRoleIDs)

	// Deprecated, provided only for backwards compatibility.
	oldRoleID, newRoleID := d.GetChange(keyRoleID)
	if roleID := oldRoleID.(string); roleID != "" {
		oldRoleIDs.(*schema.Set).Add(roleID)
	}
	if roleID := newRoleID.(string); roleID != "" {
		newRoleIDs.(*schema.Set).Add(roleID)
	}

	addSet := make(map[uuid.UUID]struct{}, newRoleIDs.(*schema.Set).Len())
	for _, id := range newRoleIDs.(*schema.Set).List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return nil, nil, err
		}
		addSet[id] = struct{}{}
	}

	removeRoleIDs := make([]uuid.UUID, 0, oldRoleIDs.(*schema.Set).Len())
	for _, id := range oldRoleIDs.(*schema.Set).List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return nil, nil, err
		}
		if _, ok := addSet[id]; !ok {
			removeRoleIDs = append(removeRoleIDs, id)
		} else {
			delete(addSet, id)
		}
	}

	addRoleIDs := make([]uuid.UUID, 0, len(addSet))
	for id := range addSet {
		addRoleIDs = append(addRoleIDs, id)
	}

	return addRoleIDs, removeRoleIDs, nil
}

// setRoleIDs sets the role IDs in the resource data.
func setRoleIDs(d *schema.ResourceData, roles []gqlaccess.RoleRef) error {
	// Deprecated, provided only for backwards compatibility.
	if id := d.Get(keyRoleID).(string); id != "" {
		id, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		var roleID string
		for _, role := range roles {
			if role.ID == id {
				roleID = id.String()
				break
			}
		}
		if err := d.Set(keyRoleID, roleID); err != nil {
			return err
		}

		return nil
	}

	roleIDs := d.Get(keyRoleIDs).(*schema.Set)
	set := make(map[uuid.UUID]struct{}, roleIDs.Len())
	for _, id := range roleIDs.List() {
		id, err := uuid.Parse(id.(string))
		if err != nil {
			return err
		}
		set[id] = struct{}{}
	}
	for _, role := range roles {
		delete(set, role.ID)
	}
	for id := range set {
		roleIDs.Remove(id.String())
	}
	if err := d.Set(keyRoleIDs, roleIDs); err != nil {
		return err
	}

	return nil
}
