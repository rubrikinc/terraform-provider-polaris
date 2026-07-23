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
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gqlcloudcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cloudcluster"
)

const dataSourceGCPServiceAccountsDescription = `
The ´polaris_gcp_service_accounts´ data source returns the GCP service accounts
that RSC has discovered for a cloud account. It exposes the same catalog the RSC
UI uses when selecting a service account for a GCP cloud cluster, so the service
account email required by the ´polaris_gcp_cloud_cluster´ resource can be
discovered at plan time instead of hard-coded.
`

var _ datasource.DataSource = &gcpServiceAccountsDataSource{}

type gcpServiceAccountsDataSource struct {
	client *client
}

type gcpServiceAccountsModel struct {
	ID              types.String `tfsdk:"id"`
	CloudAccountID  types.String `tfsdk:"cloud_account_id"`
	ServiceAccounts types.Set    `tfsdk:"service_accounts"`
}

func newGcpServiceAccountsDataSource() datasource.DataSource {
	return &gcpServiceAccountsDataSource{}
}

func (d *gcpServiceAccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "gcpServiceAccountsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_gcp_service_accounts"
}

func (d *gcpServiceAccountsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "gcpServiceAccountsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceGCPServiceAccountsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the service account emails returned.",
			},
			keyCloudAccountID: schema.StringAttribute{
				Required:    true,
				Description: "RSC cloud account ID (UUID).",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyServiceAccounts: schema.SetNestedAttribute{
				Computed:    true,
				Description: "GCP service accounts available for the cloud account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyEmail: schema.StringAttribute{
							Computed:    true,
							Description: "Service account email.",
						},
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Service account display name.",
						},
					},
				},
			},
		},
	}
}

func (d *gcpServiceAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "gcpServiceAccountsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *gcpServiceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "gcpServiceAccountsDataSource.Read")

	var config gcpServiceAccountsModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	cloudAccountID, err := uuid.Parse(config.CloudAccountID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid cloud account ID", err.Error())
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	serviceAccounts, err := gqlcloudcluster.Wrap(polarisClient.GQL).GcpServiceAccounts(ctx, cloudAccountID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read GCP service accounts", err.Error())
		return
	}

	slices.SortFunc(serviceAccounts, func(a, b gqlcloudcluster.GcpServiceAccount) int {
		return cmp.Compare(a.Email, b.Email)
	})

	hash := sha256.New()
	saValues := make([]attr.Value, 0, len(serviceAccounts))
	for _, sa := range serviceAccounts {
		hash.Write([]byte(sa.Email))

		saValue, diags := types.ObjectValue(gcpServiceAccountAttrTypes(), map[string]attr.Value{
			keyEmail: types.StringValue(sa.Email),
			keyName:  types.StringValue(sa.Name),
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		saValues = append(saValues, saValue)
	}

	saSet, diags := types.SetValue(types.ObjectType{AttrTypes: gcpServiceAccountAttrTypes()}, saValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := gcpServiceAccountsModel{
		ID:              types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		CloudAccountID:  config.CloudAccountID,
		ServiceAccounts: saSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func gcpServiceAccountAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyEmail: types.StringType,
		keyName:  types.StringType,
	}
}
