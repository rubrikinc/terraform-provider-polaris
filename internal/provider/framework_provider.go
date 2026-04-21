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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
)

var _ provider.ProviderWithListResources = &FrameworkProvider{}

const Name = "registry.terraform.io/rubrikinc/polaris"

type FrameworkProvider struct {
	Version string
}

type frameworkProviderModel struct {
	Credentials      types.String `tfsdk:"credentials"`
	TokenCache       types.Bool   `tfsdk:"token_cache"`
	TokenCacheDir    types.String `tfsdk:"token_cache_dir"`
	TokenCacheSecret types.String `tfsdk:"token_cache_secret"`
}

func (p *FrameworkProvider) Metadata(ctx context.Context, _ provider.MetadataRequest, res *provider.MetadataResponse) {
	tflog.Trace(ctx, "FrameworkProvider.Metadata")

	res.TypeName = keyPolaris
	res.Version = p.Version
}

func (p *FrameworkProvider) Schema(ctx context.Context, _ provider.SchemaRequest, res *provider.SchemaResponse) {
	tflog.Trace(ctx, "FrameworkProvider.Schema")

	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			keyCredentials: schema.StringAttribute{
				Optional: true,
				Description: "The service account credentials, service account credentials file name or local user " +
					"account name to use when accessing RSC.",
			},
			keyTokenCache: schema.BoolAttribute{
				Optional:    true,
				Description: "Enable or disable the token cache. The token cache is enabled by default.",
			},
			keyTokenCacheDir: schema.StringAttribute{
				Optional: true,
				Description: "The directory where cached authentication tokens are stored. The OS directory for " +
					"temporary files is used by default.",
			},
			keyTokenCacheSecret: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "The secret used as input when generating an encryption key for the authentication " +
					"token. The encryption key is derived from the RSC account information by default.",
			},
		},
	}
}

func (p *FrameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, res *provider.ConfigureResponse) {
	tflog.Trace(ctx, "FrameworkProvider.Configure")

	var config frameworkProviderModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	var credentials string
	if !config.Credentials.IsNull() {
		credentials = config.Credentials.ValueString()
	}

	cacheParams := polaris.CacheParams{
		Enable: config.TokenCache.ValueBool(),
	}
	if !config.TokenCacheDir.IsNull() {
		cacheParams.Dir = config.TokenCacheDir.ValueString()
	}
	if !config.TokenCacheSecret.IsNull() {
		cacheParams.Secret = config.TokenCacheSecret.ValueString()
	}

	c, err := newClient(ctx, credentials, cacheParams)
	if err != nil {
		res.Diagnostics.AddError("Failed to configure provider", err.Error())
		return
	}

	res.ResourceData = c
	res.DataSourceData = c
	res.ListResourceData = c
}

func (p *FrameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "FrameworkProvider.Resources")

	return []func() resource.Resource{
		newCustomRoleResource,
		newRoleAssignmentResource,
		newSSOGroupResource,
		newUserResource,
	}
}

func (p *FrameworkProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "FrameworkProvider.DataSources")

	return []func() datasource.DataSource{
		newFeatureFlagDataSource,
		newIdentityProviderDataSource,
		newRoleDataSource,
		newRoleTemplateDataSource,
		newSSOGroupDataSource,
		newUserDataSource,
	}
}

func (p *FrameworkProvider) ListResources(ctx context.Context) []func() list.ListResource {
	tflog.Trace(ctx, "FrameworkProvider.ListResources")

	return []func() list.ListResource{
		newCustomRoleListResource,
		newUserListResource,
	}
}
