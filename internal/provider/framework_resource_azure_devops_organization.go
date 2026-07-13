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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/devops"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqldevops "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/devops"
	azureregions "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
)

// Provider-facing cloud values, mapped to gqlazure.Cloud before being sent
// to RSC.
const (
	cloudTypePublic = "PUBLIC"
	cloudTypeChina  = "CHINA"
	cloudTypeUSGov  = "USGOV"
)

const resourceAzureDevOpsOrganizationDescription = `
The ´polaris_azure_devops_organization´ resource onboards an Azure DevOps
organization to RSC using a customer-supplied application (non-OAuth).

Before creating this resource, register the customer application for the Azure
DevOps use case with a ´polaris_azure_service_principal´ resource using
´use_case = "AZURE_DEVOPS"´, then generate the onboarding script with the
´polaris_azure_devops_script´ data source and run it against the organization.
The provider does not run the script; see that data source for how to run it.

Each resource instance manages a single organization. Manage multiple
organizations with multiple instances or ´for_each´.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the ´feature´ block.

´AZURE_DEVOPS_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´AZURE_DEVOPS_REPOSITORY_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.

´AZURE_DEVOPS_DEVELOPER_COLLABORATION_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.

## Import

RSC does not return the ´cloud´ type or the enabled ´feature´ blocks for an
onboarded organization, so neither can be read back on import.

Import by RSC organization ID (UUID). Because ´cloud´ is not returned, supply it
in the ´identity´ block of an ´import´ block so the imported state is not
mislabeled (omit it to default to ´PUBLIC´):

´´´terraform
import {
  to = polaris_azure_devops_organization.org
  identity = {
    id    = "a1b2c3d4-1234-4c5b-9abc-0123456789ab"
    cloud = "CHINA"
  }
}
´´´

The plain string ID form (´terraform import ... <id>´) is also accepted and
defaults ´cloud´ to ´PUBLIC´.

´feature´ blocks cannot be imported. After import, declare the organization's
feature blocks in configuration; the first ´apply´ writes them into state and
subsequent plans are clean.
`

var (
	_ resource.Resource                = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithConfigure   = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithIdentity    = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithImportState = &azureDevOpsOrganizationResource{}
)

type azureDevOpsOrganizationResource struct {
	client *client
}

type azureDevOpsOrganizationModel struct {
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
	DeleteSnapshotsOnDestroy     types.Bool   `tfsdk:"delete_snapshots_on_destroy"`
	ConnectionStatus             types.String `tfsdk:"connection_status"`
	ProjectCount                 types.Int64  `tfsdk:"project_count"`
	RepoCount                    types.Int64  `tfsdk:"repo_count"`
	LastRefreshTime              types.String `tfsdk:"last_refresh_time"`
}

