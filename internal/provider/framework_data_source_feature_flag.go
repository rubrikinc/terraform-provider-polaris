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
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceFeatureFlagDescription = `
The ´polaris_feature_flag´ data source is used to check if a feature flag is enabled for
the RSC account.
`

var _ datasource.DataSource = &featureFlagDataSource{}

type featureFlagDataSource struct {
	client *client
}

type featureFlagModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func newFeatureFlagDataSource() datasource.DataSource {
	return &featureFlagDataSource{}
}

func (d *featureFlagDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "featureFlagDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyFeatureFlag
}

func (d *featureFlagDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "featureFlagDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceFeatureFlagDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the feature flag name.",
			},
			keyName: schema.StringAttribute{
				Required:    true,
				Description: "Feature flag name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			keyEnabled: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the feature flag is enabled for the RSC account.",
			},
		},
	}
}

func (d *featureFlagDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "featureFlagDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *featureFlagDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "featureFlagDataSource.Read")

	var config featureFlagModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	name := config.Name.ValueString()
	flag, err := core.Wrap(polarisClient.GQL).FeatureFlag(ctx, core.FeatureFlagName(name))
	if err != nil {
		res.Diagnostics.AddError("Failed to read feature flag "+name, err.Error())
		return
	}

	hash := sha256.New()
	hash.Write([]byte(name))

	state := featureFlagModel{
		ID:      types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		Name:    types.StringValue(name),
		Enabled: types.BoolValue(flag.Enabled),
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
