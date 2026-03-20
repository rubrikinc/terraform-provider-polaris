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
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

var (
	_ resource.Resource                 = &roleAssignmentResource{}
	_ resource.ResourceWithImportState  = &roleAssignmentResource{}
	_ resource.ResourceWithUpgradeState = &roleAssignmentResource{}
)

type roleAssignmentResource struct {
	client *client
}

type roleAssignmentModel struct {
	ID         types.String `tfsdk:"id"`
	RoleID     types.String `tfsdk:"role_id"`
	RoleIDs    types.Set    `tfsdk:"role_ids"`
	SSOGroupID types.String `tfsdk:"sso_group_id"`
	UserEmail  types.String `tfsdk:"user_email"`
	UserID     types.String `tfsdk:"user_id"`
}

func newRoleAssignmentResource() resource.Resource {
	return &roleAssignmentResource{}
}

func (r *roleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyRoleAssignment
}

func (r *roleAssignmentResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceRoleAssignmentDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "User or SSO group ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyRoleID: schema.StringAttribute{
				Optional:           true,
				Description:        "Role ID (UUID). **Deprecated:** use `role_ids` instead.",
				DeprecationMessage: "use `role_ids` instead.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyRoleIDs)),
				},
			},
			keyRoleIDs: schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Role IDs (UUID).",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			keySSOGroupID: schema.StringAttribute{
				Optional:    true,
				Description: "SSO group ID. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot(keyUserEmail),
						path.MatchRoot(keyUserID),
					),
					isNotWhiteSpace(),
				},
			},
			keyUserEmail: schema.StringAttribute{
				Optional: true,
				Description: "User email address. Changing this forces a new resource to be created. " +
					"**Deprecated:** use `user_id` with the `polaris_user` data source instead.",
				DeprecationMessage: "use `user_id` with the `polaris_user` data source instead.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyUserID: schema.StringAttribute{
				Optional:    true,
				Description: "User ID. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
		},
		Version: 1,
	}
}