type azureDevOpsFeatureModel struct {
	Name             types.String `tfsdk:"name"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
}

type azureDevOpsOrganizationIdentityModel struct {
	ID    types.String `tfsdk:"id"`
	Cloud types.String `tfsdk:"cloud"`
}

func newAzureDevOpsOrganizationResource() resource.Resource {
	return &azureDevOpsOrganizationResource{}
}

func (r *azureDevOpsOrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureDevOpsOrganization
}

func (r *azureDevOpsOrganizationResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceAzureDevOpsOrganizationDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC organization ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyNativeID: schema.StringAttribute{
				Required: true,
				Description: "Azure DevOps organization native identifier. This is the organization name " +
					"visible in the Azure DevOps URL (e.g., \"my-org\" from https://dev.azure.com/my-org). " +
					"Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyTenantDomain: schema.StringAttribute{
				Required: true,
				Description: "Azure AD tenant primary domain. Changing this forces a new resource to be " +
					"created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyCloud: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(cloudTypePublic),
				Description: "Azure cloud type. One of `PUBLIC` (default), `CHINA` or `USGOV`. Changing this " +
					"forces a new resource to be created. RSC does not return the cloud type, so it is never " +
					"refreshed from RSC and drift is not detected; on import, supply it via the identity block.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(cloudTypePublic, cloudTypeChina, cloudTypeUSGov),
				},
			},
			keyExocomputeHostType: schema.StringAttribute{
				Required: true,
				Description: "Type of exocompute host. One of `CUSTOMER_HOST` (requires " +
					"`exocompute_host_cloud_account_id`) or `RUBRIK_HOST` (requires `exocompute_region`).",
				Validators: []validator.String{
					stringvalidator.OneOf(string(gqldevops.HostTypeCustomer), string(gqldevops.HostTypeRubrik)),
				},
			},
			keyStorageType: schema.StringAttribute{
				Required: true,
				Description: "Type of backup storage. One of `BYOS` (Bring Your Own Storage, requires " +
					"`archival_location_id`) or `RCV` (Rubrik Cloud Vault, auto-provisioned).",
				Validators: []validator.String{
					stringvalidator.OneOf(string(gqldevops.StorageTypeBYOS), string(gqldevops.StorageTypeRCV)),
				},
			},
			keyArchivalLocationID: schema.StringAttribute{
				Optional:    true,
				Description: "Archival location ID for backups. Required when `storage_type` is `BYOS`.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyExocomputeHostCloudAccountID: schema.StringAttribute{
				Optional: true,
				Description: "RSC cloud account ID providing exocompute. Required when `exocompute_host_type` is " +
					"`CUSTOMER_HOST`.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyExocomputeRegion: schema.StringAttribute{
				Optional: true,
				Description: "Azure region for Rubrik-hosted exocompute (e.g. `eastus`). Required when " +
					"`exocompute_host_type` is `RUBRIK_HOST`.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyDeleteSnapshotsOnDestroy: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Delete the organization's snapshots when the resource is destroyed.",
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
		Blocks: map[string]schema.Block{
			keyFeature: schema.SetNestedBlock{
				Description: "RSC features to enable for the organization. At least one is required. Features " +
					"are set only at onboarding: RSC does not return them and they cannot be updated, so they " +
					"are never refreshed and drift is not detected. Declare them in configuration, including " +
					"after import (an imported organization has no feature blocks until you add them).",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Required: true,
							Description: "Feature name. One of `AZURE_DEVOPS_PROTECTION`, " +
								"`AZURE_DEVOPS_REPOSITORY_PROTECTION` or " +
								"`AZURE_DEVOPS_DEVELOPER_COLLABORATION_PROTECTION`.",
							Validators: []validator.String{
								isNotWhiteSpace(),
							},
						},
						keyPermissionGroups: schema.SetAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Permission groups to enable for the feature. Empty enables all of the " +
								"feature's groups. See the resource description for the groups each feature " +
								"supports. Like `feature`, this is not returned by RSC and drift is not detected.",
						},
					},
				},
			},
		},
	}
}

func (r *azureDevOpsOrganizationResource) IdentitySchema(ctx context.Context, _ resource.IdentitySchemaRequest, res *resource.IdentitySchemaResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.IdentitySchema")

	res.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			keyID: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "RSC organization ID (UUID).",
			},
			keyCloud: identityschema.StringAttribute{
				OptionalForImport: true,
				Description: "Azure cloud type the organization was onboarded with. One of `PUBLIC` " +
					"(default), `CHINA` or `USGOV`. RSC does not return the cloud type, so the value is stored " +
					"as provided and not verified; omit it to default to `PUBLIC`.",
			},
		},
	}
}

func (r *azureDevOpsOrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *azureDevOpsOrganizationResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Create")

	var plan azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	params, diags := r.addParams(ctx, plan)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	orgs, err := devops.Wrap(polarisClient).AddAzureCloudAccount(ctx, params)
	if err != nil {
		res.Diagnostics.AddError("Failed to add Azure DevOps organization", err.Error())
		return
	}
	if len(orgs) == 0 {
		res.Diagnostics.AddError("Failed to add Azure DevOps organization", "no organization returned after onboarding")
		return
	}

	setStateFromOrg(&plan, orgs[0])
	res.Diagnostics.Append(res.State.Set(ctx, plan)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: plan.ID, Cloud: plan.Cloud}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *azureDevOpsOrganizationResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Read")

	var state azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid organization ID", err.Error())
		return
	}

	org, err := devops.Wrap(polarisClient).AzureOrganizationByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
		return
	}

	setStateFromOrg(&state, org)
	res.Diagnostics.Append(res.State.Set(ctx, state)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: state.ID, Cloud: state.Cloud}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *azureDevOpsOrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Update")

	var plan azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(plan.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid organization ID", err.Error())
		return
	}

	params := gqldevops.UpdateAzureCloudAccountParams{
		OrganizationID: id,
		HostType:       gqldevops.HostType(plan.ExocomputeHostType.ValueString()),
		StorageType:    gqldevops.StorageType(plan.StorageType.ValueString()),
	}
	if v := plan.ArchivalLocationID.ValueString(); v != "" {
		locID, err := uuid.Parse(v)
		if err != nil {
			res.Diagnostics.AddError("Invalid backup location ID", err.Error())
			return
		}
		params.BackupLocationID = &locID
	}
	if v := plan.ExocomputeHostCloudAccountID.ValueString(); v != "" {
		acctID, err := uuid.Parse(v)
		if err != nil {
			res.Diagnostics.AddError("Invalid exocompute cloud account ID", err.Error())
			return
		}
		params.ExocomputeCloudAccountID = &acctID
	}
	if v := plan.ExocomputeRegion.ValueString(); v != "" {
		region := azureregions.RegionFromName(v)
		params.ExocomputeRegion = &region
	}

	if err := devops.Wrap(polarisClient).UpdateAzureCloudAccount(ctx, params); err != nil {
		res.Diagnostics.AddError("Failed to update Azure DevOps organization", err.Error())
		return
	}

	org, err := devops.Wrap(polarisClient).AzureOrganizationByID(ctx, id)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
		return
	}

	setStateFromOrg(&plan, org)
	res.Diagnostics.Append(res.State.Set(ctx, plan)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: plan.ID, Cloud: plan.Cloud}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *azureDevOpsOrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Delete")

	var state azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid organization ID", err.Error())
		return
	}

	err = devops.Wrap(polarisClient).DeleteAzureCloudAccount(ctx, id, state.DeleteSnapshotsOnDestroy.ValueBool())
	if err != nil && !errors.Is(err, graphql.ErrNotFound) {
		res.Diagnostics.AddError("Failed to delete Azure DevOps organization", err.Error())
	}
}

func (r *azureDevOpsOrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.ImportState")

	// cloud and feature/permission_groups are not returned by RSC, so they
	// cannot be read back on import. cloud is seeded from the import identity
	// (defaulting to PUBLIC) so a subsequent read does not mislabel it; feature
	// blocks must be declared in configuration after import.
	var identity azureDevOpsOrganizationIdentityModel
	if req.ID != "" {
		// Import by plain resource ID; cloud defaults to PUBLIC.
		id, err := uuid.Parse(req.ID)
		if err != nil {
			res.Diagnostics.AddError("Invalid import ID", err.Error())
			return
		}
		identity = azureDevOpsOrganizationIdentityModel{
			ID:    types.StringValue(id.String()),
			Cloud: types.StringValue(cloudTypePublic),
		}
	} else {
		// Import by identity block (id and optional cloud).
		res.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if res.Diagnostics.HasError() {
			return
		}
		if identity.Cloud.ValueString() == "" {
			identity.Cloud = types.StringValue(cloudTypePublic)
		}
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), identity.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyCloud), identity.Cloud)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyDeleteSnapshotsOnDestroy), false)...)
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

// addParams builds the SDK add parameters from the plan.
func (r *azureDevOpsOrganizationResource) addParams(ctx context.Context, plan azureDevOpsOrganizationModel) (gqldevops.AddAzureCloudAccountParams, diag.Diagnostics) {
	var diags diag.Diagnostics

	features, featureDiags := toFeatures(ctx, plan.Feature)
	diags.Append(featureDiags...)

	params := gqldevops.AddAzureCloudAccountParams{
		OrganizationNativeIDs: []string{plan.NativeID.ValueString()},
		TenantDomain:          plan.TenantDomain.ValueString(),
		Cloud:                 azureDevOpsCloud(plan.Cloud.ValueString()),
		Features:              features,
		HostType:              gqldevops.HostType(plan.ExocomputeHostType.ValueString()),
		StorageType:           gqldevops.StorageType(plan.StorageType.ValueString()),
	}
	if v := plan.ArchivalLocationID.ValueString(); v != "" {
		locID, err := uuid.Parse(v)
		if err != nil {
			diags.AddError("Invalid backup location ID", err.Error())
		} else {
			params.BackupLocationID = &locID
		}
	}
	if v := plan.ExocomputeHostCloudAccountID.ValueString(); v != "" {
		acctID, err := uuid.Parse(v)
		if err != nil {
			diags.AddError("Invalid exocompute cloud account ID", err.Error())
		} else {
			params.ExocomputeCloudAccountID = &acctID
		}
	}
	if v := plan.ExocomputeRegion.ValueString(); v != "" {
		region := azureregions.RegionFromName(v)
		params.ExocomputeRegion = &region
	}

	return params, diags
}

// setStateFromOrg writes both the computed fields and the RSC-reconciled input
// fields from the organization into the model. Reconciling the input fields
// (host/storage type and their dependent identifiers) keeps out-of-band drift
// detectable and makes terraform import populate a complete state.
func setStateFromOrg(m *azureDevOpsOrganizationModel, org gqldevops.AzureOrganization) {
	m.ID = types.StringValue(org.ID.String())
	m.NativeID = types.StringValue(org.NativeID)
	m.TenantDomain = types.StringValue(org.TenantDomain)
	// cloud is intentionally not set here: RSC does not return the cloud type on
	// the organization read, so it is never reconciled and the prior state value
	// (from config, default, or import) is preserved.
	m.ConnectionStatus = types.StringValue(string(org.ConnectionStatus))
	m.ProjectCount = types.Int64Value(int64(org.ProjectCount))
	m.RepoCount = types.Int64Value(int64(org.RepoCount))
	if org.LastRefreshTime != nil {
		m.LastRefreshTime = types.StringValue(org.LastRefreshTime.Format(time.RFC3339))
	} else {
		m.LastRefreshTime = types.StringNull()
	}

	// Host type and its dependent identifier. RUBRIK_HOST carries an exocompute
	// region; CUSTOMER_HOST carries an exocompute cloud account.
	switch {
	case org.RubrikHostedExocompute != nil:
		m.ExocomputeHostType = types.StringValue(string(gqldevops.HostTypeRubrik))
		m.ExocomputeRegion = types.StringValue(org.RubrikHostedExocompute.Region.Name())
		m.ExocomputeHostCloudAccountID = types.StringNull()
	case org.CloudNativeExocompute != nil:
		m.ExocomputeHostType = types.StringValue(string(gqldevops.HostTypeCustomer))
		m.ExocomputeHostCloudAccountID = types.StringValue(org.CloudNativeExocompute.ID.String())
		m.ExocomputeRegion = types.StringNull()
	}

	// Storage type and its dependent identifier. BYOS carries a backup location;
	// RCV auto-provisions storage and takes no backup location.
	if org.BackupLocation != nil && org.BackupLocation.StorageType == gqldevops.StorageTypeBYOS {
		m.StorageType = types.StringValue(string(gqldevops.StorageTypeBYOS))
		m.ArchivalLocationID = types.StringValue(org.BackupLocation.ArchivalGroupID.String())
	} else {
		m.StorageType = types.StringValue(string(gqldevops.StorageTypeRCV))
		m.ArchivalLocationID = types.StringNull()
	}
}

// azureDevOpsFeatureAttrTypes returns the attribute types of the feature block,
// used to construct a correctly-typed null set for the feature attribute.
func azureDevOpsFeatureAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:             types.StringType,
		keyPermissionGroups: types.SetType{ElemType: types.StringType},
	}
}

// toFeatures converts the feature block set into SDK features.
func toFeatures(ctx context.Context, set types.Set) ([]core.Feature, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() {
		return nil, diags
	}

	var blocks []azureDevOpsFeatureModel
	diags.Append(set.ElementsAs(ctx, &blocks, false)...)
	if diags.HasError() {
		return nil, diags
	}

	features := make([]core.Feature, 0, len(blocks))
	for _, block := range blocks {
		var groups []string
		diags.Append(block.PermissionGroups.ElementsAs(ctx, &groups, false)...)

		permGroups := make([]core.PermissionGroup, 0, len(groups))
		for _, g := range groups {
			permGroups = append(permGroups, core.PermissionGroup(g))
		}
		features = append(features, core.Feature{
			Name:             block.Name.ValueString(),
			PermissionGroups: permGroups,
		})
	}

	return features, diags
}

// azureDevOpsCloud maps the provider-facing cloud value to the SDK cloud
// type.
func azureDevOpsCloud(cloudType string) gqlazure.Cloud {
	switch cloudType {
	case cloudTypeChina:
		return gqlazure.ChinaCloud
	case cloudTypeUSGov:
		return gqlazure.USGovCloud
	default:
		return gqlazure.PublicCloud
	}
}
