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
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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

const resourceCustomRoleDescription = `
The ´polaris_custom_role´ resource is used to create and manage custom roles in
RSC.
`

var (
	_ resource.Resource                = &customRoleResource{}
	_ resource.ResourceWithIdentity    = &customRoleResource{}
	_ resource.ResourceWithImportState = &customRoleResource{}
)

type customRoleResource struct {
	client *client
}

type customRoleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permission  types.Set    `tfsdk:"permission"`
}

type customRoleIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newCustomRoleResource() resource.Resource {
	return &customRoleResource{}
}

func (r *customRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "customRoleResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyCustomRole
}

func (r *customRoleResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "customRoleResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceCustomRoleDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Role ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyName: schema.StringAttribute{
				Required:    true,
				Description: "Role name.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyDescription: schema.StringAttribute{
				Optional:    true,
				Description: "Role description.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			keyPermission: schema.SetNestedBlock{
				Description: "Role permission.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyOperation: schema.StringAttribute{
							Required:    true,
							Description: "Operation to allow on object IDs under the snappable hierarchy.",
							Validators: []validator.String{
								isNotWhiteSpace(),
							},
						},
					},
					Blocks: map[string]schema.Block{
						keyHierarchy: schema.SetNestedBlock{
							Description: "Snappable hierarchy.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									keySnappableType: schema.StringAttribute{
										Required:    true,
										Description: "Snappable/workload type.",
										Validators: []validator.String{
											isNotWhiteSpace(),
										},
									},
									keyObjectIDs: schema.SetAttribute{
										ElementType: types.StringType,
										Required:    true,
										Description: "Object/workload identifiers.",
										Validators: []validator.Set{
											setvalidator.ValueStringsAre(isNotWhiteSpace()),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *customRoleResource) IdentitySchema(ctx context.Context, _ resource.IdentitySchemaRequest, res *resource.IdentitySchemaResponse) {
	tflog.Trace(ctx, "customRoleResource.IdentitySchema")

	res.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			keyID: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Role ID (UUID).",
			},
		},
	}
}

func (r *customRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "customRoleResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *customRoleResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "customRoleResource.Create")

	var plan customRoleModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	permissions, diags := toPermissions(ctx, plan.Permission)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	id, err := access.Wrap(polarisClient).CreateRole(ctx, plan.Name.ValueString(), plan.Description.ValueString(), permissions)
	if err != nil {
		res.Diagnostics.AddError("Failed to create custom role", err.Error())
		return
	}

	plan.ID = types.StringValue(id.String())
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)

	identity := customRoleIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *customRoleResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "customRoleResource.Read")

	var state customRoleModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid role ID", err.Error())
		return
	}

	role, err := access.Wrap(polarisClient).RoleByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read custom role", err.Error())
		return
	}

	state.Name = types.StringValue(role.Name)
	state.Description = types.StringValue(role.Description)

	permissionSet, diags := fromPermissions(ctx, role.AssignedPermissions)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	state.Permission = permissionSet

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)

	identity := customRoleIdentityModel{ID: state.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *customRoleResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "customRoleResource.Update")

	var plan customRoleModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	var state customRoleModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid role ID", err.Error())
		return
	}

	permissions, diags := toPermissions(ctx, plan.Permission)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	if err = access.Wrap(polarisClient).UpdateRole(ctx, id, plan.Name.ValueString(), plan.Description.ValueString(), permissions); err != nil {
		res.Diagnostics.AddError("Failed to update custom role", err.Error())
		return
	}

	plan.ID = state.ID
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)

	identity := customRoleIdentityModel{ID: plan.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *customRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "customRoleResource.Delete")

	var state customRoleModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid role ID", err.Error())
		return
	}

	err = access.Wrap(polarisClient).DeleteRole(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to delete custom role", err.Error())
		return
	}
}

func (r *customRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "customRoleResource.ImportState")

	resource.ImportStatePassthroughWithIdentity(ctx, path.Root(keyID), path.Root(keyID), req, res)
}
