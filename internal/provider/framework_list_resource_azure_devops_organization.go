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
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const listResourceAzureDevOpsOrganizationDescription = `
The ´polaris_azure_devops_organization´ list resource lists Azure DevOps
organizations onboarded to RSC. Results can be filtered by ´native_id´, the
Azure DevOps organization name shown in the organization's URL (e.g. ´my-org´
in https://dev.azure.com/my-org).

The ´delete_snapshots_on_destroy´ lifecycle setting is not returned by RSC and
is left null in list results; it defaults to ´false´ on the resource.

The ´permissions´ field on each feature block is a client-side signal not
stored in RSC and is left null in list results.
`

var (
	_ list.ListResource              = &azureDevOpsOrganizationListResource{}
	_ list.ListResourceWithConfigure = &azureDevOpsOrganizationListResource{}
)

type azureDevOpsOrganizationListResource struct {
	client *client
}

type azureDevOpsOrganizationListConfigModel struct {
	NativeID types.String `tfsdk:"native_id"`
}

func newAzureDevOpsOrganizationListResource() list.ListResource {
	return &azureDevOpsOrganizationListResource{}
}

func (r *azureDevOpsOrganizationListResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationListResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsOrganization
}

func (r *azureDevOpsOrganizationListResource) ListResourceConfigSchema(ctx context.Context, _ list.ListResourceSchemaRequest, res *list.ListResourceSchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationListResource.ListResourceConfigSchema")

	res.Schema = listschema.Schema{
		Description: description(listResourceAzureDevOpsOrganizationDescription),
		Attributes: map[string]listschema.Attribute{
			keyNativeID: listschema.StringAttribute{
				Optional: true,
				Description: "Filter organizations by native ID. The native ID is the Azure DevOps " +
					"organization name shown in the organization's URL (e.g., `my-org` from " +
					"https://dev.azure.com/my-org). Matches the organization whose native ID equals the " +
					"given value.",
			},
		},
	}
}

func (r *azureDevOpsOrganizationListResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationListResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *azureDevOpsOrganizationListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	tflog.Trace(ctx, "azureDevOpsOrganizationListResource.List")

	var config azureDevOpsOrganizationListConfigModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		diags.AddError("RSC client error", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	// Enumerate the organizations via the hierarchy inventory to build the
	// result identities. This is a lightweight id/name walk; full organization
	// detail is only fetched below when the caller requests the resource. For
	// organizations the hierarchy name equals the native ID.
	objects, err := hierarchy.ObjectsByType[hierarchy.AzureDevOpsOrganization](ctx, hierarchy.Wrap(polarisClient.GQL),
		hierarchy.WorkloadAllSubHierarchyType, activeObjectFilters()...)
	if err != nil {
		diags.AddError("Failed to list Azure DevOps organizations", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	nativeID := config.NativeID.ValueString()
	objects = slices.DeleteFunc(objects, func(obj hierarchy.AzureDevOpsOrganization) bool {
		return nativeID != "" && obj.Object.Name != nativeID
	})

	stream.Results = func(push func(list.ListResult) bool) {
		for i, obj := range objects {
			if int64(i) >= req.Limit {
				return
			}

			result := req.NewListResult(ctx)
			result.DisplayName = obj.Object.Name

			identity := azureDevOpsOrganizationIdentityModel{
				ID: types.StringValue(obj.Object.ID.String()),
			}
			result.Diagnostics.Append(result.Identity.Set(ctx, identity)...)
			if result.Diagnostics.HasError() {
				push(result)
				return
			}

			if req.IncludeResource {
				org, err := devops.Wrap(polarisClient).AzureOrganizationByID(ctx, obj.Object.ID)
				if err != nil {
					result.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
					push(result)
					return
				}
				perms, err := devops.Wrap(polarisClient).ListOrgPermissions(ctx, obj.Object.ID)
				if err != nil {
					result.Diagnostics.AddError("Failed to read Azure DevOps organization permissions", err.Error())
					push(result)
					return
				}

				featureSet, diags := fromFeaturesWithPermissions(attachPermissions(perms.ToFeatures(), nil))
				result.Diagnostics.Append(diags...)
				if result.Diagnostics.HasError() {
					push(result)
					return
				}

				// The delete_snapshots_on_destroy lifecycle setting is not
				// returned by RSC and is left null.
				model := azureDevOpsOrganizationModel{
					DeleteSnapshotsOnDestroy: types.BoolNull(),
					Feature:                  featureSet,
				}
				fromAzureOrganization(&model, org)

				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
				if result.Diagnostics.HasError() {
					push(result)
					return
				}
			}

			if !push(result) {
				return
			}
		}
	}
}