func (r *roleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *roleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Create")

	var plan roleAssignmentModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	roleIDs, diags := r.collectRoleIDs(ctx, plan)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	// Using user ID.
	if !plan.UserID.IsNull() {
		userID := plan.UserID.ValueString()
		if err := access.Wrap(polarisClient).AssignUserRoles(ctx, userID, roleIDs); err != nil {
			res.Diagnostics.AddError("Failed to assign user roles", err.Error())
			return
		}

		plan.ID = types.StringValue(userID)
		res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
		return
	}

	// Using group ID.
	if !plan.SSOGroupID.IsNull() {
		groupID := plan.SSOGroupID.ValueString()
		if err := access.Wrap(polarisClient).AssignSSOGroupRoles(ctx, groupID, roleIDs); err != nil {
			res.Diagnostics.AddError("Failed to assign SSO group roles", err.Error())
			return
		}

		plan.ID = types.StringValue(groupID)
		res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
		return
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(polarisClient).UserByEmail(ctx, plan.UserEmail.ValueString(), gqlaccess.DomainLocal)
	if err != nil {
		res.Diagnostics.AddError("Failed to look up user by email", err.Error())
		return
	}
	if err := access.Wrap(polarisClient).AssignUserRoles(ctx, user.ID, roleIDs); err != nil {
		res.Diagnostics.AddError("Failed to assign user roles", err.Error())
		return
	}

	plan.ID = types.StringValue(user.ID)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *roleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Read")

	var state roleAssignmentModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	// Using user ID.
	if !state.UserID.IsNull() {
		user, err := access.Wrap(polarisClient).UserByID(ctx, state.UserID.ValueString())
		if errors.Is(err, graphql.ErrNotFound) {
			res.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			res.Diagnostics.AddError("Failed to read user", err.Error())
			return
		}

		state.UserID = types.StringValue(user.ID)
		r.updateRoleState(ctx, &state, user.Roles, res)
		if res.Diagnostics.HasError() {
			return
		}

		res.Diagnostics.Append(res.State.Set(ctx, &state)...)
		return
	}

	// Using group ID.
	if !state.SSOGroupID.IsNull() {
		group, err := access.Wrap(polarisClient).SSOGroupByID(ctx, state.SSOGroupID.ValueString())
		if errors.Is(err, graphql.ErrNotFound) {
			res.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			res.Diagnostics.AddError("Failed to read SSO group", err.Error())
			return
		}

		state.SSOGroupID = types.StringValue(group.ID)
		r.updateRoleState(ctx, &state, group.Roles, res)
		if res.Diagnostics.HasError() {
			return
		}

		res.Diagnostics.Append(res.State.Set(ctx, &state)...)
		return
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(polarisClient).UserByEmail(ctx, state.UserEmail.ValueString(), gqlaccess.DomainLocal)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read user by email", err.Error())
		return
	}

	state.UserEmail = types.StringValue(user.Email)
	r.updateRoleState(ctx, &state, user.Roles, res)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *roleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Update")

	var plan roleAssignmentModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	var state roleAssignmentModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	newRoleIDs, diags := r.collectRoleIDs(ctx, plan)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	oldRoleIDs, diags := r.collectRoleIDs(ctx, state)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	addIDs, removeIDs := diffRoleIDSets(newRoleIDs, oldRoleIDs)

	// Using user ID.
	if !plan.UserID.IsNull() {
		userID := plan.UserID.ValueString()
		if len(removeIDs) > 0 {
			if err := access.Wrap(polarisClient).UnassignUserRoles(ctx, userID, removeIDs); err != nil {
				res.Diagnostics.AddError("Failed to unassign user roles", err.Error())
				return
			}
		}
		if len(addIDs) > 0 {
			if err := access.Wrap(polarisClient).AssignUserRoles(ctx, userID, addIDs); err != nil {
				res.Diagnostics.AddError("Failed to assign user roles", err.Error())
				return
			}
		}

		plan.ID = state.ID
		res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
		return
	}

	// Using group ID.
	if !plan.SSOGroupID.IsNull() {
		groupID := plan.SSOGroupID.ValueString()
		if len(removeIDs) > 0 {
			if err := access.Wrap(polarisClient).UnassignSSOGroupRoles(ctx, groupID, removeIDs); err != nil {
				res.Diagnostics.AddError("Failed to unassign SSO group roles", err.Error())
				return
			}
		}
		if len(addIDs) > 0 {
			if err := access.Wrap(polarisClient).AssignSSOGroupRoles(ctx, groupID, addIDs); err != nil {
				res.Diagnostics.AddError("Failed to assign SSO group roles", err.Error())
				return
			}
		}

		plan.ID = state.ID
		res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
		return
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(polarisClient).UserByEmail(ctx, plan.UserEmail.ValueString(), gqlaccess.DomainLocal)
	if err != nil {
		res.Diagnostics.AddError("Failed to look up user by email", err.Error())
		return
	}
	if len(removeIDs) > 0 {
		if err := access.Wrap(polarisClient).UnassignUserRoles(ctx, user.ID, removeIDs); err != nil {
			res.Diagnostics.AddError("Failed to unassign user roles", err.Error())
			return
		}
	}
	if len(addIDs) > 0 {
		if err := access.Wrap(polarisClient).AssignUserRoles(ctx, user.ID, addIDs); err != nil {
			res.Diagnostics.AddError("Failed to assign user roles", err.Error())
			return
		}
	}

	plan.ID = state.ID
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *roleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.Delete")

	var state roleAssignmentModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	roleIDs, diags := r.collectRoleIDs(ctx, state)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	// Using user ID.
	if !state.UserID.IsNull() {
		err := access.Wrap(polarisClient).UnassignUserRoles(ctx, state.UserID.ValueString(), roleIDs)
		if errors.Is(err, graphql.ErrNotFound) {
			return
		}
		if err != nil {
			res.Diagnostics.AddError("Failed to unassign user roles", err.Error())
		}
		return
	}

	// Using group ID.
	if !state.SSOGroupID.IsNull() {
		err := access.Wrap(polarisClient).UnassignSSOGroupRoles(ctx, state.SSOGroupID.ValueString(), roleIDs)
		if errors.Is(err, graphql.ErrNotFound) {
			return
		}
		if err != nil {
			res.Diagnostics.AddError("Failed to unassign SSO group roles", err.Error())
		}
		return
	}

	// Using user email. Deprecated, provided only for backwards compatibility.
	user, err := access.Wrap(polarisClient).UserByEmail(ctx, state.UserEmail.ValueString(), gqlaccess.DomainLocal)
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to look up user by email", err.Error())
		return
	}
	err = access.Wrap(polarisClient).UnassignUserRoles(ctx, user.ID, roleIDs)
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to unassign user roles", err.Error())
	}
}

