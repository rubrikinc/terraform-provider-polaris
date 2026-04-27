// Copyright 2026 Rubrik, Inc.
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
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

const resourceSSOGroupDescription = `
The ´polaris_sso_group´ resource is used to create and manage SSO groups in RSC.
`

var (
	_ resource.Resource                = &ssoGroupResource{}
	_ resource.ResourceWithIdentity    = &ssoGroupResource{}
	_ resource.ResourceWithImportState = &ssoGroupResource{}
)

type ssoGroupResource struct {
	client *client
}

type ssoGroupResourceModel struct {
	ID           types.String `tfsdk:"id"`
	AuthDomainID types.String `tfsdk:"auth_domain_id"`
	DomainName   types.String `tfsdk:"domain_name"`
	GroupName    types.String `tfsdk:"group_name"`
	RoleIDs      types.Set    `tfsdk:"role_ids"`
}

type ssoGroupIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newSSOGroupResource() resource.Resource {
	return &ssoGroupResource{}
}

func (r *ssoGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keySSOGroup
}

func (r *ssoGroupResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceSSOGroupDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SSO group ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyAuthDomainID: schema.StringAttribute{
				Required: true,
				Description: "Auth domain ID (identity provider ID). Changing this forces a new " +
					"resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyDomainName: schema.StringAttribute{
				Computed:    true,
				Description: "The domain name of the SSO group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyGroupName: schema.StringAttribute{
				Required: true,
				Description: "SSO group name. Changing this forces a new " +
					"resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyRoleIDs: schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Roles assigned to the SSO group (UUIDs).",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(isUUID()),
				},
			},
		},
	}
}

func (r *ssoGroupResource) IdentitySchema(ctx context.Context, _ resource.IdentitySchemaRequest, res *resource.IdentitySchemaResponse) {
	tflog.Trace(ctx, "ssoGroupResource.IdentitySchema")

	res.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			keyID: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "SSO group ID.",
			},
		},
	}
}

func (r *ssoGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *ssoGroupResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Create")

	var plan ssoGroupResourceModel
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

	err = access.Wrap(polarisClient).CreateSSOGroup(
		ctx,
		plan.GroupName.ValueString(),
		roleIDs,
		plan.AuthDomainID.ValueString(),
	)
	if err != nil {
		res.Diagnostics.AddError("Failed to create SSO group", err.Error())
		return
	}

	// Read back the group by name and auth domain to get the ID and computed
	// fields.
	group, err := access.Wrap(polarisClient).SSOGroupByNameAndAuthDomain(
		ctx, plan.GroupName.ValueString(), plan.AuthDomainID.ValueString(),
	)
	if err != nil {
		res.Diagnostics.AddError("Failed to read SSO group after create", err.Error())
		return
	}

	plan.ID = types.StringValue(group.ID)
	plan.DomainName = types.StringValue(group.DomainName)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := ssoGroupIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *ssoGroupResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Read")

	var state ssoGroupResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	group, err := access.Wrap(polarisClient).SSOGroupByID(ctx, state.ID.ValueString())
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read SSO group", err.Error())
		return
	}

	roleIDs := make([]string, 0, len(group.Roles))
	for _, role := range group.Roles {
		roleIDs = append(roleIDs, role.ID.String())
	}
	roleIDsSet, diags := types.SetValueFrom(ctx, types.StringType, roleIDs)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(group.ID)
	state.DomainName = types.StringValue(group.DomainName)
	state.GroupName = types.StringValue(group.Name)
	state.RoleIDs = roleIDsSet
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := ssoGroupIdentityModel{ID: state.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *ssoGroupResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Update")

	var plan ssoGroupResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	var state ssoGroupResourceModel
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

	if err := access.Wrap(polarisClient).ReplaceSSOGroupRoles(ctx, state.ID.ValueString(), roleIDs); err != nil {
		res.Diagnostics.AddError("Failed to update SSO group roles", err.Error())
		return
	}

	// Save plan to state before read-back so the user-configured fields are
	// up-to-date even if the read-back fails.
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	group, err := access.Wrap(polarisClient).SSOGroupByID(ctx, state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddWarning("Failed to read SSO group after update",
			fmt.Sprintf("The SSO group was updated successfully but the computed fields could not be refreshed: %s", err.Error()))
		return
	}

	serverRoleIDs := make([]string, 0, len(group.Roles))
	for _, role := range group.Roles {
		serverRoleIDs = append(serverRoleIDs, role.ID.String())
	}
	roleIDsSet, diags := types.SetValueFrom(ctx, types.StringType, serverRoleIDs)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	plan.DomainName = types.StringValue(group.DomainName)
	plan.RoleIDs = roleIDsSet
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	identity := ssoGroupIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *ssoGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "ssoGroupResource.Delete")

	var state ssoGroupResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	err = access.Wrap(polarisClient).DeleteSSOGroup(ctx, state.ID.ValueString())
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to delete SSO group", err.Error())
	}
}

func (r *ssoGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "ssoGroupResource.ImportState")

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	// Import by identity block (Terraform 1.12+).
	if req.Identity != nil {
		var identity ssoGroupIdentityModel
		res.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if res.Diagnostics.HasError() {
			return
		}

		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), identity.ID.ValueString())...)
		res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
		return
	}

	// Import by raw UUID.
	if _, err := uuid.Parse(req.ID); err == nil {
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), req.ID)...)

		identity := ssoGroupIdentityModel{ID: types.StringValue(req.ID)}
		res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
		return
	}

	// Import by legacy composite format: "<group_name>:<identity_provider_id>".
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		res.Diagnostics.AddError("Invalid import ID",
			`Expected a UUID or the legacy format "<group_name>:<identity_provider_id>"`)
		return
	}
	groupName := parts[0]
	authDomainID := parts[1]

	if _, err := uuid.Parse(authDomainID); err != nil {
		res.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("The identity provider ID %q is not a valid UUID: %s", authDomainID, err))
		return
	}

	group, err := access.Wrap(polarisClient).SSOGroupByNameAndAuthDomain(ctx, groupName, authDomainID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read SSO group", err.Error())
		return
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), group.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyAuthDomainID), authDomainID)...)

	identity := ssoGroupIdentityModel{ID: types.StringValue(group.ID)}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

// collectRoleIDs extracts role UUIDs from the model RoleIDs set.
func (r *ssoGroupResource) collectRoleIDs(ctx context.Context, model ssoGroupResourceModel) ([]uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics

	var ids []string
	diags.Append(model.RoleIDs.ElementsAs(ctx, &ids, false)...)
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
