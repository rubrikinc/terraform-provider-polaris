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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

~> **Warning:** Changing ´cloud´, ´native_id´ or ´tenant_domain´ forces the
organization to be replaced: it is destroyed and re-onboarded, which runs the
destroy step and therefore honours ´delete_snapshots_on_destroy´. A
´permission_groups´ change does not — re-run the onboarding script against the
organization to grant the new permissions before applying. See
´delete_snapshots_on_destroy´ for details.

~> **Note:** Set each feature's ´permissions´ field to the ´id´ of a
´polaris_azure_devops_permissions´ data source. When RSC changes the permissions
required for the feature the ´id´ changes, and applying the change notifies RSC
that the updated permissions have been granted. Re-run the onboarding script
against the organization to grant them before applying.

## Supported Configurations
The ´storage_type´ and ´exocompute_host_type´ can only be combined in the ways
listed below. Any other combination is rejected with an error explaining what
is allowed.

Onboarding supports:
  * ´BYOS´ storage with ´CUSTOMER_HOST´ exocompute.
  * ´BYOS´ storage with ´RUBRIK_HOST´ exocompute.
  * ´RCV´ storage with ´RUBRIK_HOST´ exocompute.

The only host type transition supported on an existing organization is
´CUSTOMER_HOST´ to ´RUBRIK_HOST´, which requires ´exocompute_region´ to be set.
Any other host or storage type change requires re-onboarding the organization
(destroy and re-create).

In-place field updates are supported for:
  * ´BYOS´ + ´CUSTOMER_HOST´: ´archival_location_id´ and
    ´exocompute_host_cloud_account_id´.
  * ´BYOS´ + ´RUBRIK_HOST´: ´archival_location_id´.

Any other field update requires re-onboarding the organization. ´RCV´
organizations are immutable apart from their ´feature´ permission groups.

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
	_ resource.Resource                = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithConfigure   = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithIdentity    = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithImportState = &azureDevOpsOrganizationResource{}
	_ resource.ResourceWithModifyPlan  = &azureDevOpsOrganizationResource{}
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

