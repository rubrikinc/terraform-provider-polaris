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

const dataSourceRoleTemplateDescription = `
The ´polaris_role_template´ data source is used to access information about an
RSC role template. A role template is looked up using either the ID or the name.
`

var _ datasource.DataSource = &roleTemplateDataSource{}

type roleTemplateDataSource struct {
	client *client
}

type roleTemplateModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Permission     types.Set    `tfsdk:"permission"`
	RoleTemplateID types.String `tfsdk:"role_template_id"`
}

func newRoleTemplateDataSource() datasource.DataSource {
	return &roleTemplateDataSource{}
}

func (d *roleTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "roleTemplateDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyRoleTemplate
}

func (d *roleTemplateDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "roleTemplateDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceRoleTemplateDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Role template ID (UUID).",
			},
			keyDescription: schema.StringAttribute{
				Computed:    true,
				Description: "Role template description.",
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "Role template name.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("role_template_id")),
					isNotWhiteSpace(),
				},
			},
			keyRoleTemplateID: schema.StringAttribute{
				Optional:    true,
				Description: "Role template ID (UUID).",
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

func (d *roleTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "roleTemplateDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *roleTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "roleTemplateDataSource.Read")

	var config roleTemplateModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var roleTemplate gqlaccess.RoleTemplate
	if !config.RoleTemplateID.IsNull() {
		roleTemplateID, err := uuid.Parse(config.RoleTemplateID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid role template ID", err.Error())
			return
		}
		roleTemplate, err = access.Wrap(polarisClient).RoleTemplateByID(ctx, roleTemplateID)
		if err != nil {
			res.Diagnostics.AddError("Failed to read role template", err.Error())
			return
		}
	} else {
		roleTemplate, err = access.Wrap(polarisClient).RoleTemplateByName(ctx, config.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read role template", err.Error())
			return
		}
	}

	permissionSet, diags := fromPermissions(ctx, roleTemplate.AssignedPermissions)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := roleTemplateModel{
		ID:             types.StringValue(roleTemplate.ID.String()),
		Description:    types.StringValue(roleTemplate.Description),
		Name:           types.StringValue(roleTemplate.Name),
		Permission:     permissionSet,
		RoleTemplateID: types.StringValue(roleTemplate.ID.String()),
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
