// Copyright 2025 Rubrik, Inc.
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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceSSOGroupDescription = `
The ´polaris_sso_group´ data source is used to access information about an SSO
group in RSC. An SSO group is looked up using either the ID or the name.
`

var _ datasource.DataSource = &ssoGroupDataSource{}

type ssoGroupDataSource struct {
	client *client
}

type ssoGroupModel struct {
	ID         types.String `tfsdk:"id"`
	DomainName types.String `tfsdk:"domain_name"`
	Name       types.String `tfsdk:"name"`
	Roles      types.Set    `tfsdk:"roles"`
	SSOGroupID types.String `tfsdk:"sso_group_id"`
	Users      types.Set    `tfsdk:"users"`
}

func newSSOGroupDataSource() datasource.DataSource {
	return &ssoGroupDataSource{}
}

func (d *ssoGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "ssoGroupDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keySSOGroup
}

func (d *ssoGroupDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "ssoGroupDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceSSOGroupDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SSO group ID.",
			},
			keyDomainName: schema.StringAttribute{
				Computed:    true,
				Description: "The domain name of the SSO group.",
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "SSO group name.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keySSOGroupID)),
				},
			},
			keyRoles: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Roles assigned to the SSO group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyID: schema.StringAttribute{
							Computed:    true,
							Description: "Role ID (UUID).",
						},
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Role name.",
						},
					},
				},
			},
			keySSOGroupID: schema.StringAttribute{
				Optional:    true,
				Description: "SSO group ID.",
			},
			keyUsers: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Users in the SSO group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyID: schema.StringAttribute{
							Computed:    true,
							Description: "User ID.",
						},
						keyEmail: schema.StringAttribute{
							Computed:    true,
							Description: "User email address.",
						},
					},
				},
			},
		},
	}
}

func (d *ssoGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "ssoGroupDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *ssoGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "ssoGroupDataSource.Read")

	var config ssoGroupModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("Client error", err.Error())
		return
	}

	var group gqlaccess.SSOGroup
	if !config.SSOGroupID.IsNull() {
		group, err = access.Wrap(polarisClient).SSOGroupByID(ctx, config.SSOGroupID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read SSO group", err.Error())
			return
		}
	} else {
		group, err = access.Wrap(polarisClient).SSOGroupByName(ctx, config.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read SSO group", err.Error())
			return
		}
	}

	rolesSet, diags := fromRoleRefs(group.Roles)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	usersSet, diags := fromUserRefs(group.Users)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := ssoGroupModel{
		ID:         types.StringValue(group.ID),
		DomainName: types.StringValue(group.DomainName),
		Name:       types.StringValue(group.Name),
		Roles:      rolesSet,
		SSOGroupID: types.StringValue(group.ID),
		Users:      usersSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

// roleRefAttrTypes returns the attribute types for the role nested set.
func roleRefAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyID:   types.StringType,
		keyName: types.StringType,
	}
}

// fromRoleRefs converts a slice of RoleRef to a Terraform Framework set.
func fromRoleRefs(roles []gqlaccess.RoleRef) (types.Set, diag.Diagnostics) {
	roleValues := make([]attr.Value, 0, len(roles))
	for _, role := range roles {
		roleValue, diags := types.ObjectValue(roleRefAttrTypes(), map[string]attr.Value{
			keyID:   types.StringValue(role.ID.String()),
			keyName: types.StringValue(role.Name),
		})
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: roleRefAttrTypes()}), diags
		}
		roleValues = append(roleValues, roleValue)
	}

	return types.SetValue(types.ObjectType{AttrTypes: roleRefAttrTypes()}, roleValues)
}

// userRefAttrTypes returns the attribute types for the user nested set.
func userRefAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyID:    types.StringType,
		keyEmail: types.StringType,
	}
}

// fromUserRefs converts a slice of UserRef to a Terraform Framework set.
func fromUserRefs(users []gqlaccess.UserRef) (types.Set, diag.Diagnostics) {
	userValues := make([]attr.Value, 0, len(users))
	for _, user := range users {
		userValue, diags := types.ObjectValue(userRefAttrTypes(), map[string]attr.Value{
			keyID:    types.StringValue(user.ID),
			keyEmail: types.StringValue(user.Email),
		})
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: userRefAttrTypes()}), diags
		}
		userValues = append(userValues, userValue)
	}

	return types.SetValue(types.ObjectType{AttrTypes: userRefAttrTypes()}, userValues)
}
