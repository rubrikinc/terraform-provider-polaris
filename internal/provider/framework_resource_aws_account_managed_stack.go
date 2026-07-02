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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

// managedStackOnboardTimeout bounds the trigger -> poll-until-connected ->
// complete sequence when the request context has no deadline of its own.
const managedStackOnboardTimeout = 60 * time.Minute

const frameworkResourceAwsAccountManagedStackDescription = `
The ´polaris_aws_account_managed_stack´ resource completes the RSC-managed AWS
onboarding (BaaS) flow after the CloudFormation stack has been deployed. It
waits for the account's features to connect and then completes onboarding.

The account, its features and its regions were all defined by the
´polaris_aws_account_managed´ resource, so this resource only needs the RSC
account ID and the deployed stack ARN. Wire the stack ARN in so onboarding runs
only after the stack exists.

-> **Note:** Destroying this resource removes it from state only; the account
   remains onboarded in RSC.
`

var (
	_ resource.Resource                = &awsAccountManagedStackResource{}
	_ resource.ResourceWithImportState = &awsAccountManagedStackResource{}
)

type awsAccountManagedStackResource struct {
	client *client
}

type awsAccountManagedStackModel struct {
	ID                 types.String `tfsdk:"id"`
	AccountID          types.String `tfsdk:"account_id"`
	StackARN           types.String `tfsdk:"stack_arn"`
	PermissionsVersion types.String `tfsdk:"permissions_version"`
}

func newAwsAccountManagedStackResource() resource.Resource {
	return &awsAccountManagedStackResource{}
}

func (r *awsAccountManagedStackResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Metadata")
	res.TypeName = req.ProviderTypeName + "_aws_account_managed_stack"
}

func (r *awsAccountManagedStackResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Schema")

	res.Schema = schema.Schema{
		Description: description(frameworkResourceAwsAccountManagedStackDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyAccountID: schema.StringAttribute{
				Required: true,
				Description: "RSC cloud account ID (UUID) from the `polaris_aws_account_managed` resource. " +
					"Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			keyStackARN: schema.StringAttribute{
				Required: true,
				Description: "ARN of the deployed CloudFormation stack. Reference `aws_cloudformation_stack.<name>.id` " +
					"so onboarding runs after the stack is created. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			keyPermissionsVersion: schema.StringAttribute{
				Required: true,
				Description: "Permission-set version from the `polaris_aws_account_managed` resource. When it " +
					"changes, RSC has raised a permission version and the CloudFormation stack has been redeployed; " +
					"the resource then re-completes onboarding (notifies RSC and waits for the features to reconnect).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *awsAccountManagedStackResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Configure")
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *awsAccountManagedStackResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Create")

	var plan awsAccountManagedStackModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	accountID, err := uuid.Parse(plan.AccountID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid account ID", err.Error())
		return
	}

	// Bound the trigger -> poll -> complete sequence.
	onboardCtx, cancel := context.WithTimeout(ctx, managedStackOnboardTimeout)
	defer cancel()

	if err := aws.Wrap(polarisClient).CompleteManagedAccountOnboarding(onboardCtx, accountID); err != nil {
		res.Diagnostics.AddError("Failed to complete RSC-managed AWS onboarding", err.Error())
		return
	}

	plan.ID = plan.AccountID
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *awsAccountManagedStackResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Read")

	var state awsAccountManagedStackModel
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

	if _, err := aws.Wrap(polarisClient).AccountByID(ctx, accountID); errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	} else if err != nil {
		res.Diagnostics.AddError("Failed to read RSC-managed AWS account", err.Error())
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *awsAccountManagedStackResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Update")

	var plan awsAccountManagedStackModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}
	var state awsAccountManagedStackModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	// account_id and stack_arn force replacement, so the only in-place change is
	// permissions_version. A change means RSC raised a permission version and the
	// CloudFormation stack has been redeployed with the updated permissions -
	// notify RSC and wait for the features to reconnect.
	if !plan.PermissionsVersion.Equal(state.PermissionsVersion) {
		polarisClient, err := r.client.polaris()
		if err != nil {
			res.Diagnostics.AddError("RSC client error", err.Error())
			return
		}
		accountID, err := uuid.Parse(plan.AccountID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid account ID", err.Error())
			return
		}

		onboardCtx, cancel := context.WithTimeout(ctx, managedStackOnboardTimeout)
		defer cancel()

		if err := aws.Wrap(polarisClient).CompleteManagedAccountUpdate(onboardCtx, accountID); err != nil {
			res.Diagnostics.AddError("Failed to complete RSC-managed AWS permissions update", err.Error())
			return
		}
	}

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *awsAccountManagedStackResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.Delete")
	// No-op: the account remains onboarded in RSC (teardown is handled out of band).
}

func (r *awsAccountManagedStackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "awsAccountManagedStackResource.ImportState")
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyID), req.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root(keyAccountID), req.ID)...)
}
