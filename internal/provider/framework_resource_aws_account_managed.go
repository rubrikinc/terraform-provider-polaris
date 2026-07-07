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
				Description: "Account name. Derived from the AWS account when not specified.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					"`features` is set. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
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
					"Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
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
	templateURL, err := aws.Wrap(polarisClient).PrepareManagedAccountUpdate(ctx, accountID)
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

	// All configurable inputs force replacement, so Update only re-runs the
	// register step defensively.
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
	if err := aws.Wrap(polarisClient).FinalizeManagedAccountDeletion(ctx, accountID); err != nil {
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
		featureNames = aws.BaaSDefaultFeatureNames()
	}

	regions, regionNames, regionDiags := resolveManagedRegions(ctx, m.Regions)
	diags.Append(regionDiags...)
	if diags.HasError() {
		return diags
	}

	artifacts, err := aws.Wrap(polarisClient).RegisterManagedAccount(ctx,
		aws.AccountWithName(m.Cloud.ValueString(), m.NativeID.ValueString(), m.Name.ValueString()), featureNames, regions)
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

	version, err := aws.Wrap(polarisClient).ManagedAccountPermissionsVersion(ctx, artifacts.AccountID)
	if err != nil {
		diags.AddError("Failed to read permission version", err.Error())
		return diags
	}

	m.ID = types.StringValue(artifacts.AccountID.String())
	m.Name = types.StringValue(artifacts.Name)
	m.Features = featureSet
	m.Regions = regionSet
	m.CloudFormationURL = types.StringValue(artifacts.CloudFormationURL)
	m.TemplateURL = types.StringValue(artifacts.TemplateURL)
	m.StackName = types.StringValue(artifacts.StackName)
	m.PermissionsVersion = types.StringValue(version)

	return diags
}

// resolveManagedRegions returns the typed regions and their canonical names for
// state. When the set is null/unknown/empty it defaults to the full
// BaaS-supported region set.
func resolveManagedRegions(ctx context.Context, set types.Set) ([]awsregions.Region, []string, diag.Diagnostics) {
	var diags diag.Diagnostics

	regionNames := func(regions []awsregions.Region) []string {
		names := make([]string, 0, len(regions))
		for _, region := range regions {
			names = append(names, region.Name())
		}
		return names
	}

	if set.IsNull() || set.IsUnknown() {
		regions := aws.BaaSSupportedRegions()
		return regions, regionNames(regions), diags
	}

	var names []string
	diags.Append(set.ElementsAs(ctx, &names, false)...)
	if diags.HasError() {
		return nil, nil, diags
	}
	if len(names) == 0 {
		regions := aws.BaaSSupportedRegions()
		return regions, regionNames(regions), diags
	}

	regions := make([]awsregions.Region, 0, len(names))
	for _, name := range names {
		region := awsregions.RegionFromName(name)
		if region == awsregions.RegionUnknown {
			diags.AddError("Invalid AWS region", fmt.Sprintf("unknown AWS region: %q", name))
			return nil, nil, diags
		}
		regions = append(regions, region)
	}
	return regions, regionNames(regions), diags
}