type azureDevOpsOrganizationIdentityModel struct {
	ID types.String `tfsdk:"id"`
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
				Description: fmt.Sprintf("Azure cloud type. %s. Default value is `PUBLIC`. Changing it forces "+
					"a new resource to be created.", possibleValues([]string{cloudTypePublic})),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(cloudTypePublic),
				},
			},
			keyExocomputeHostType: schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("Type of exocompute host. %s. `CUSTOMER_HOST` requires "+
					"`exocompute_host_cloud_account_id`; `RUBRIK_HOST` requires `exocompute_region`.",
					possibleValues([]gqldevops.HostType{gqldevops.HostTypeCustomer, gqldevops.HostTypeRubrik})),
				Validators: []validator.String{
					stringvalidator.OneOf(string(gqldevops.HostTypeCustomer), string(gqldevops.HostTypeRubrik)),
				},
			},
			keyStorageType: schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("Type of backup storage. %s. `BYOS` (Bring Your Own Storage) requires "+
					"`archival_location_id`; `RCV` (Rubrik Cloud Vault) is auto-provisioned.",
					possibleValues([]gqldevops.StorageType{gqldevops.StorageTypeBYOS, gqldevops.StorageTypeRCV})),
				Validators: []validator.String{
					stringvalidator.OneOf(string(gqldevops.StorageTypeBYOS), string(gqldevops.StorageTypeRCV)),
				},
			},
			keyArchivalLocationID: schema.StringAttribute{
				Optional:    true,
				Description: "Archival location ID for backups. Required when `storage_type` is `BYOS`.",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyExocomputeHostCloudAccountID: schema.StringAttribute{
				Optional: true,
				Description: "RSC cloud account ID providing exocompute. Required when `exocompute_host_type` is " +
					"`CUSTOMER_HOST`.",
				Validators: []validator.String{
					isUUID(),
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
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Delete the organization's snapshots when the resource is destroyed. Default value " +
					"is `false`.",
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
				Description: "RSC features to enable for the organization. At least one is required when onboarding.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Required: true,
							Description: fmt.Sprintf("Feature name. %s.",
								possibleValues([]string{core.FeatureAzureDevOpsRepositoryProtection.Name})),
							Validators: []validator.String{
								stringvalidator.OneOf(core.FeatureAzureDevOpsRepositoryProtection.Name),
							},
						},
						keyPermissionGroups: schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: fmt.Sprintf("Permission groups to enable for the feature. At least one is "+
								"required. %s. Re-run the onboarding script against the organization to grant the "+
								"new permissions before applying a change.",
								possibleValues(devops.AzureSupportedPermissionGroupNames())),
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.ValueStringsAre(stringvalidator.OneOf(devops.AzureSupportedPermissionGroupNames()...)),
							},
						},
						keyPermissions: schema.StringAttribute{
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will " +
								"notify RSC that the permissions for the feature have been updated. Use this field " +
								"with the `polaris_azure_devops_permissions` data source.",
							Validators: []validator.String{
								isNotWhiteSpace(),
							},
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

// ModifyPlan validates the rules and guards against replacement silently
// deleting snapshots. The replacement-causing fields, cloud, native_id, and
// tenant_domain, force replacement via their schema plan modifiers; the guard
// re-derives which of them changed because that is not exposed to ModifyPlan.
func (r *azureDevOpsOrganizationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, res *resource.ModifyPlanResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.ModifyPlan")

	// Destroy: no plan-time policy applies. Delete honours the prior state's
	// delete_snapshots_on_destroy on its own.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	// BYOS requires archival_location_id and RCV forbids it.
	if v := plan.StorageType; !v.IsUnknown() && !v.IsNull() {
		switch v.ValueString() {
		case string(gqldevops.StorageTypeBYOS):
			if plan.ArchivalLocationID.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyArchivalLocationID),
					"Missing archival_location_id", "archival_location_id is required when storage_type is BYOS.")
			}
		case string(gqldevops.StorageTypeRCV):
			if !plan.ArchivalLocationID.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyArchivalLocationID),
					"Unexpected archival_location_id", "archival_location_id must not be set when storage_type is RCV; RCV auto-provisions storage.")
			}
		}
	}

	// CUSTOMER_HOST requires exocompute_host_cloud_account_id and RUBRIK_HOST
	// requires exocompute_region.
	if v := plan.ExocomputeHostType; !v.IsUnknown() && !v.IsNull() {
		switch v.ValueString() {
		case string(gqldevops.HostTypeCustomer):
			if plan.ExocomputeHostCloudAccountID.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyExocomputeHostCloudAccountID),
					"Missing exocompute_host_cloud_account_id", "exocompute_host_cloud_account_id is required when exocompute_host_type is CUSTOMER_HOST.")
			}
			if !plan.ExocomputeRegion.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyExocomputeRegion),
					"Unexpected exocompute_region", "exocompute_region must not be set when exocompute_host_type is CUSTOMER_HOST.")
			}
		case string(gqldevops.HostTypeRubrik):
			if plan.ExocomputeRegion.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyExocomputeRegion),
					"Missing exocompute_region", "exocompute_region is required when exocompute_host_type is RUBRIK_HOST.")
			}
			if !plan.ExocomputeHostCloudAccountID.IsNull() {
				res.Diagnostics.AddAttributeError(path.Root(keyExocomputeHostCloudAccountID),
					"Unexpected exocompute_host_cloud_account_id", "exocompute_host_cloud_account_id must not be set when exocompute_host_type is RUBRIK_HOST.")
			}
		}
	}

	// Customer-hosted compute with RCV storage is an unsupported combination.
	if h, s := plan.ExocomputeHostType, plan.StorageType; !h.IsUnknown() && !h.IsNull() && !s.IsUnknown() && !s.IsNull() &&
		h.ValueString() == string(gqldevops.HostTypeCustomer) && s.ValueString() == string(gqldevops.StorageTypeRCV) {
		res.Diagnostics.AddError(
			"Unsupported host and storage combination",
			"Customer-hosted compute (CUSTOMER_HOST) with RCV storage is not supported. Use BYOS storage with "+
				"CUSTOMER_HOST, or RUBRIK_HOST with RCV.")
	}

	// Check feature specific permission groups.
	if !plan.Feature.IsNull() && !plan.Feature.IsUnknown() {
		var features []featureWithPermissionsModel
		res.Diagnostics.Append(plan.Feature.ElementsAs(ctx, &features, false)...)
		if res.Diagnostics.HasError() {
			return
		}

		for _, feature := range features {
			if feature.Name.IsUnknown() || feature.PermissionGroups.IsNull() || feature.PermissionGroups.IsUnknown() {
				continue
			}
			allowed, ok := azureDevOpsFeaturePermissionGroups[feature.Name.ValueString()]
			if !ok {
				continue
			}

			var groups []string
			res.Diagnostics.Append(feature.PermissionGroups.ElementsAs(ctx, &groups, false)...)
			for _, group := range groups {
				if _, valid := allowed[group]; !valid {
					res.Diagnostics.AddAttributeError(path.Root(keyFeature),
						"Invalid permission group for feature", fmt.Sprintf("permission group %q is not supported by feature %q.", group, feature.Name.ValueString()))
				}
			}
		}
	}
	if res.Diagnostics.HasError() {
		return
	}

	// No replacement policy applies to a brand-new resource.
	if req.State.Raw.IsNull() {
		return
	}

	var state azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	// cloud, native_id and tenant_domain force replacement via their schema
	// plan modifiers. Re-derive which of them changed for the snapshot guard
	// below, since the accumulated RequiresReplace is not exposed to ModifyPlan.
	var changed []string
	if !plan.Cloud.IsUnknown() && !state.Cloud.Equal(plan.Cloud) {
		changed = append(changed, keyCloud)
	}
	if !state.NativeID.Equal(plan.NativeID) {
		changed = append(changed, keyNativeID)
	}
	if !state.TenantDomain.Equal(plan.TenantDomain) {
		changed = append(changed, keyTenantDomain)
	}
	if len(changed) == 0 {
		return
	}

	// Replacement destroys the old organization before creating the new one,
	// and Delete honours the PRIOR state's delete_snapshots_on_destroy. Guard
	// on that value so an immutable-field edit cannot silently delete
	// snapshots.
	if state.DeleteSnapshotsOnDestroy.ValueBool() {
		res.Diagnostics.AddError(
			"Replacement would delete snapshots",
			fmt.Sprintf("Changing %s forces the organization to be re-onboarded (destroy then create), and "+
				"delete_snapshots_on_destroy is true in state, so the destroy step would delete all of the "+
				"organization's snapshots. Set delete_snapshots_on_destroy to false and apply that change "+
				"first, then make this change.", strings.Join(changed, ", ")),
		)
		return
	}

	res.Diagnostics.AddWarning(
		"Organization will be re-onboarded",
		fmt.Sprintf("Changing %s forces the organization to be re-onboarded (destroy then create). "+
			"delete_snapshots_on_destroy is false, so snapshots are retained.", strings.Join(changed, ", ")),
	)
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

	features, diags := toFeaturesWithPermissions(ctx, plan.Feature)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	params := gqldevops.AddAzureCloudAccountParams{
		OrganizationNativeIDs: []string{plan.NativeID.ValueString()},
		TenantDomain:          plan.TenantDomain.ValueString(),
		Cloud:                 toAzureCloud(plan.Cloud.ValueString()),
		Features:              stripPermissions(features),
		HostType:              gqldevops.HostType(plan.ExocomputeHostType.ValueString()),
		StorageType:           gqldevops.StorageType(plan.StorageType.ValueString()),
	}
	if v := plan.ArchivalLocationID.ValueString(); v != "" {
		locID, err := uuid.Parse(v)
		if err != nil {
			res.Diagnostics.AddError("Invalid archival location ID", err.Error())
		} else {
			params.BackupLocationID = &locID
		}
	}
	if v := plan.ExocomputeHostCloudAccountID.ValueString(); v != "" {
		hostID, err := uuid.Parse(v)
		if err != nil {
			res.Diagnostics.AddError("Invalid exocompute host cloud account ID", err.Error())
		} else {
			params.ExocomputeCloudAccountID = &hostID
		}
	}
	if v := plan.ExocomputeRegion.ValueString(); v != "" {
		region := azureregions.RegionFromName(v)
		params.ExocomputeRegion = &region
	}
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

	fromAzureOrganization(&plan, orgs[0])
	res.Diagnostics.Append(res.State.Set(ctx, plan)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: plan.ID}
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

	perms, err := devops.Wrap(polarisClient).ListOrgPermissions(ctx, id)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization permissions", err.Error())
		return
	}

	features, diags := toFeaturesWithPermissions(ctx, state.Feature)
	res.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	featureSet, diags := fromFeaturesWithPermissions(attachPermissions(perms.ToFeatures(), features))
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	state.Feature = featureSet

	fromAzureOrganization(&state, org)
	res.Diagnostics.Append(res.State.Set(ctx, state)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: state.ID}
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

