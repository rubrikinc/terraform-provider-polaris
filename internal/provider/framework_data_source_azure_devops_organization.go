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
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	gqldevops "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/devops"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceAzureDevOpsOrganizationDescription = `
The ´polaris_azure_devops_organization´ data source reads an onboarded Azure
DevOps organization from RSC. Look it up by ´id´ or by ´native_id´. The
´native_id´ is the Azure DevOps organization name shown in the organization's
URL (e.g. ´my-org´ in https://dev.azure.com/my-org).
`

var (
	_ datasource.DataSource              = &azureDevOpsOrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsOrganizationDataSource{}
)

type azureDevOpsOrganizationDataSource struct {
	client *client
}

type azureDevOpsOrganizationDataSourceModel struct {
	ID                           types.String `tfsdk:"id"`
	NativeID                     types.String `tfsdk:"native_id"`
	TenantDomain                 types.String `tfsdk:"tenant_domain"`
	Cloud                        types.String `tfsdk:"cloud"`
	Feature                      types.Set    `tfsdk:"feature"`
	ExocomputeHostType           types.String `tfsdk:"exocompute_host_type"`
	StorageType                  types.String `tfsdk:"storage_type"`
	ArchivalLocationID           types.String `tfsdk:"archival_location_id"`
	ExocomputeHostCloudAccountID types.String `tfsdk:"exocompute_host_cloud_account_id"`
	ExocomputeRegion             types.String `tfsdk:"exocompute_region"`
	ConnectionStatus             types.String `tfsdk:"connection_status"`
	ProjectCount                 types.Int64  `tfsdk:"project_count"`
	RepoCount                    types.Int64  `tfsdk:"repo_count"`
	LastRefreshTime              types.String `tfsdk:"last_refresh_time"`
}

func newAzureDevOpsOrganizationDataSource() datasource.DataSource {
	return &azureDevOpsOrganizationDataSource{}
}

func (d *azureDevOpsOrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsOrganization
}

func (d *azureDevOpsOrganizationDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureDevOpsOrganizationDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "RSC organization ID (UUID). Exactly one of `id` or `native_id` must be set.",
			},
			keyNativeID: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyID)),
				},
				Description: "Azure DevOps organization native identifier. This is the organization name " +
					"visible in the Azure DevOps URL (e.g., `my-org` from https://dev.azure.com/my-org). " +
					"Exactly one of `id` or `native_id` must be set.",
			},
			keyTenantDomain: schema.StringAttribute{
				Computed:    true,
				Description: "Azure AD tenant primary domain.",
			},
			keyCloud: schema.StringAttribute{
				Computed:    true,
				Description: "Azure cloud type.",
			},
			keyFeature: schema.SetNestedAttribute{
				Computed:    true,
				Description: "RSC features enabled for the organization, with their permission groups.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Feature name.",
						},
						keyPermissionGroups: schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Permission groups enabled for the feature.",
						},
					},
				},
			},
			keyExocomputeHostType: schema.StringAttribute{
				Computed:    true,
				Description: "Type of exocompute host.",
			},
			keyStorageType: schema.StringAttribute{
				Computed:    true,
				Description: "Type of backup storage.",
			},
			keyArchivalLocationID: schema.StringAttribute{
				Computed:    true,
				Description: "Archival location ID for backups. Set when `storage_type` is `BYOS`.",
			},
			keyExocomputeHostCloudAccountID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC cloud account ID providing exocompute. Set when `exocompute_host_type` is `CUSTOMER_HOST`.",
			},
			keyExocomputeRegion: schema.StringAttribute{
				Computed:    true,
				Description: "Azure region for Rubrik-hosted exocompute. Set when `exocompute_host_type` is `RUBRIK_HOST`.",
			},
			keyConnectionStatus: schema.StringAttribute{
				Computed:    true,
				Description: "Connection status of the organization.",
			},
			keyProjectCount: schema.Int64Attribute{
				Computed:    true,
				Description: "Number of projects in the organization.",
			},
			keyRepoCount: schema.Int64Attribute{
				Computed:    true,
				Description: "Number of repositories in the organization.",
			},
			keyLastRefreshTime: schema.StringAttribute{
				Computed:    true,
				Description: "Time the organization was last refreshed (RFC3339).",
			},
		},
	}
}

