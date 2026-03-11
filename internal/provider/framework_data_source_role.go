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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceRoleDescription = `
The ´polaris_role´ data source is used to access information about an RSC role.
A role is looked up using either the ID or the name.
`

var _ datasource.DataSource = &roleDataSource{}

type roleDataSource struct {
	client *client
}

type roleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsOrgAdmin  types.Bool   `tfsdk:"is_org_admin"`
	Permission  types.Set    `tfsdk:"permission"`
	RoleID      types.String `tfsdk:"role_id"`
}

func newRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

func (d *roleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "roleDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyRole
}

func (d *roleDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "roleDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceRoleDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Role ID (UUID).",
			},
			keyDescription: schema.StringAttribute{
				Computed:    true,
				Description: "Role description.",
			},
			keyIsOrgAdmin: schema.BoolAttribute{
				Computed:    true,
				Description: "True if the role is the organization administrator.",
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "Role name.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("role_id")),
				},
			},
			keyRoleID: schema.StringAttribute{
				Optional:    true,
				Description: "Role ID (UUID).",
				Validators: []validator.String{
					isUUID(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			keyPermission: schema.SetNestedBlock{
				Description: "Role permission.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyOperation: schema.StringAttribute{
							Computed:    true,
							Description: "Operation allowed on object IDs under the snappable hierarchy.",
						},
					},
					Blocks: map[string]schema.Block{
						keyHierarchy: schema.SetNestedBlock{
							Description: "Snappable hierarchy.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									keySnappableType: schema.StringAttribute{
										Computed:    true,
										Description: "Snappable/workload type.",
									},
									keyObjectIDs: schema.SetAttribute{
										ElementType: types.StringType,
										Computed:    true,
										Description: "Object/workload identifiers.",
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

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "roleDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "roleDataSource.Read")

	var config roleModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("Client error", err.Error())
		return
	}

	var role gqlaccess.Role
	if !config.RoleID.IsNull() {
		roleID, err := uuid.Parse(config.RoleID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid role ID", err.Error())
			return
		}
		role, err = access.Wrap(polarisClient).RoleByID(ctx, roleID)
		if err != nil {
			res.Diagnostics.AddError("Failed to read role", err.Error())
			return
		}
	} else {
		role, err = access.Wrap(polarisClient).RoleByName(ctx, config.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read role", err.Error())
			return
		}
	}

	permissionSet, diags := fromPermissions(ctx, role.AssignedPermissions)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := roleModel{
		ID:          types.StringValue(role.ID.String()),
		Description: types.StringValue(role.Description),
		IsOrgAdmin:  types.BoolValue(role.IsOrgAdmin),
		Name:        types.StringValue(role.Name),
		Permission:  permissionSet,
		RoleID:      types.StringValue(role.ID.String()),
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
