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

const dataSourceGitHubRepositoryDescription = `
The ´polaris_github_repository´ data source reads a GitHub repository from RSC.
Look it up by ´id´ or by ´name´. The repository is the snappable object.

Repository names are only unique within an organization. When looking up by
´name´, set ´org_id´ to disambiguate a name shared across organizations;
without it a name matching more than one repository is an error.
`

var (
	_ datasource.DataSource              = &gitHubRepositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &gitHubRepositoryDataSource{}
)

type gitHubRepositoryDataSource struct {
	client *client
}

type gitHubRepositoryDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	OrgID   types.String `tfsdk:"org_id"`
	OrgName types.String `tfsdk:"org_name"`
	Size    types.Int64  `tfsdk:"size"`
}

func newGitHubRepositoryDataSource() datasource.DataSource {
	return &gitHubRepositoryDataSource{}
}

func (d *gitHubRepositoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "gitHubRepositoryDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyGitHubRepository
}

func (d *gitHubRepositoryDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "gitHubRepositoryDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceGitHubRepositoryDescription),
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
				Optional: true,
				Computed: true,
				Description: "RSC ID of the organization the repository belongs to. May be set when looking up by " +
					"`name` to disambiguate a repository name shared across organizations.",
			},
			keyOrgName: schema.StringAttribute{
				Computed:    true,
				Description: "Name of the organization the repository belongs to.",
			},
			keySize: schema.Int64Attribute{
				Computed:    true,
				Description: "Repository size in bytes.",
			},
		},
	}
}

func (d *gitHubRepositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "gitHubRepositoryDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *gitHubRepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "gitHubRepositoryDataSource.Read")

	var config gitHubRepositoryDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var repo gqldevops.GitHubRepository
	if !config.ID.IsNull() {
		id, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid repository ID", err.Error())
			return
		}
		repo, err = devops.Wrap(polarisClient).GitHubRepositoryByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read GitHub repository", err.Error())
			return
		}
	} else {
		name := config.Name.ValueString()
		candidates, err := devops.Wrap(polarisClient).GitHubRepositoriesByName(ctx, name,
			activeObjectFilters(hierarchy.Filter{Field: "NAME_EXACT_MATCH", Texts: []string{name}})...)
		if err != nil {
			res.Diagnostics.AddError("Failed to look up GitHub repository", err.Error())
			return
		}

		// The exact name match is done server-side. Repository names are only
		// unique within an organization, so narrow to that organization when
		// org_id is set.
		var matches []gqldevops.GitHubRepository
		for _, candidate := range candidates {
			if !config.OrgID.IsNull() && candidate.OrgID.String() != config.OrgID.ValueString() {
				continue
			}
			matches = append(matches, candidate)
		}

		switch len(matches) {
		case 0:
			res.Diagnostics.AddError("GitHub repository not found", fmt.Sprintf("no repository with name %q", name))
			return
		case 1:
			repo = matches[0]
		default:
			res.Diagnostics.AddError("Multiple GitHub repositories found",
				fmt.Sprintf("%d repositories are named %q; set org_id to disambiguate", len(matches), name))
			return
		}
	}

	config.ID = types.StringValue(repo.ID.String())
	config.Name = types.StringValue(repo.Name)
	config.OrgID = types.StringValue(repo.OrgID.String())
	config.OrgName = types.StringValue(repo.OrgName)
	config.Size = types.Int64Value(repo.Size)

	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}
