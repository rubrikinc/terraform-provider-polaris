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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/dspm"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqldspm "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/dspm"
)

const resourceDataSecurityPolicyDescription = `
The ´polaris_data_security_policy´ resource is used to create and manage data
security policies in RSC.
`

var (
	_ resource.Resource                = &dataSecurityPolicyResource{}
	_ resource.ResourceWithImportState = &dataSecurityPolicyResource{}
)

type dataSecurityPolicyResource struct {
	client *client
}

type dataSecurityPolicyResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	Name            types.String  `tfsdk:"name"`
	Description     types.String  `tfsdk:"description"`
	Category        types.String  `tfsdk:"category"`
	Severity        types.String  `tfsdk:"severity"`
	Enabled         types.Bool    `tfsdk:"enabled"`
	Predefined      types.Bool    `tfsdk:"predefined"`
	Filter          []filterModel `tfsdk:"filter"`
	ThresholdFilter []filterModel `tfsdk:"threshold_filter"`
}

func newDataSecurityPolicyResource() resource.Resource {
	return &dataSecurityPolicyResource{}
}

func (r *dataSecurityPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyDataSecurityPolicy
}

// conditionBlockSchema returns the schema for a condition block, used in both
// filter and group blocks.
func conditionBlockSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Filter condition.",
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				keyFilterType: schema.StringAttribute{
					Required:    true,
					Description: "Filter type (e.g. SECURITY_DOCUMENT_SENSITIVITY).",
				},
				keyRelationship: schema.StringAttribute{
					Required:    true,
					Description: "Comparison operator (e.g. IS, IS_NOT, CONTAINS).",
				},
				keyValues: schema.ListAttribute{
					ElementType: types.StringType,
					Required:    true,
					Description: "Filter values.",
				},
			},
		},
	}
}

// filterBlockSchema returns the schema for a filter or threshold_filter block.
func filterBlockSchema(description string, sizeValidator validator.List) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		Validators:  []validator.List{sizeValidator},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				keyOp: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("AND"),
					Description: "Logical operator (AND or OR). Defaults to AND.",
					Validators: []validator.String{
						stringvalidator.OneOf("AND", "OR"),
					},
				},
			},
			Blocks: map[string]schema.Block{
				keyCondition: conditionBlockSchema(),
				keyGroup: schema.ListNestedBlock{
					Description: "Nested filter group.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							keyOp: schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("AND"),
								Description: "Logical operator (AND or OR). Defaults to AND.",
								Validators: []validator.String{
									stringvalidator.OneOf("AND", "OR"),
								},
							},
						},
						Blocks: map[string]schema.Block{
							keyCondition: conditionBlockSchema(),
						},
					},
				},
			},
		},
	}
}

func (r *dataSecurityPolicyResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceDataSecurityPolicyDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Data security policy ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyName: schema.StringAttribute{
				Required:    true,
				Description: "Name of the data security policy.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyDescription: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the data security policy.",
			},
			keyCategory: schema.StringAttribute{
				Required:    true,
				Description: "Category of the data security policy. Valid values are MISPLACED, OVEREXPOSED, REDUNDANT and UNPROTECTED.",
				Validators: []validator.String{
					stringvalidator.OneOf("MISPLACED", "OVEREXPOSED", "REDUNDANT", "UNPROTECTED"),
				},
			},
			keySeverity: schema.StringAttribute{
				Required:    true,
				Description: "Severity of the data security policy. Valid values are LOW, MEDIUM, HIGH and CRITICAL.",
				Validators: []validator.String{
					stringvalidator.OneOf("LOW", "MEDIUM", "HIGH", "CRITICAL"),
				},
			},
			keyEnabled: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the data security policy is enabled.",
			},
			keyPredefined: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the data security policy is predefined.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			keyFilter:          filterBlockSchema("Filter configuration for the data security policy.", listvalidator.SizeBetween(1, 1)),
			keyThresholdFilter: filterBlockSchema("Threshold filter configuration for the data security policy.", listvalidator.SizeAtMost(1)),
		},
	}
}

