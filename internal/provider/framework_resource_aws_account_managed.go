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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	awsregions "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"
)

const frameworkResourceAwsAccountManagedDescription = `
The ´polaris_aws_account_managed´ resource is the first step of the RSC-managed
AWS onboarding (BaaS) flow. It validates the AWS account with RSC, registers it,
and returns the CloudFormation template information needed to deploy the RSC
cross-account stack.

Everything about the account - its features and regions - is chosen here. After
this resource, deploy the CloudFormation stack (e.g. with the
´aws_cloudformation_stack´ resource of the AWS provider) using the exported
´template_url´ and ´stack_name´, then complete onboarding with the
´polaris_aws_account_managed_stack´ resource.

-> **Note:** Only the ´STANDARD´ (commercial) AWS cloud is supported. When
   ´features´ or ´regions´ are omitted, all BaaS-supported values are used.
`

var (
	_ resource.Resource                = &awsAccountManagedResource{}
	_ resource.ResourceWithImportState = &awsAccountManagedResource{}
	_ resource.ResourceWithModifyPlan  = &awsAccountManagedResource{}
)

type awsAccountManagedResource struct {
	client *client
}

type awsAccountManagedModel struct {
	ID                 types.String `tfsdk:"id"`
	NativeID           types.String `tfsdk:"native_id"`
	Name               types.String `tfsdk:"name"`
	Cloud              types.String `tfsdk:"cloud"`
	Features           types.Set    `tfsdk:"features"`
	Regions            types.Set    `tfsdk:"regions"`
	CloudFormationURL  types.String `tfsdk:"cloud_formation_url"`
	TemplateURL        types.String `tfsdk:"template_url"`
	StackName          types.String `tfsdk:"stack_name"`
	PermissionsVersion types.String `tfsdk:"permissions_version"`
}

func newAwsAccountManagedResource() resource.Resource {
	return &awsAccountManagedResource{}
}

func (r *awsAccountManagedResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Metadata")
	res.TypeName = req.ProviderTypeName + "_aws_account_managed"
}

func (r *awsAccountManagedResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Schema")

	res.Schema = schema.Schema{
		Description: description(frameworkResourceAwsAccountManagedDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyNativeID: schema.StringAttribute{
				Required:    true,
				Description: "AWS account ID. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Account name. Derived from the AWS account when not specified. Can be updated in place.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyCloud: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("STANDARD"),
				Description: "AWS cloud type. Only `STANDARD` is supported. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("STANDARD"),
				},
			},
			keyFeatures: schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "RSC features to onboard. When omitted, all BaaS-supported features are used: " +
					"`CLOUD_NATIVE_PROTECTION`, `RDS_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION` and `CLOUD_DISCOVERY`. " +
					"`CLOUD_DISCOVERY` is a prerequisite for the protection features and must be included when " +
					"`features` is set. When omitted, the account tracks the current default set and newly " +
					"released features are added in place. Adding a feature is applied in place (the " +
					"CloudFormation stack is redeployed via the `polaris_aws_account_managed_stack` resource); " +
					"removing a feature forces a new resource to be created.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(
						"CLOUD_NATIVE_PROTECTION", "RDS_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION", "CLOUD_DISCOVERY",
					)),
					setMustContain("CLOUD_DISCOVERY"),
				},
			},
			keyRegions: schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "AWS regions to protect. When omitted, all BaaS-supported regions are used. " +
					"Changing regions on an existing resource is not supported yet; recreate the resource to " +
					"change them.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			keyCloudFormationURL: schema.StringAttribute{
				Computed:    true,
				Description: "AWS console URL for creating the CloudFormation stack.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyTemplateURL: schema.StringAttribute{
				Computed:    true,
				Description: "CloudFormation template URL. Use with the `aws_cloudformation_stack` resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyStackName: schema.StringAttribute{
				Computed:    true,
				Description: "CloudFormation stack name generated by RSC. Use with the `aws_cloudformation_stack` resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyPermissionsVersion: schema.StringAttribute{
				Computed: true,
				Description: "Identifier of the account's permission-set version. It changes when RSC raises a " +
					"permission version (a feature becomes `MISSING_PERMISSIONS`). Wire it into the " +
					"`polaris_aws_account_managed_stack` resource so onboarding re-completes after the " +
					"CloudFormation stack is redeployed with the updated permissions.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ModifyPlan drives the feature-set behavior:
//
//   - When features is not set in config, the account tracks the current default
//     set (aws.ManagedAccountDefaultFeatureNames), which grows as new BaaS
//     features are released. The default is forced into the plan so a newly
//     released feature shows up as a diff instead of being frozen to prior state
//     (the Plugin Framework equivalent of an SDKv2 CustomizeDiff SetNew).
//   - Adding a feature is applied in place. The CloudFormation template and the
//     permission version change as a result, so those computed attributes are
//     set to unknown (SetNewComputed) to keep the plan consistent with apply.
//   - Removing a feature cannot be done in place (it needs the RSC feature
//     disable flow), so it forces replacement.
func (r *awsAccountManagedResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, res *resource.ModifyPlanResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.ModifyPlan")

	// Nothing to do on destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var config, plan awsAccountManagedModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	// When features is omitted, plan the current default set so the account
	// tracks newly released features.
	if config.Features.IsNull() {
		featureSet, d := types.SetValueFrom(ctx, types.StringType, aws.ManagedAccountDefaultFeatureNames())
		res.Diagnostics.Append(d...)
		if res.Diagnostics.HasError() {
			return
		}
		plan.Features = featureSet
		res.Diagnostics.Append(res.Plan.SetAttribute(ctx, path.Root(keyFeatures), featureSet)...)
	}

	// On create there is no prior state to compare against.
	if req.State.Raw.IsNull() {
		return
	}

	var state awsAccountManagedModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	// In-place region updates require RSC's BaaS edit flow
	// (finalizeAwsCloudAccountProtection with action UPDATE_REGIONS, then
	// completeBaasOnboarding), which is not yet available in the SDK. Block
	// region changes until a future release adds it, rather than issuing an
	// incorrect update.
	if !plan.Regions.Equal(state.Regions) {
		res.Diagnostics.AddError(
			"Region updates not supported yet",
			"Changing `regions` on an existing polaris_aws_account_managed resource is not supported in this "+
				"release; it will be added in a future release. To change regions now, recreate the resource "+
				"(for example with `terraform apply -replace=<address>`).",
		)
		return
	}

	var planFeatures, stateFeatures []string
	res.Diagnostics.Append(plan.Features.ElementsAs(ctx, &planFeatures, false)...)
	res.Diagnostics.Append(state.Features.ElementsAs(ctx, &stateFeatures, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	planSet := make(map[string]struct{}, len(planFeatures))
	for _, name := range planFeatures {
		planSet[name] = struct{}{}
	}
	stateSet := make(map[string]struct{}, len(stateFeatures))
	for _, name := range stateFeatures {
		stateSet[name] = struct{}{}
	}

	removed := false
	for name := range stateSet {
		if _, ok := planSet[name]; !ok {
			removed = true
			break
		}
	}
	added := false
	for name := range planSet {
		if _, ok := stateSet[name]; !ok {
			added = true
			break
		}
	}

	// Removing a feature cannot be done in place - force replacement. On replace
	// the computed attributes are recomputed on create, so nothing else to do.
	if removed {
		res.RequiresReplace = append(res.RequiresReplace, path.Root(keyFeatures))
		return
	}

	// Adding a feature changes the CloudFormation template and permission
	// version. Mark those computed attributes unknown so apply can write the new
	// values without a "provider produced inconsistent result" error.
	//
	// stack_name is deliberately NOT marked unknown: RSC reuses the same stack
	// name across feature edits (CloudFormation stack names are immutable). It
	// feeds aws_cloudformation_stack.name, which is ForceNew - marking it unknown
	// makes Terraform replace the stack, which cascades into replacing the
	// phase-2 polaris_aws_account_managed_stack resource. That destroy runs
	// RemoveManagedAccount (tearing the account down) before phase-1's in-place
	// update, which then re-creates the account under a new ID. Keeping stack_name
	// stable lets the template change apply as an in-place stack update instead.
	if added {
		res.Diagnostics.Append(res.Plan.SetAttribute(ctx, path.Root(keyTemplateURL), types.StringUnknown())...)
		res.Diagnostics.Append(res.Plan.SetAttribute(ctx, path.Root(keyCloudFormationURL), types.StringUnknown())...)
		res.Diagnostics.Append(res.Plan.SetAttribute(ctx, path.Root(keyPermissionsVersion), types.StringUnknown())...)
	}
}

func (r *awsAccountManagedResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Configure")
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *awsAccountManagedResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Create")

	var plan awsAccountManagedModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(r.register(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *awsAccountManagedResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Read")

	var state awsAccountManagedModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	accountID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid account ID", err.Error())
		return
	}

	// Refresh the name, which can change out of band and is reconciled in place.
	// AccountByID also serves as the existence check.
	//
	// Regions are intentionally not refreshed: in-place region changes are
	// blocked until the SDK supports the BaaS region-edit flow, so region drift
	// cannot be reconciled and refreshing it would only risk a persistent diff.
	account, err := aws.Wrap(polarisClient).AccountByID(ctx, accountID)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read RSC-managed AWS account", err.Error())
		return
	}
	state.Name = types.StringValue(account.Name)

	// Always refresh the permission-set version so it stays populated (never
	// null) and reflects RSC's current permission versions. It is deterministic,
	// so a healthy account shows no diff; a change means RSC raised a permission
	// version.
	version, err := aws.Wrap(polarisClient).ManagedAccountPermissionsVersion(ctx, accountID)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read permission version", err.Error())
		return
	}
	state.PermissionsVersion = types.StringValue(version)

	// On a required permissions upgrade (a feature is MISSING_PERMISSIONS),
	// refresh the CloudFormation template URL so the stack is redeployed and the
	// polaris_aws_account_managed_stack resource re-completes onboarding. When up
	// to date the signed template URL is left untouched to avoid churn.
	templateURL, err := aws.Wrap(polarisClient).UpdateManagedAccount(ctx, accountID)
	if err != nil {
		res.Diagnostics.AddError("Failed to check RSC-managed AWS account for updates", err.Error())
		return
	}
	if templateURL != "" {
		state.TemplateURL = types.StringValue(templateURL)
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *awsAccountManagedResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Update")

	var plan awsAccountManagedModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state awsAccountManagedModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	accountID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid account ID", err.Error())
		return
	}

	// native_id and cloud force replacement, feature removal is turned into a
	// replacement in ModifyPlan, and region changes are blocked in ModifyPlan
	// (in-place region updates need the RSC BaaS edit flow, which is not yet
	// available in the SDK). So Update only reconciles name and feature additions.

	// Name is reconciled first (a cheap, RSC-side rename) so that the feature
	// re-registration below sees the already-updated name.
	if !plan.Name.Equal(state.Name) {
		// The feature argument is unused by the name update path.
		if err := aws.Wrap(polarisClient).UpdateAccount(ctx, accountID, core.Feature{}, aws.Name(plan.Name.ValueString())); err != nil {
			res.Diagnostics.AddError("Failed to update RSC-managed AWS account name", err.Error())
			return
		}
	}

	// Feature additions are applied in place by re-registering the desired set
	// (validate + finalize). This produces a new CloudFormation template and
	// bumps the permission version, which drives the phase-2
	// polaris_aws_account_managed_stack resource to redeploy the stack and
	// finalize.
	if !plan.Features.Equal(state.Features) {
		res.Diagnostics.Append(r.register(ctx, &plan)...)
		if res.Diagnostics.HasError() {
			return
		}
		res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *awsAccountManagedResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.Delete")

	var state awsAccountManagedModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	accountID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid account ID", err.Error())
		return
	}

	// By now the features have been disabled (polaris_aws_account_managed_stack)
	// and the CloudFormation stack deleted, so finalize the account removal in
	// RSC. This is a no-op if the stack's deletion notifier already removed it.
	if err := aws.Wrap(polarisClient).RemoveManagedAccountFinalize(ctx, accountID); err != nil {
		res.Diagnostics.AddError("Failed to finalize RSC-managed AWS account deletion", err.Error())
	}
}

func (r *awsAccountManagedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "awsAccountManagedResource.ImportState")
	resource.ImportStatePassthroughID(ctx, path.Root(keyID), req, res)
}