func (d *azureDevOpsOrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureDevOpsOrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationDataSource.Read")

	var config azureDevOpsOrganizationDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var org gqldevops.AzureOrganization
	if !config.ID.IsNull() {
		id, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid organization ID", err.Error())
			return
		}
		org, err = devops.Wrap(polarisClient).AzureOrganizationByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
			return
		}
	} else {
		nativeID := config.NativeID.ValueString()

		// The exact-match filter applies to the organization's name, which
		// equals its native ID, so the lookup resolves the native ID
		// server-side.
		candidates, err := devops.Wrap(polarisClient).AzureOrganizationsByName(ctx, nativeID,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{nativeID}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps organization", err.Error())
			return
		}

		switch len(candidates) {
		case 0:
			res.Diagnostics.AddError("Azure DevOps organization not found", fmt.Sprintf("no organization with native ID %q", nativeID))
			return
		case 1:
			org = candidates[0]
		default:
			res.Diagnostics.AddError("Multiple Azure DevOps organizations found",
				fmt.Sprintf("%d organizations have native ID %q; look up by id instead", len(candidates), nativeID))
			return
		}
	}

	config.ID = types.StringValue(org.ID.String())
	config.NativeID = types.StringValue(org.NativeID)
	config.TenantDomain = types.StringValue(org.TenantDomain)
	config.Cloud = types.StringValue(fromAzureCloud(org.Cloud))
	config.ConnectionStatus = types.StringValue(string(org.ConnectionStatus))
	config.ProjectCount = types.Int64Value(int64(org.ProjectCount))
	config.RepoCount = types.Int64Value(int64(org.RepoCount))
	config.LastRefreshTime = lastRefreshTime(org)

	// RUBRIK_HOST carries an exocompute region, CUSTOMER_HOST carries an
	// exocompute cloud account.
	config.ExocomputeHostType = types.StringValue(string(org.RepoHostType))
	switch org.RepoHostType {
	case gqldevops.HostTypeRubrik:
		if org.RubrikHostedExocompute != nil {
			config.ExocomputeRegion = types.StringValue(org.RubrikHostedExocompute.Region.Name())
		}
		config.ExocomputeHostCloudAccountID = types.StringNull()
	case gqldevops.HostTypeCustomer:
		if org.CloudNativeExocompute != nil {
			config.ExocomputeHostCloudAccountID = types.StringValue(org.CloudNativeExocompute.ID.String())
		}
		config.ExocomputeRegion = types.StringNull()
	}

	// BYOS carries a backup location, RCV auto-provisions storage and takes no
	// backup location.
	if org.BackupLocation != nil && org.BackupLocation.StorageType == gqldevops.StorageTypeBYOS {
		config.StorageType = types.StringValue(string(gqldevops.StorageTypeBYOS))
		config.ArchivalLocationID = types.StringValue(org.BackupLocation.ArchivalGroupID.String())
	} else {
		config.StorageType = types.StringValue(string(gqldevops.StorageTypeRCV))
		config.ArchivalLocationID = types.StringNull()
	}

	// Read the organizations current features and permission groups.
	perms, err := devops.Wrap(polarisClient).ListOrgPermissions(ctx, org.ID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization permissions", err.Error())
		return
	}
	featureSet, diags := fromFeatures(perms.ToFeatures())
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	config.Feature = featureSet

	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}

// lastRefreshTime returns the organization's last refresh time as an RFC3339
// string value, or null when unset.
func lastRefreshTime(org gqldevops.AzureOrganization) types.String {
	if org.LastRefreshTime == nil {
		return types.StringNull()
	}
	return types.StringValue(org.LastRefreshTime.Format(time.RFC3339))
}
