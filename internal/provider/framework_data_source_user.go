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
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const dataSourceUserDescription = `
The ´polaris_user´ data source is used to access information about an RSC user.
Information for both local and SSO users can be accessed. A user is looked up
using either the ID or the email address.

-> **Note:** RSC allows the same email address to be used, at the same time, by
   both local and SSO users. Use the ´domain´ field to specify in which domain
   to look for a user.

-> **Note:** The ´status´ field will always be ´UNKNOWN´ for SSO users.
`

var _ datasource.DataSource = &userDataSource{}

type userDataSource struct {
	client *client
}

type userModel struct {
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	Email          types.String `tfsdk:"email"`
	IsAccountOwner types.Bool   `tfsdk:"is_account_owner"`
	Roles          types.Set    `tfsdk:"roles"`
	Status         types.String `tfsdk:"status"`
	UserID         types.String `tfsdk:"user_id"`
}

func newUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "userDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyUser
}

func (d *userDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "userDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceUserDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "User ID.",
			},
			keyDomain: schema.StringAttribute{
				Optional: true,
				Description: "The domain in which to look for a user when an email address is specified. " +
					"Possible values are `LOCAL` and `SSO`.",
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "SSO"),
					stringvalidator.ConflictsWith(path.MatchRoot(keyUserID)),
				},
			},
			keyEmail: schema.StringAttribute{
				Optional:    true,
				Description: "User email address.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyUserID)),
				},
			},
			keyIsAccountOwner: schema.BoolAttribute{
				Computed:    true,
				Description: "True if the user is the account owner.",
			},
			keyRoles: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Roles assigned to the user.",
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
			keyStatus: schema.StringAttribute{
				Computed:    true,
				Description: "User status.",
			},
			keyUserID: schema.StringAttribute{
				Optional:    true,
				Description: "User ID.",
			},
		},
	}
}

func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "userDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "userDataSource.Read")

	var config userModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var user gqlaccess.User
	if !config.UserID.IsNull() {
		user, err = access.Wrap(polarisClient).UserByID(ctx, config.UserID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read user", err.Error())
			return
		}
	} else {
		var domain gqlaccess.UserDomain
		if !config.Domain.IsNull() {
			domain = gqlaccess.UserDomain(config.Domain.ValueString())
		}
		user, err = access.Wrap(polarisClient).UserByEmail(ctx, config.Email.ValueString(), domain)
		if err != nil {
			res.Diagnostics.AddError("Failed to read user", err.Error())
			return
		}
	}

	rolesSet, diags := fromRoleRefs(user.Roles)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := userModel{
		ID:             types.StringValue(user.ID),
		Domain:         types.StringValue(string(user.Domain)),
		Email:          types.StringValue(user.Email),
		IsAccountOwner: types.BoolValue(user.IsAccountOwner),
		Roles:          rolesSet,
		Status:         types.StringValue(user.Status),
		UserID:         types.StringValue(user.ID),
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