func (r *azureDevOpsOrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "azureDevOpsOrganizationResource.Update")

	var plan, state azureDevOpsOrganizationModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	// The fields UpdateAzureCloudAccount can change in place, broken out
	// individually so the backend's transition rules can be enforced below.
	hostTypeChanged := !plan.ExocomputeHostType.Equal(state.ExocomputeHostType)
	storageTypeChanged := !plan.StorageType.Equal(state.StorageType)
	locationChanged := !plan.ArchivalLocationID.Equal(state.ArchivalLocationID)
	exoAccountChanged := !plan.ExocomputeHostCloudAccountID.Equal(state.ExocomputeHostCloudAccountID)
	exoRegionChanged := !plan.ExocomputeRegion.Equal(state.ExocomputeRegion)
	updatableFieldsChanged := hostTypeChanged || storageTypeChanged || locationChanged || exoAccountChanged || exoRegionChanged

	// Enforce the backend's transition rules here rather than in ModifyPlan.
	// terraform destroy with pending config changes still hands ModifyPlan a
	// non-null plan, so validating there would block the destroy. Update runs
	// only on an in-place change, never on destroy.
	switch {
	case state.StorageType.ValueString() == string(gqldevops.StorageTypeRCV) && updatableFieldsChanged:
		// RCV organizations are immutable; the update endpoint rejects the whole
		// request. Permission group changes go through UpgradeAzureCloudAccount
		// and are unaffected.
		res.Diagnostics.AddError(
			"RCV organization cannot be updated",
			"An RCV organization's exocompute host type, storage type, archival location and exocompute "+
				"settings cannot be changed in place; only permission groups can be updated. To change these "+
				"settings, destroy and re-onboard the organization.")
	case state.StorageType.ValueString() == string(gqldevops.StorageTypeBYOS) && plan.StorageType.ValueString() == string(gqldevops.StorageTypeRCV):
		// The backend rejects a BYOS to RCV switch.
		res.Diagnostics.AddAttributeError(path.Root(keyStorageType),
			"Unsupported storage type change",
			"Switching storage_type from BYOS to RCV is not supported. To move to RCV storage, destroy and "+
				"re-onboard the organization.")
	case state.ExocomputeHostType.ValueString() == string(gqldevops.HostTypeRubrik) &&
		state.StorageType.ValueString() == string(gqldevops.StorageTypeBYOS) &&
		(hostTypeChanged || storageTypeChanged || exoAccountChanged || exoRegionChanged):
		// For a Rubrik-hosted BYOS organization the update endpoint only applies
		// the archival location; any other change is silently dropped by the
		// backend, so reject it here.
		res.Diagnostics.AddError(
			"Unsupported update for Rubrik-hosted BYOS organization",
			"For a Rubrik-hosted (RUBRIK_HOST) BYOS organization only archival_location_id can be changed in "+
				"place. To change host type, storage type or exocompute settings, destroy and re-onboard the "+
				"organization.")
	}
	if res.Diagnostics.HasError() {
		return
	}

	// UpdateAzureCloudAccount only changes the host/storage fields; permission
	// groups go through UpgradeAzureCloudAccount below. A plan that reaches Update
	// with only permission groups changed must not call the update endpoint.
	// Call it only when a host/storage field changed.
	if updatableFieldsChanged {
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
	}

	// Permission group changes are applied in place by acknowledging the new
	// permissions on the feature. The caller is expected to have re-run the
	// onboarding script to grant them, mirroring the create-time trust model.
	if !plan.Feature.Equal(state.Feature) {
		features, diags := toFeaturesWithPermissions(ctx, plan.Feature)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		if err := devops.Wrap(polarisClient).UpgradeAzureCloudAccount(ctx, gqldevops.UpgradeAzureCloudAccountParams{
			OrganizationID:    id,
			FeaturesToUpgrade: stripPermissions(features),
		}); err != nil {
			res.Diagnostics.AddError("Failed to upgrade Azure DevOps organization permissions", err.Error())
			return
		}
	}

	org, err := devops.Wrap(polarisClient).AzureOrganizationByID(ctx, id)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure DevOps organization", err.Error())
		return
	}

	fromAzureOrganization(&plan, org)
	res.Diagnostics.Append(res.State.Set(ctx, plan)...)

	identity := azureDevOpsOrganizationIdentityModel{ID: plan.ID}
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

	var identity azureDevOpsOrganizationIdentityModel
	if req.ID != "" {
		// Import by plain resource ID.
		id, err := uuid.Parse(req.ID)
		if err != nil {
			res.Diagnostics.AddError("Invalid import ID", err.Error())
			return
		}
		identity = azureDevOpsOrganizationIdentityModel{ID: types.StringValue(id.String())}
	} else {
		// Import by identity block (id only).
		res.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if res.Diagnostics.HasError() {
			return
		}
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), identity.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyDeleteSnapshotsOnDestroy), false)...)
	res.Diagnostics.Append(res.Identity.Set(ctx, identity)...)
}

