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
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAzureDevOpsPermissionsDescription = `
The ´polaris_azure_devops_permissions´ data source returns the permissions RSC
requires for a single Azure DevOps feature and permission groups, along with the
version of each of its permission groups.

The ´permissions´ field is the permission document returned verbatim by RSC as a
raw JSON string. Use ´jsondecode´ to inspect it. It describes the Azure DevOps
permissions the feature needs, and is the aggregate of the permissions across
the feature's permission groups.

The ´permission_group_versions´ field tracks the version of each permission
group. The ´id´ is a SHA-256 hash of the feature, permissions and permission
group versions, so it changes whenever RSC updates the permissions required for
the feature. This makes it convenient for triggering a re-apply of the
onboarding script when the required permissions change.

## Permission Groups
Following is a list of features and their applicable permission groups.

´AZURE_DEVOPS_REPOSITORY_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.
`

var (
	_ datasource.DataSource              = &azureDevOpsPermissionsDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsPermissionsDataSource{}
)

type azureDevOpsPermissionsDataSource struct {
	client *client
}

type azureDevOpsPermissionsModel struct {
	ID                      types.String `tfsdk:"id"`
	Feature                 types.String `tfsdk:"feature"`
	PermissionGroups        types.Set    `tfsdk:"permission_groups"`
	Permissions             types.String `tfsdk:"permissions"`
	PermissionGroupVersions types.Map    `tfsdk:"permission_group_versions"`
}

func newAzureDevOpsPermissionsDataSource() datasource.DataSource {
	return &azureDevOpsPermissionsDataSource{}
}

func (d *azureDevOpsPermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsPermissionsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsPermissions
}

func (d *azureDevOpsPermissionsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsPermissionsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureDevOpsPermissionsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed: true,
				Description: "SHA-256 hash of the feature, permissions and permission group versions returned. " +
					"Changes when RSC updates the permissions required for the feature.",
			},
			keyFeature: schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("RSC Azure DevOps feature to look up permissions for. %s.",
					possibleValues(devops.AzureSupportedFeatureNames())),
				Validators: []validator.String{
					stringvalidator.OneOf(devops.AzureSupportedFeatureNames()...),
				},
			},
			keyPermissionGroups: schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: fmt.Sprintf("Permission groups to look up for the feature. %s.",
					possibleValues(devops.AzureSupportedPermissionGroupNames())),
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(devops.AzureSupportedPermissionGroupNames()...)),
				},
			},
			keyPermissions: schema.StringAttribute{
				Computed: true,
				Description: "The permissions required by the feature, as a raw JSON document returned verbatim by " +
					"RSC. Use `jsondecode` to inspect it.",
			},
			keyPermissionGroupVersions: schema.MapAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "Map of permission group name to its permission version.",
			},
		},
	}
}

func (d *azureDevOpsPermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsPermissionsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureDevOpsPermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsPermissionsDataSource.Read")

	var config azureDevOpsPermissionsModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var groups []string
	res.Diagnostics.Append(config.PermissionGroups.ElementsAs(ctx, &groups, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	feature := core.Feature{Name: config.Feature.ValueString()}
	for _, group := range groups {
		feature = feature.WithPermissionGroups(core.PermissionGroup(group))
	}

	if err := devops.AzureCheckFeature(feature); err != nil {
		res.Diagnostics.AddError("Unsupported Azure DevOps feature or permission group", err.Error())
		return
	}

	perms, err := devops.Wrap(polarisClient).ListPermissions(ctx, []core.Feature{feature})
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps permissions", err.Error())
		return
	}
	if len(perms.FeaturePermissions) != 1 {
		res.Diagnostics.AddError("Unexpected RSC response for Azure DevOps permissions",
			fmt.Sprintf("expected exactly 1 feature in response for %q, got %d", feature.Name, len(perms.FeaturePermissions)))
		return
	}
	featurePerm := perms.FeaturePermissions[0]

	hash := sha256.New()
	hash.Write([]byte(feature.Name))
	hash.Write([]byte(featurePerm.Permissions))
	for _, groupVersion := range featurePerm.PermissionGroupVersions {
		hash.Write([]byte(groupVersion.PermissionGroup))
		hash.Write([]byte(strconv.Itoa(groupVersion.Version)))
	}

	groupVersionValues := make(map[string]attr.Value, len(featurePerm.PermissionGroupVersions))
	for _, groupVersion := range featurePerm.PermissionGroupVersions {
		groupVersionValues[string(groupVersion.PermissionGroup)] = types.Int64Value(int64(groupVersion.Version))
	}
	groupVersionMap, diags := types.MapValue(types.Int64Type, groupVersionValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := azureDevOpsPermissionsModel{
		ID:                      types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		Feature:                 config.Feature,
		PermissionGroups:        config.PermissionGroups,
		Permissions:             types.StringValue(featurePerm.Permissions),
		PermissionGroupVersions: groupVersionMap,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
