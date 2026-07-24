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
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	gqldevops "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/devops"
)

const dataSourceAzureDevOpsScriptDescription = `
The ´polaris_azure_devops_script´ data source generates the Azure DevOps
onboarding scripts for one or more organizations. Run the generated script
against the Azure DevOps organization to create the Rubrik group, grant the
Rubrik service principal read access, and assign a Basic license.

~> **Warning:** The Azure AD tenant and service principal referenced by
´tenant_domain´ must already be onboarded to RSC for the Azure DevOps use case.
Reading this data source fails if the tenant has not been onboarded.

The provider does not run the script — it only generates it. Run it out of band
with the Azure CLI signed in (´az login´) as a Project Collection Administrator
in each target organization; the script mints a short-lived Azure DevOps token
from that ´az´ session, so no personal access token is required.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the ´feature´ block.

´AZURE_DEVOPS_REPOSITORY_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.
`

var (
	_ datasource.DataSource              = &azureDevOpsScriptDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevOpsScriptDataSource{}
)

type azureDevOpsScriptDataSource struct {
	client *client
}

type azureDevOpsScriptModel struct {
	ID               types.String `tfsdk:"id"`
	TenantDomain     types.String `tfsdk:"tenant_domain"`
	Cloud            types.String `tfsdk:"cloud"`
	Feature          types.Set    `tfsdk:"feature"`
	OrgNativeIDs     types.Set    `tfsdk:"org_native_ids"`
	BashScript       types.String `tfsdk:"bash_script"`
	PowershellScript types.String `tfsdk:"powershell_script"`
}

func newAzureDevOpsScriptDataSource() datasource.DataSource {
	return &azureDevOpsScriptDataSource{}
}

func (d *azureDevOpsScriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsScriptDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsScript
}

func (d *azureDevOpsScriptDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsScriptDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureDevOpsScriptDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the generated scripts.",
			},
			keyTenantDomain: schema.StringAttribute{
				Required:    true,
				Description: "Azure AD tenant primary domain.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyCloud: schema.StringAttribute{
				Optional: true,
				Description: fmt.Sprintf("Azure cloud type. %s. Default value is `PUBLIC`.",
					possibleValues([]string{cloudTypePublic, cloudTypeChina, cloudTypeUSGov})),
				Validators: []validator.String{
					stringvalidator.OneOf(cloudTypePublic, cloudTypeChina, cloudTypeUSGov),
				},
			},
			keyOrgNativeIDs: schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Azure DevOps organization native identifiers, i.e. the organization names visible " +
					"in the Azure DevOps URL (e.g., \"my-org\" from https://dev.azure.com/my-org). The script is " +
					"generated for each organization in the set. At least one is required.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			keyBashScript: schema.StringAttribute{
				Computed:    true,
				Description: "The generated bash onboarding script (decoded).",
			},
			keyPowershellScript: schema.StringAttribute{
				Computed:    true,
				Description: "The generated PowerShell onboarding script (decoded).",
			},
		},
		Blocks: map[string]schema.Block{
			keyFeature: schema.SetNestedBlock{
				Description: "RSC features to include in the generated script. At least one is required.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Required:    true,
							Description: "Feature name.",
							Validators: []validator.String{
								isNotWhiteSpace(),
							},
						},
						keyPermissionGroups: schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: fmt.Sprintf("Permission groups to enable for the feature. At least one is "+
								"required. %s.", possibleValues(devops.AzureSupportedPermissionGroupNames())),
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.ValueStringsAre(stringvalidator.OneOf(devops.AzureSupportedPermissionGroupNames()...)),
							},
						},
					},
				},
			},
		},
	}
}

func (d *azureDevOpsScriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsScriptDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureDevOpsScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsScriptDataSource.Read")

	var config azureDevOpsScriptModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	features, diags := toFeatures(ctx, config.Feature)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	var orgNativeIDs []string
	res.Diagnostics.Append(config.OrgNativeIDs.ElementsAs(ctx, &orgNativeIDs, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	script, err := devops.Wrap(polarisClient).GenerateAzureOnboardingScript(ctx, gqldevops.GenerateAzureOnboardingScriptParams{
		TenantDomain:          config.TenantDomain.ValueString(),
		Cloud:                 toAzureCloud(config.Cloud.ValueString()),
		Features:              features,
		OrganizationNativeIDs: orgNativeIDs,
	})
	if err != nil {
		res.Diagnostics.AddError("Failed to generate Azure DevOps onboarding script", err.Error())
		return
	}

	bashScript, err := base64.StdEncoding.DecodeString(script.BashScript)
	if err != nil {
		res.Diagnostics.AddError("Failed to decode bash script", err.Error())
		return
	}
	powershellScript, err := base64.StdEncoding.DecodeString(script.PowershellScript)
	if err != nil {
		res.Diagnostics.AddError("Failed to decode PowerShell script", err.Error())
		return
	}

	sum := sha256.Sum256(append(bashScript, powershellScript...))
	config.ID = types.StringValue(fmt.Sprintf("%x", sum[:]))
	config.BashScript = types.StringValue(string(bashScript))
	config.PowershellScript = types.StringValue(string(powershellScript))

	res.Diagnostics.Append(res.State.Set(ctx, config)...)
}