// ImportState import roles assigned to a user or group.
//
// Note, the role assignment resource is designed to only manage role
// assignments owned by the resource. An import on the other hand will take
// ownership of all role assignments for a user or group.
func (r *roleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "roleAssignmentResource.ImportState")

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	// Using user ID.
	user, err := access.Wrap(polarisClient).UserByID(ctx, req.ID)
	if err == nil {
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), user.ID)...)
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyUserID), user.ID)...)

		roleIDs := make([]string, 0, len(user.Roles))
		for _, role := range user.Roles {
			roleIDs = append(roleIDs, role.ID.String())
		}
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyRoleIDs), roleIDs)...)
		return
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		res.Diagnostics.AddError("Failed to look up user", err.Error())
		return
	}

	// Using group ID.
	group, err := access.Wrap(polarisClient).SSOGroupByID(ctx, req.ID)
	if err == nil {
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), group.ID)...)
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keySSOGroupID), group.ID)...)

		roleIDs := make([]string, 0, len(group.Roles))
		for _, role := range group.Roles {
			roleIDs = append(roleIDs, role.ID.String())
		}
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyRoleIDs), roleIDs)...)
		return
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		res.Diagnostics.AddError("Failed to look up SSO group", err.Error())
		return
	}

	res.Diagnostics.AddError("Import failed", fmt.Sprintf("user or SSO group %q not found", req.ID))
}

// collectRoleIDs gathers role IDs from both the deprecated role_id field and
// the role_ids field.
func (r *roleAssignmentResource) collectRoleIDs(ctx context.Context, model roleAssignmentModel) ([]uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics

	var roleIDs []uuid.UUID
	if !model.RoleIDs.IsNull() {
		var ids []string
		diags.Append(model.RoleIDs.ElementsAs(ctx, &ids, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for _, id := range ids {
			parsed, err := uuid.Parse(id)
			if err != nil {
				diags.AddError("Invalid role ID", err.Error())
				return nil, diags
			}
			roleIDs = append(roleIDs, parsed)
		}
	}

	// Deprecated, provided only for backwards compatibility.
	if !model.RoleID.IsNull() && model.RoleID.ValueString() != "" {
		parsed, err := uuid.Parse(model.RoleID.ValueString())
		if err != nil {
			diags.AddError("Invalid role ID", err.Error())
			return nil, diags
		}
		roleIDs = append(roleIDs, parsed)
	}

	return roleIDs, diags
}

// updateRoleState reconciles the assigned roles with the state. It handles the
// deprecated role_id path and the role_ids path separately.
func (r *roleAssignmentResource) updateRoleState(ctx context.Context, state *roleAssignmentModel, roles []gqlaccess.RoleRef, res *resource.ReadResponse) {
	// Deprecated, provided only for backwards compatibility.
	if !state.RoleID.IsNull() && state.RoleID.ValueString() != "" {
		id, err := uuid.Parse(state.RoleID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid role ID", err.Error())
			return
		}

		var roleID string
		for _, role := range roles {
			if role.ID == id {
				roleID = id.String()
				break
			}
		}
		state.RoleID = types.StringValue(roleID)
		return
	}

	if !state.RoleIDs.IsNull() {
		var stateRoleIDs []string
		res.Diagnostics.Append(state.RoleIDs.ElementsAs(ctx, &stateRoleIDs, false)...)
		if res.Diagnostics.HasError() {
			return
		}

		// Build a set of role IDs that are actually assigned in RSC.
		assignedSet := make(map[string]struct{}, len(roles))
		for _, role := range roles {
			assignedSet[role.ID.String()] = struct{}{}
		}

		// Filter the role IDs in the state, keeping only role IDs that are
		// assigned.
		var reconciledIDs []string
		for _, id := range stateRoleIDs {
			if _, ok := assignedSet[id]; ok {
				reconciledIDs = append(reconciledIDs, id)
			}
		}

		reconciledSet, diags := types.SetValueFrom(ctx, types.StringType, reconciledIDs)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		state.RoleIDs = reconciledSet
	}
}

// diffRoleIDSets computes the delta between two UUID slices. I.e., the role IDs
// to add and remove given the changes to the role_ids resource data.
func diffRoleIDSets(newIDs, oldIDs []uuid.UUID) ([]uuid.UUID, []uuid.UUID) {
	newSet := make(map[uuid.UUID]struct{}, len(newIDs))
	for _, id := range newIDs {
		newSet[id] = struct{}{}
	}

	remove := make([]uuid.UUID, 0, len(oldIDs))
	for _, id := range oldIDs {
		if _, ok := newSet[id]; !ok {
			remove = append(remove, id)
		} else {
			delete(newSet, id)
		}
	}

	add := make([]uuid.UUID, 0, len(newSet))
	for id := range newSet {
		add = append(add, id)
	}

	return add, remove
}
