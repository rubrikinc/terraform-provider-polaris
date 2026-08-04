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

const dataSourceGCPRegionsDescription = `
The ´polaris_gcp_regions´ data source returns the GCP regions RSC supports for a
cloud account, each with its availability zones. Use it to discover a valid
´region´ and ´zone´ for the ´polaris_gcp_cloud_cluster´ resource, and to drive
the ´subnet_az_config´ blocks of a Multi-AZ cluster with ´for_each´ over a
region's zones.
`

var _ datasource.DataSource = &gcpRegionsDataSource{}

type gcpRegionsDataSource struct {
	client *client
}

type gcpRegionsModel struct {
	ID             types.String `tfsdk:"id"`
	CloudAccountID types.String `tfsdk:"cloud_account_id"`
	Regions        types.Set    `tfsdk:"regions"`
}

func newGcpRegionsDataSource() datasource.DataSource {
	return &gcpRegionsDataSource{}
}

func (d *gcpRegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "gcpRegionsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_gcp_regions"
}

func (d *gcpRegionsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "gcpRegionsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceGCPRegionsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the regions returned.",
			},
			keyCloudAccountID: schema.StringAttribute{
				Required:    true,
				Description: "RSC cloud account ID (UUID).",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyRegions: schema.SetNestedAttribute{
				Computed:    true,
				Description: "GCP regions available for the cloud account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Region name, e.g. `us-west1`.",
						},
						keyZones: schema.SetAttribute{
							ElementType: types.StringType,
							Computed:    true,
							Description: "Availability zones in the region, e.g. `us-west1-a`.",
						},
					},
				},
			},
		},
	}
}

func (d *gcpRegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "gcpRegionsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *gcpRegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "gcpRegionsDataSource.Read")

	var config gcpRegionsModel
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

	regions, err := gqlcloudcluster.Wrap(polarisClient.GQL).GcpRegions(ctx, cloudAccountID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read GCP regions", err.Error())
		return
	}

	slices.SortFunc(regions, func(a, b gqlcloudcluster.GcpRegionInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	hash := sha256.New()
	regionValues := make([]attr.Value, 0, len(regions))
	for _, region := range regions {
		hash.Write([]byte(region.Name))

		zones := slices.Clone(region.Zones)
		slices.Sort(zones)
		for _, z := range zones {
			hash.Write([]byte(z))
		}

		zoneSet, diags := types.SetValueFrom(ctx, types.StringType, zones)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		regionValue, diags := types.ObjectValue(gcpRegionAttrTypes(), map[string]attr.Value{
			keyName:  types.StringValue(region.Name),
			keyZones: zoneSet,
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		regionValues = append(regionValues, regionValue)
	}

	regionSet, diags := types.SetValue(types.ObjectType{AttrTypes: gcpRegionAttrTypes()}, regionValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := gcpRegionsModel{
		ID:             types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		CloudAccountID: config.CloudAccountID,
		Regions:        regionSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func gcpRegionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:  types.StringType,
		keyZones: types.SetType{ElemType: types.StringType},
	}
}
