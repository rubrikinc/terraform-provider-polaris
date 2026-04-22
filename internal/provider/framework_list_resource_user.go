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

	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
)

const listResourceUserDescription = `
The ´polaris_user´ list resource lists local users in RSC.
`

var (
	_ list.ListResource              = &userListResource{}
	_ list.ListResourceWithConfigure = &userListResource{}
)

type userListResource struct {
	client *client
}

type userListConfigModel struct {
	Email types.String `tfsdk:"email"`
}

func newUserListResource() list.ListResource {
	return &userListResource{}
}

func (r *userListResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "userListResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyUser
}

func (r *userListResource) ListResourceConfigSchema(ctx context.Context, _ list.ListResourceSchemaRequest, res *list.ListResourceSchemaResponse) {
	tflog.Trace(ctx, "userListResource.ListResourceConfigSchema")

	res.Schema = listschema.Schema{
		Description: description(listResourceUserDescription),
		Attributes: map[string]listschema.Attribute{
			keyEmail: listschema.StringAttribute{
				Optional:    true,
				Description: "Filter users by email.",
			},
		},
	}
}

func (r *userListResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "userListResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *userListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	tflog.Trace(ctx, "userListResource.List")

	var config userListConfigModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		diags.AddError("RSC client error", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	emailFilter := config.Email.ValueString()
	users, err := access.Wrap(polarisClient).Users(ctx, emailFilter)
	if err != nil {
		diags.AddError("Failed to list users", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i, user := range users {
			if int64(i) >= req.Limit {
				return
			}

			result := req.NewListResult(ctx)
			result.DisplayName = user.Email

			identity := userIdentityModel{
				ID: types.StringValue(user.ID),
			}
			result.Diagnostics.Append(result.Identity.Set(ctx, identity)...)
			if result.Diagnostics.HasError() {
				push(result)
				return
			}

			if req.IncludeResource {
				roleIDs := make([]string, 0, len(user.Roles))
				for _, role := range user.Roles {
					roleIDs = append(roleIDs, role.ID.String())
				}
				roleIDsSet, setDiags := types.SetValueFrom(ctx, types.StringType, roleIDs)
				result.Diagnostics.Append(setDiags...)
				if result.Diagnostics.HasError() {
					push(result)
					return
				}

				model := userResourceModel{
					ID:             types.StringValue(user.ID),
					Domain:         types.StringValue(string(user.Domain)),
					Email:          types.StringValue(user.Email),
					IsAccountOwner: types.BoolValue(user.IsAccountOwner),
					RoleIDs:        roleIDsSet,
					Status:         types.StringValue(user.Status),
				}
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
				if result.Diagnostics.HasError() {
					push(result)
					return
				}
			}

			if !push(result) {
				return
			}
		}
	}
}