// register runs validate + finalize and populates the model's computed
// attributes (RSC account ID and CloudFormation artifacts).
func (r *awsAccountManagedResource) register(ctx context.Context, m *awsAccountManagedModel) diag.Diagnostics {
	var diags diag.Diagnostics

	polarisClient, err := r.client.polaris()
	if err != nil {
		diags.AddError("RSC client error", err.Error())
		return diags
	}

	var featureNames []string
	if !m.Features.IsNull() && !m.Features.IsUnknown() {
		diags.Append(m.Features.ElementsAs(ctx, &featureNames, false)...)
		if diags.HasError() {
			return diags
		}
	}
	if len(featureNames) == 0 {
		featureNames = aws.ManagedAccountDefaultFeatureNames()
	}
	features := make([]core.Feature, 0, len(featureNames))
	for _, name := range featureNames {
		features = append(features, core.Feature{Name: name})
	}

	regionNames, regionDiags := resolveManagedRegions(ctx, m.Regions)
	diags.Append(regionDiags...)
	if diags.HasError() {
		return diags
	}

	accountID, stack, err := aws.Wrap(polarisClient).AddManagedAccount(ctx,
		aws.AccountWithName(m.Cloud.ValueString(), m.NativeID.ValueString(), m.Name.ValueString()), features,
		aws.Regions(regionNames...))
	if err != nil {
		diags.AddError("Failed to register RSC-managed AWS account", err.Error())
		return diags
	}

	featureSet, d := types.SetValueFrom(ctx, types.StringType, featureNames)
	diags.Append(d...)
	regionSet, d := types.SetValueFrom(ctx, types.StringType, regionNames)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	account, err := aws.Wrap(polarisClient).AccountByID(ctx, accountID)
	if err != nil {
		diags.AddError("Failed to read RSC-managed AWS account", err.Error())
		return diags
	}

	version, err := aws.Wrap(polarisClient).ManagedAccountPermissionsVersion(ctx, accountID)
	if err != nil {
		diags.AddError("Failed to read permission version", err.Error())
		return diags
	}

	m.ID = types.StringValue(accountID.String())
	m.Name = types.StringValue(account.Name)
	m.Features = featureSet
	m.Regions = regionSet
	m.CloudFormationURL = types.StringValue(stack.CloudFormationURL)
	m.TemplateURL = types.StringValue(stack.TemplateURL)
	m.StackName = types.StringValue(stack.StackName)
	m.PermissionsVersion = types.StringValue(version)

	return diags
}

// resolveManagedRegions validates and returns the canonical region names for
// state. When the set is null/unknown/empty it defaults to the full
// BaaS-supported region set.
func resolveManagedRegions(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	regionNames := func(regions []awsregions.Region) []string {
		names := make([]string, 0, len(regions))
		for _, region := range regions {
			names = append(names, region.Name())
		}
		return names
	}

	if set.IsNull() || set.IsUnknown() {
		return regionNames(aws.ManagedAccountSupportedRegions()), diags
	}

	var names []string
	diags.Append(set.ElementsAs(ctx, &names, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(names) == 0 {
		return regionNames(aws.ManagedAccountSupportedRegions()), diags
	}

	for _, name := range names {
		if awsregions.RegionFromName(name) == awsregions.RegionUnknown {
			diags.AddError("Invalid AWS region", fmt.Sprintf("unknown AWS region: %q", name))
			return nil, diags
		}
	}
	return names, diags
}