// azureDevOpsFeaturePermissionGroups maps each feature to the permission groups
// it supports. AZURE_DEVOPS_PROTECTION supports only BASIC; the repository and
// developer-collaboration features additionally support RECOVERY.
var azureDevOpsFeaturePermissionGroups = map[string]map[string]struct{}{
	core.FeatureAzureDevOpsProtection.Name: {
		string(core.PermissionGroupBasic): {},
	},
	core.FeatureAzureDevOpsRepositoryProtection.Name: {
		string(core.PermissionGroupBasic):    {},
		string(core.PermissionGroupRecovery): {},
	},
	core.FeatureAzureDevOpsDeveloperCollaborationProtection.Name: {
		string(core.PermissionGroupBasic):    {},
		string(core.PermissionGroupRecovery): {},
	},
}

const (
	cloudTypePublic = "PUBLIC"
	cloudTypeChina  = "CHINA"
	cloudTypeUSGov  = "USGOV"
)

func toAzureCloud(cloudType string) gqlazure.Cloud {
	switch cloudType {
	case cloudTypeChina:
		return gqlazure.ChinaCloud
	case cloudTypeUSGov:
		return gqlazure.USGovCloud
	default:
		return gqlazure.PublicCloud
	}
}

func fromAzureCloud(cloud gqlazure.Cloud) string {
	switch cloud {
	case gqlazure.ChinaCloud:
		return cloudTypeChina
	case gqlazure.USGovCloud:
		return cloudTypeUSGov
	default:
		return cloudTypePublic
	}
}

