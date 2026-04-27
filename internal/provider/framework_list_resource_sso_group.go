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
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

const listResourceSSOGroupDescription = `
The ´polaris_sso_group´ list resource lists SSO groups in RSC.
`

var (
	_ list.ListResource              = &ssoGroupListResource{}
	_ list.ListResourceWithConfigure = &ssoGroupListResource{}
)

type ssoGroupListResource struct {
	client *client
}

type ssoGroupListConfigModel struct {
	Name         types.String `tfsdk:"name"`
	AuthDomainID types.String `tfsdk:"auth_domain_id"`
}

func newSSOGroupListResource() list.ListResource {
	return &ssoGroupListResource{}
}

func (r *ssoGroupListResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "ssoGroupListResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keySSOGroup
}

func (r *ssoGroupListResource) ListResourceConfigSchema(ctx context.Context, _ list.ListResourceSchemaRequest, res *list.ListResourceSchemaResponse) {
	tflog.Trace(ctx, "ssoGroupListResource.ListResourceConfigSchema")

	res.Schema = listschema.Schema{
		Description: description(listResourceSSOGroupDescription),
		Attributes: map[string]listschema.Attribute{
			keyName: listschema.StringAttribute{
				Optional:    true,
				Description: "Filter SSO groups by name. Matches groups whose name contains the given value (case-insensitive).",
			},
			keyAuthDomainID: listschema.StringAttribute{
				Optional:    true,
				Description: "Filter SSO groups by auth domain ID (identity provider ID).",
			},
		},
	}
}

func (r *ssoGroupListResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "ssoGroupListResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *ssoGroupListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	tflog.Trace(ctx, "ssoGroupListResource.List")

	var config ssoGroupListConfigModel
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

	var filter gqlaccess.SSOGroupFilter
	if !config.Name.IsNull() {
		filter.Name = config.Name.ValueString()
	}
	if !config.AuthDomainID.IsNull() {
		filter.AuthDomainIDs = []string{config.AuthDomainID.ValueString()}
	}

	groups, err := gqlaccess.ListSSOGroups(ctx, polarisClient.GQL, filter)
	if err != nil {
		diags.AddError("Failed to list SSO groups", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i, group := range groups {
			if int64(i) >= req.Limit {
				return
			}

			result := req.NewListResult(ctx)
			result.DisplayName = group.Name

			identity := ssoGroupIdentityModel{
				ID: types.StringValue(group.ID),
			}
			result.Diagnostics.Append(result.Identity.Set(ctx, identity)...)
			if result.Diagnostics.HasError() {
				push(result)
				return
			}

			if req.IncludeResource {
				roleIDs := make([]string, 0, len(group.Roles))
				for _, role := range group.Roles {
					roleIDs = append(roleIDs, role.ID.String())
				}
				roleIDsSet, setDiags := types.SetValueFrom(ctx, types.StringType, roleIDs)
				result.Diagnostics.Append(setDiags...)
				if result.Diagnostics.HasError() {
					push(result)
					return
				}

				model := ssoGroupResourceModel{
					ID:         types.StringValue(group.ID),
					DomainName: types.StringValue(group.DomainName),
					GroupName:  types.StringValue(group.Name),
					RoleIDs:    roleIDsSet,
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
