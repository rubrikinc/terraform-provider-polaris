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
	"errors"
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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqldevops "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/devops"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceGitHubOrganizationDescription = `
The ´polaris_github_organization´ data source reads an onboarded GitHub
organization from RSC. Look it up by ´id´, ´name´, or ´native_id´. The ´name´
is the GitHub organization login shown in the organization's URL (e.g. ´my-org´
in https://github.com/my-org). The ´native_id´ is GitHub's numeric organization
ID (e.g. ´54376070´), which is stable across organization renames. Unlike some
other providers, a GitHub organization's ´name´ and ´native_id´ are different
values.

GitHub organizations cannot be onboarded through the provider; use the RSC UI
to onboard them. This data source is read-only.
`

var (
	_ datasource.DataSource              = &gitHubOrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &gitHubOrganizationDataSource{}
)

type gitHubOrganizationDataSource struct {
	client *client
}

type gitHubOrganizationDataSourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	NativeID                     types.String `tfsdk:"native_id"`
	OrgURL                       types.String `tfsdk:"org_url"`
	Feature                      types.Set    `tfsdk:"feature"`
	ExocomputeHostType           types.String `tfsdk:"exocompute_host_type"`
	StorageType                  types.String `tfsdk:"storage_type"`
	ArchivalLocationID           types.String `tfsdk:"archival_location_id"`
	ExocomputeHostCloudAccountID types.String `tfsdk:"exocompute_host_cloud_account_id"`
	ExocomputeRegion             types.String `tfsdk:"exocompute_region"`
	ConnectionStatus             types.String `tfsdk:"connection_status"`
	RepoCount                    types.Int64  `tfsdk:"repo_count"`
	LastRefreshTime              types.String `tfsdk:"last_refresh_time"`
}

func newGitHubOrganizationDataSource() datasource.DataSource {
	return &gitHubOrganizationDataSource{}
}

func (d *gitHubOrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "gitHubOrganizationDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyGitHubOrganization
}

func (d *gitHubOrganizationDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "gitHubOrganizationDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceGitHubOrganizationDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "RSC organization ID (UUID). Exactly one of `id`, `name`, or `native_id` must be set.",
			},
			keyName: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyID), path.MatchRoot(keyNativeID)),
				},
				Description: "GitHub organization name. This is the organization login visible in the GitHub " +
					"URL (e.g., `my-org` from https://github.com/my-org). Exactly one of `id`, `name`, or " +
					"`native_id` must be set.",
			},
			keyNativeID: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "GitHub organization native identifier. This is GitHub's numeric organization " +
					"ID (e.g., `54376070`), which is stable across organization renames. Exactly one of " +
					"`id`, `name`, or `native_id` must be set.",
			},
			keyOrgURL: schema.StringAttribute{
				Computed:    true,
				Description: "GitHub organization URL.",
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
				Computed: true,
				Description: "RSC cloud account ID providing exocompute. Set when `exocompute_host_type` is " +
					"`CUSTOMER_HOST`.",
			},
			keyExocomputeRegion: schema.StringAttribute{
				Computed:    true,
				Description: "Region for Rubrik-hosted exocompute. Set when `exocompute_host_type` is `RUBRIK_HOST`.",
			},
			keyConnectionStatus: schema.StringAttribute{
				Computed:    true,
				Description: "Connection status of the organization.",
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

func (d *gitHubOrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "gitHubOrganizationDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *gitHubOrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "gitHubOrganizationDataSource.Read")

	var config gitHubOrganizationDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var org gqldevops.GitHubOrganization
	switch {
	case !config.ID.IsNull():
		id, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid organization ID", err.Error())
			return
		}
		org, err = devops.Wrap(polarisClient).GitHubOrganizationByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read GitHub organization", err.Error())
			return
		}
	case !config.NativeID.IsNull():
		// The native ID is GitHub's numeric organization ID, distinct from the
		// name, so it is matched client-side rather than through a name filter.
		nativeID := config.NativeID.ValueString()
		org, err = devops.Wrap(polarisClient).GitHubOrganizationByNativeID(ctx, nativeID)
		if err != nil {
			if errors.Is(err, graphql.ErrNotFound) {
				res.Diagnostics.AddError("GitHub organization not found",
					fmt.Sprintf("no organization with native ID %q", nativeID))
				return
			}
			res.Diagnostics.AddError("Failed to look up GitHub organization", err.Error())
			return
		}
	default:
		name := config.Name.ValueString()

		candidates, err := devops.Wrap(polarisClient).GitHubOrganizationsByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up GitHub organization", err.Error())
			return
		}

		switch len(candidates) {
		case 0:
			res.Diagnostics.AddError("GitHub organization not found", fmt.Sprintf("no organization named %q", name))
			return
		case 1:
			org = candidates[0]
		default:
			res.Diagnostics.AddError("Multiple GitHub organizations found",
				fmt.Sprintf("%d organizations are named %q; look up by id instead", len(candidates), name))
			return
		}
	}

	config.ID = types.StringValue(org.ID.String())
	config.Name = types.StringValue(org.Name)
	config.NativeID = types.StringValue(org.NativeID)
	config.OrgURL = types.StringValue(org.OrgURL)
	config.ConnectionStatus = types.StringValue(string(org.ConnectionStatus))
	config.RepoCount = types.Int64Value(int64(org.RepoCount))
	if org.LastRefreshTime != nil {
		config.LastRefreshTime = types.StringValue(org.LastRefreshTime.Format(time.RFC3339))
	} else {
		config.LastRefreshTime = types.StringNull()
	}

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
		if org.Exocompute != nil {
			config.ExocomputeHostCloudAccountID = types.StringValue(org.Exocompute.ID.String())
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

	// Read the organization's current features and permission groups.
	perms, err := devops.Wrap(polarisClient).GitHubListOrgPermissions(ctx, org.ID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read GitHub organization permissions", err.Error())
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