func fromAzureOrganization(m *azureDevOpsOrganizationModel, org gqldevops.AzureOrganization) {
	m.ID = types.StringValue(org.ID.String())
	m.NativeID = types.StringValue(org.NativeID)
	m.TenantDomain = types.StringValue(org.TenantDomain)
	m.Cloud = types.StringValue(fromAzureCloud(org.Cloud))
	m.ConnectionStatus = types.StringValue(string(org.ConnectionStatus))
	m.ProjectCount = types.Int64Value(int64(org.ProjectCount))
	m.RepoCount = types.Int64Value(int64(org.RepoCount))
	if org.LastRefreshTime != nil {
		m.LastRefreshTime = types.StringValue(org.LastRefreshTime.Format(time.RFC3339))
	} else {
		m.LastRefreshTime = types.StringNull()
	}

	// RUBRIK_HOST carries an exocompute region, CUSTOMER_HOST carries an
	// exocompute cloud account.
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

	// BYOS carries a backup location, RCV auto-provisions storage and takes no
	// backup location.
	if org.BackupLocation != nil && org.BackupLocation.StorageType == gqldevops.StorageTypeBYOS {
		m.StorageType = types.StringValue(string(gqldevops.StorageTypeBYOS))
		m.ArchivalLocationID = types.StringValue(org.BackupLocation.ArchivalGroupID.String())
	} else {
		m.StorageType = types.StringValue(string(gqldevops.StorageTypeRCV))
		m.ArchivalLocationID = types.StringNull()
	}
}
