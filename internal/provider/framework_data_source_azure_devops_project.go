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

const dataSourceAzureDevOpsProjectDescription = `
The ´polaris_azure_devops_project´ data source reads an Azure DevOps project from
RSC. Look it up by ´id´ or by ´name´.

Project names are only unique within an organization. When looking up by ´name´,
set ´org_id´ to disambiguate a name shared across organizations; without it a
name matching more than one project is an error.
`

var (
	_ datasource.DataSource              = &azureDevOpsProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsProjectDataSource{}
)

type azureDevOpsProjectDataSource struct {
	client *client
}

type azureDevOpsProjectDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	NativeID  types.String `tfsdk:"native_id"`
	OrgID     types.String `tfsdk:"org_id"`
	OrgName   types.String `tfsdk:"org_name"`
	URL       types.String `tfsdk:"url"`
	RepoCount types.Int64  `tfsdk:"repo_count"`
}

func newAzureDevOpsProjectDataSource() datasource.DataSource {
	return &azureDevOpsProjectDataSource{}
}

func (d *azureDevOpsProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsProjectDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsProject
}

func (d *azureDevOpsProjectDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsProjectDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureDevOpsProjectDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "RSC project ID (UUID). Exactly one of `id` or `name` must be set.",
			},
			keyName: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyID)),
				},
				Description: "Project name. Exactly one of `id` or `name` must be set.",
			},
			keyNativeID: schema.StringAttribute{
				Computed:    true,
				Description: "Azure DevOps project native ID.",
			},
			keyOrgID: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "RSC ID of the organization the project belongs to. May be set when looking up by " +
					"`name` to disambiguate a project name shared across organizations.",
			},
			keyOrgName: schema.StringAttribute{
				Computed:    true,
				Description: "Name of the organization the project belongs to.",
			},
			keyURL: schema.StringAttribute{
				Computed:    true,
				Description: "Azure DevOps project URL.",
			},
			keyRepoCount: schema.Int64Attribute{
				Computed:    true,
				Description: "Number of repositories in the project.",
			},
		},
	}
}

func (d *azureDevOpsProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsProjectDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureDevOpsProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsProjectDataSource.Read")

	var config azureDevOpsProjectDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var project gqldevops.AzureProject
	if !config.ID.IsNull() {
		id, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid project ID", err.Error())
			return
		}
		project, err = devops.Wrap(polarisClient).AzureProjectByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read Azure DevOps project", err.Error())
			return
		}
	} else {
		name := config.Name.ValueString()
		candidates, err := devops.Wrap(polarisClient).AzureProjectsByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps project", err.Error())
			return
		}

		// The exact name match is done server-side. Project names are only
		// unique within an organization, so narrow to that organization when
		// org_id is set.
		var matches []gqldevops.AzureProject
		for _, candidate := range candidates {
			if !config.OrgID.IsNull() && candidate.OrgID.String() != config.OrgID.ValueString() {
				continue
			}
			matches = append(matches, candidate)
		}

		switch len(matches) {
		case 0:
			res.Diagnostics.AddError("Azure DevOps project not found", fmt.Sprintf("no project with name %q", name))
			return
		case 1:
			project = matches[0]
		default:
			res.Diagnostics.AddError("Multiple Azure DevOps projects found",
				fmt.Sprintf("%d projects are named %q; set org_id to disambiguate", len(matches), name))
			return
		}
	}

	config.ID = types.StringValue(project.ID.String())
	config.Name = types.StringValue(project.Name)
	config.NativeID = types.StringValue(project.NativeID)
	config.OrgID = types.StringValue(project.OrgID.String())
	config.OrgName = types.StringValue(project.OrgName)
	config.URL = types.StringValue(project.URL)
	config.RepoCount = types.Int64Value(int64(project.RepoCount))

	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}