func (r *dataSecurityPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *dataSecurityPolicyResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Create")

	var plan dataSecurityPolicyResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	if len(plan.Filter) != 1 {
		res.Diagnostics.AddError("Invalid filter", "Exactly one filter block is required.")
		return
	}
	filter, diags := toGroupConfig(ctx, plan.Filter[0])
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	input := gqldspm.CreateInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Category:    gqldspm.Category(plan.Category.ValueString()),
		Severity:    gqldspm.Severity(plan.Severity.ValueString()),
		Filter:      filter,
	}

	switch len(plan.ThresholdFilter) {
	case 0:
		// No threshold filter, nothing to do.
	case 1:
		tf, diags := toGroupConfig(ctx, plan.ThresholdFilter[0])
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		input.ThresholdFilter = &tf
	default:
		res.Diagnostics.AddError("Invalid threshold filter", "At most one threshold_filter block is allowed.")
		return
	}

	policyID, err := dspm.Wrap(polarisClient).CreatePolicy(ctx, input)
	if err != nil {
		res.Diagnostics.AddError("Failed to create data security policy", err.Error())
		return
	}

	// The create API always creates an enabled policy. If the user
	// requested disabled, issue a follow-up update.
	if !plan.Enabled.ValueBool() {
		enabled := false
		if err := dspm.Wrap(polarisClient).UpdatePolicy(ctx, gqldspm.UpdateInput{
			ID:      policyID,
			Enabled: &enabled,
		}); err != nil {
			res.Diagnostics.AddError("Failed to disable data security policy after create", err.Error())
			return
		}
	}

	policy, err := dspm.Wrap(polarisClient).PolicyByID(ctx, policyID)
	if err != nil {
		res.Diagnostics.AddError("Failed to read data security policy after create", err.Error())
		return
	}

	// Only update computed fields from the read-back. The filter and other
	// user-provided fields stay as the plan specified to avoid inconsistent
	// results from normalization.
	plan.ID = types.StringValue(policy.ID.String())
	plan.Predefined = types.BoolValue(policy.Predefined)
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *dataSecurityPolicyResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Read")

	var state dataSecurityPolicyResourceModel
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
		res.Diagnostics.AddError("Invalid policy ID", err.Error())
		return
	}

	policy, err := dspm.Wrap(polarisClient).PolicyByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		res.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to read data security policy", err.Error())
		return
	}

	res.Diagnostics.Append(r.policyToModel(ctx, &state, policy)...)
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *dataSecurityPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Update")

	var plan dataSecurityPolicyResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	var state dataSecurityPolicyResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	name := plan.Name.ValueString()
	desc := plan.Description.ValueString()
	cat := gqldspm.Category(plan.Category.ValueString())
	sev := gqldspm.Severity(plan.Severity.ValueString())
	enabled := plan.Enabled.ValueBool()

	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid policy ID", err.Error())
		return
	}

	input := gqldspm.UpdateInput{
		ID:          id,
		Name:        &name,
		Description: &desc,
		Category:    &cat,
		Severity:    &sev,
		Enabled:     &enabled,
	}

	switch len(plan.Filter) {
	case 1:
		filter, diags := toGroupConfig(ctx, plan.Filter[0])
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		input.Filter = &filter
	default:
		res.Diagnostics.AddError("Invalid filter", "Exactly one filter block is required.")
		return
	}

	switch len(plan.ThresholdFilter) {
	case 0:
		// Omitting the threshold filter from the update payload clears it
		// when the filter field is present.
	case 1:
		tf, diags := toGroupConfig(ctx, plan.ThresholdFilter[0])
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		input.ThresholdFilter = &tf
	default:
		res.Diagnostics.AddError("Invalid threshold filter", "At most one threshold_filter block is allowed.")
		return
	}

	if err := dspm.Wrap(polarisClient).UpdatePolicy(ctx, input); err != nil {
		res.Diagnostics.AddError("Failed to update data security policy", err.Error())
		return
	}

	policy, err := dspm.Wrap(polarisClient).PolicyByID(ctx, id)
	if err != nil {
		res.Diagnostics.AddError("Failed to read data security policy after update", err.Error())
		return
	}

	res.Diagnostics.Append(r.policyToModel(ctx, &state, policy)...)
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func (r *dataSecurityPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.Delete")

	var state dataSecurityPolicyResourceModel
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
		res.Diagnostics.AddError("Invalid policy ID", err.Error())
		return
	}

	err = dspm.Wrap(polarisClient).DeletePolicy(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		return
	}
	if err != nil {
		res.Diagnostics.AddError("Failed to delete data security policy", err.Error())
	}
}

func (r *dataSecurityPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyResource.ImportState")

	if _, err := uuid.Parse(req.ID); err != nil {
		res.Diagnostics.AddError("Invalid import ID",
			"Expected a valid UUID as the import ID.")
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root(keyID), req, res)
}

func (r *dataSecurityPolicyResource) policyToModel(ctx context.Context, model *dataSecurityPolicyResourceModel, policy gqldspm.Policy) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(policy.ID.String())
	model.Name = types.StringValue(policy.Name)
	model.Description = types.StringValue(policy.Description)
	model.Category = types.StringValue(string(policy.Category))
	model.Severity = types.StringValue(string(policy.Severity))
	model.Enabled = types.BoolValue(policy.Enabled)
	model.Predefined = types.BoolValue(policy.Predefined)

	f, d := fromGroupConfig(ctx, policy.Filter)
	diags.Append(d...)
	if f != nil {
		model.Filter = []filterModel{*f}
	} else {
		model.Filter = nil
	}

	tf, d := fromGroupConfig(ctx, policy.ThresholdFilter)
	diags.Append(d...)
	if tf != nil {
		model.ThresholdFilter = []filterModel{*tf}
	} else {
		model.ThresholdFilter = nil
	}

	return diags
}
