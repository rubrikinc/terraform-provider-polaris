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
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

const frameworkResourceUserDescription = `
The ´polaris_user´ resource is used to create and manage local users in RSC.
`

var (
	_ resource.Resource                 = &userResource{}
	_ resource.ResourceWithIdentity     = &userResource{}
	_ resource.ResourceWithImportState  = &userResource{}
	_ resource.ResourceWithUpgradeState = &userResource{}
)

type userResource struct {
	client *client
}

type userResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	Email          types.String `tfsdk:"email"`
	IsAccountOwner types.Bool   `tfsdk:"is_account_owner"`
	RoleIDs        types.Set    `tfsdk:"role_ids"`
	Status         types.String `tfsdk:"status"`
}

type userIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "userResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyUser
}

func (r *userResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "userResource.Schema")

	res.Schema = schema.Schema{
		Description: description(frameworkResourceUserDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "User ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyDomain: schema.StringAttribute{
				Computed:    true,
				Description: "User domain. Possible values are `LOCAL` and `SSO`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyEmail: schema.StringAttribute{
				Required: true,
				Description: "User email address. Note, all letters must be lower case. Changing this forces a new " +
					"resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[^A-Z]*$`), "letters must be lower case"),
				},
			},
			keyIsAccountOwner: schema.BoolAttribute{
				Computed:    true,
				Description: "True if the user is the account owner.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			keyRoleIDs: schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Roles assigned to the user (UUIDs).",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			keyStatus: schema.StringAttribute{
				Computed:    true,
				Description: "User status.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Version: 1,
	}
}

func (r *userResource) IdentitySchema(ctx context.Context, _ resource.IdentitySchemaRequest, res *resource.IdentitySchemaResponse) {
	tflog.Trace(ctx, "userResource.IdentitySchema")

	res.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			keyID: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "User ID (UUID).",
			},
		},
	}
}

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "userResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "userResource.Create")

	var plan userResourceModel
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

	id, err := access.Wrap(polarisClient).CreateUser(ctx, plan.Email.ValueString(), roleIDs)
	if err != nil {
		res.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	// Save ID to state before read-back so Terraform can track the resource
	// even if the read fails.
	plan.ID = types.StringValue(id)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)

	user, err := access.Wrap(polarisClient).UserByID(ctx, id)
	if err != nil {
		res.Diagnostics.AddWarning("Failed to read user after create",
			fmt.Sprintf("The user was created successfully but the computed fields could not be populated: %s", err.Error()))
		return
	}

	// Update the state with computed attributes read from the API.
	plan.Domain = types.StringValue(string(user.Domain))
	plan.IsAccountOwner = types.BoolValue(user.IsAccountOwner)
	plan.Status = types.StringValue(user.Status)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := userIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "userResource.Read")

	var state userResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	user, err := access.Wrap(polarisClient).UserByID(ctx, state.ID.ValueString())
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}

	roleIDs := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID.String())
	}
	roleIDsSet, diags := types.SetValueFrom(ctx, types.StringType, roleIDs)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(user.ID)
	state.Domain = types.StringValue(string(user.Domain))
	state.Email = types.StringValue(user.Email)
	state.IsAccountOwner = types.BoolValue(user.IsAccountOwner)
	state.Status = types.StringValue(user.Status)
	state.RoleIDs = roleIDsSet
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := userIdentityModel{ID: state.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "userResource.Update")

	var plan userResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	var state userResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	if err := access.Wrap(polarisClient).ReplaceUserRoles(ctx, state.ID.ValueString(), roleIDs); err != nil {
		res.Diagnostics.AddError("Failed to update user roles", err.Error())
		return
	}

	// Save plan to state before read-back so the user-configured fields are
	// up-to-date even if the read-back fails.
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)

	user, err := access.Wrap(polarisClient).UserByID(ctx, state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddWarning("Failed to read user after update",
			fmt.Sprintf("The user was updated successfully but the computed fields could not be refreshed: %s", err.Error()))
		return
	}

	// Update the state with computed attributes read from the API.
	plan.Domain = types.StringValue(string(user.Domain))
	plan.IsAccountOwner = types.BoolValue(user.IsAccountOwner)
	plan.Status = types.StringValue(user.Status)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := userIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "userResource.Delete")

	var state userResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	err = access.Wrap(polarisClient).DeleteUser(ctx, state.ID.ValueString())
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to delete user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "userResource.ImportState")

	resource.ImportStatePassthroughWithIdentity(ctx, path.Root(keyID), path.Root(keyID), req, res)
}

// collectRoleIDs extracts role UUIDs from the model RoleIDs set.
func (r *userResource) collectRoleIDs(ctx context.Context, userModel userResourceModel) ([]uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics

	var ids []string
	diags.Append(userModel.RoleIDs.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}

	roleIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			diags.AddError("Invalid role ID", err.Error())
			return nil, diags
		}

		roleIDs = append(roleIDs, parsed)
	}

	return roleIDs, diags
}
