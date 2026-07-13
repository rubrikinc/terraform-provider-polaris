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
DevOps organization from RSC. Look it up by ´id´ or by ´native_id´.
`

var (
	_ datasource.DataSource              = &azureDevOpsOrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsOrganizationDataSource{}
)

type azureDevOpsOrganizationDataSource struct {
	client *client
}

type azureDevOpsOrganizationDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	NativeID         types.String `tfsdk:"native_id"`
	TenantDomain     types.String `tfsdk:"tenant_domain"`
	ConnectionStatus types.String `tfsdk:"connection_status"`
	ProjectCount     types.Int64  `tfsdk:"project_count"`
	RepoCount        types.Int64  `tfsdk:"repo_count"`
	LastRefreshTime  types.String `tfsdk:"last_refresh_time"`
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
					"visible in the Azure DevOps URL (e.g., \"my-org\" from https://dev.azure.com/my-org). " +
					"Exactly one of `id` or `native_id` must be set.",
			},
			keyTenantDomain: schema.StringAttribute{
				Computed:    true,
				Description: "Azure AD tenant primary domain.",
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

	var id uuid.UUID
	switch {
	case !config.ID.IsNull():
		id, err = uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid organization ID", err.Error())
			return
		}
	case !config.NativeID.IsNull():
		nativeID := config.NativeID.ValueString()
		activeFilters := activeObjectFilters()
		objects, err := hierarchy.ObjectsByName[hierarchy.AzureDevOpsOrganization](ctx, hierarchy.Wrap(polarisClient.GQL), nativeID, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps organization", err.Error())
			return
		}
		var ids []uuid.UUID
		for _, obj := range objects {
			ids = append(ids, obj.Object.ID)
		}
		switch len(ids) {
		case 0:
			res.Diagnostics.AddError("Azure DevOps organization not found", "no organization with native ID "+nativeID)
			return
		case 1:
			id = ids[0]
		default:
			res.Diagnostics.AddError("Multiple Azure DevOps organizations found",
				fmt.Sprintf("%d organizations have native ID %q; look up by id instead", len(ids), nativeID))
			return
		}
	}

	org, err := devops.Wrap(polarisClient).AzureOrganizationByID(ctx, id)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
		return
	}

	config.ID = types.StringValue(org.ID.String())
	config.NativeID = types.StringValue(org.NativeID)
	config.TenantDomain = types.StringValue(org.TenantDomain)
	config.ConnectionStatus = types.StringValue(string(org.ConnectionStatus))
	config.ProjectCount = types.Int64Value(int64(org.ProjectCount))
	config.RepoCount = types.Int64Value(int64(org.RepoCount))
	config.LastRefreshTime = lastRefreshTime(org)

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
