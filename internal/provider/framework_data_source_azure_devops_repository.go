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

const dataSourceAzureDevOpsRepositoryDescription = `
The ´polaris_azure_devops_repository´ data source reads an Azure DevOps
repository from RSC. Look it up by ´id´ or by ´name´. The repository is the
snappable object.

Repository names are only unique within a project. When looking up by ´name´,
set ´project_id´ to disambiguate a name shared across projects; without it a
name matching more than one repository is an error.
`

var (
	_ datasource.DataSource              = &azureDevOpsRepositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsRepositoryDataSource{}
)

type azureDevOpsRepositoryDataSource struct {
	client *client
}

type azureDevOpsRepositoryDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	OrgID       types.String `tfsdk:"org_id"`
	OrgName     types.String `tfsdk:"org_name"`
	ProjectID   types.String `tfsdk:"project_id"`
	ProjectName types.String `tfsdk:"project_name"`
	URL         types.String `tfsdk:"url"`
	Size        types.Int64  `tfsdk:"size"`
}

func newAzureDevOpsRepositoryDataSource() datasource.DataSource {
	return &azureDevOpsRepositoryDataSource{}
}

func (d *azureDevOpsRepositoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsRepositoryDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsRepository
}

func (d *azureDevOpsRepositoryDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsRepositoryDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureDevOpsRepositoryDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "RSC repository ID (UUID). Exactly one of `id` or `name` must be set.",
			},
			keyName: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyID)),
				},
				Description: "Repository name. Exactly one of `id` or `name` must be set.",
			},
			keyOrgID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC ID of the organization the repository belongs to.",
			},
			keyOrgName: schema.StringAttribute{
				Computed:    true,
				Description: "Name of the organization the repository belongs to.",
			},
			keyProjectID: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "RSC ID of the project the repository belongs to. May be set when looking up by `name` " +
					"to disambiguate a repository name shared across projects.",
			},
			keyProjectName: schema.StringAttribute{
				Computed:    true,
				Description: "Name of the project the repository belongs to.",
			},
			keyURL: schema.StringAttribute{
				Computed:    true,
				Description: "Azure DevOps repository URL.",
			},
			keySize: schema.Int64Attribute{
				Computed:    true,
				Description: "Repository size in bytes.",
			},
		},
	}
}

func (d *azureDevOpsRepositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsRepositoryDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureDevOpsRepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsRepositoryDataSource.Read")

	var config azureDevOpsRepositoryDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var repo gqldevops.AzureRepository
	switch {
	case !config.ID.IsNull():
		id, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid repository ID", err.Error())
			return
		}
		repo, err = devops.Wrap(polarisClient).AzureRepositoryByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read Azure DevOps repository", err.Error())
			return
		}
	case !config.Name.IsNull():
		name := config.Name.ValueString()
		activeFilters := activeObjectFilters()
		objects, err := hierarchy.ObjectsByName[hierarchy.AzureDevOpsRepository](ctx, hierarchy.Wrap(polarisClient.GQL), name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up Azure DevOps repository", err.Error())
			return
		}

		// Repository names are only unique within a project, so an exact-name
		// lookup can return repositories from multiple projects. Resolve each
		// candidate and, when project_id is set, keep only the one in that
		// project.
		var matches []gqldevops.AzureRepository
		for _, obj := range objects {
			candidate, err := devops.Wrap(polarisClient).AzureRepositoryByID(ctx, obj.Object.ID)
			if err != nil {
				res.Diagnostics.AddError("Failed to read Azure DevOps repository", err.Error())
				return
			}
			if !config.ProjectID.IsNull() && candidate.ProjectID.String() != config.ProjectID.ValueString() {
				continue
			}
			matches = append(matches, candidate)
		}

		switch len(matches) {
		case 0:
			res.Diagnostics.AddError("Azure DevOps repository not found", "no repository with name "+name)
			return
		case 1:
			repo = matches[0]
		default:
			res.Diagnostics.AddError("Multiple Azure DevOps repositories found",
				fmt.Sprintf("%d repositories are named %q; set project_id to disambiguate", len(matches), name))
			return
		}
	}

	config.ID = types.StringValue(repo.ID.String())
	config.Name = types.StringValue(repo.Name)
	config.OrgID = types.StringValue(repo.OrgID.String())
	config.OrgName = types.StringValue(repo.OrgName)
	config.ProjectID = types.StringValue(repo.ProjectID.String())
	config.ProjectName = types.StringValue(repo.ProjectName)
	config.URL = types.StringValue(repo.URL)
	config.Size = types.Int64Value(repo.Size)

	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}
