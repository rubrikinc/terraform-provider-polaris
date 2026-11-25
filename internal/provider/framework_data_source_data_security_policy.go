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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/dspm"
	gqldspm "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/dspm"
)

const dataSourceDataSecurityPolicyDescription = `
The ´polaris_data_security_policy´ data source is used to access information
about a data security policy in RSC. A data security policy is looked up using
either the policy ID or the name.
`

var _ datasource.DataSource = &dataSecurityPolicyDataSource{}

type dataSecurityPolicyDataSource struct {
	client *client
}

type dataSecurityPolicyModel struct {
	ID              types.String  `tfsdk:"id"`
	PolicyID        types.String  `tfsdk:"policy_id"`
	Name            types.String  `tfsdk:"name"`
	Description     types.String  `tfsdk:"description"`
	Category        types.String  `tfsdk:"category"`
	Severity        types.String  `tfsdk:"severity"`
	Enabled         types.Bool    `tfsdk:"enabled"`
	Predefined      types.Bool    `tfsdk:"predefined"`
	Filter          []filterModel `tfsdk:"filter"`
	ThresholdFilter []filterModel `tfsdk:"threshold_filter"`
}

func newDataSecurityPolicyDataSource() datasource.DataSource {
	return &dataSecurityPolicyDataSource{}
}

func (d *dataSecurityPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyDataSecurityPolicy
}

// computedConditionBlockSchema returns a computed condition block schema for
// data sources.
func computedConditionBlockSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Filter condition.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				keyFilterType: schema.StringAttribute{
					Computed:    true,
					Description: "Filter type.",
				},
				keyRelationship: schema.StringAttribute{
					Computed:    true,
					Description: "Comparison operator.",
				},
				keyValues: schema.ListAttribute{
					ElementType: types.StringType,
					Computed:    true,
					Description: "Filter values.",
				},
			},
		},
	}
}

// computedFilterBlockSchema returns a computed filter block schema for data
// sources.
func computedFilterBlockSchema(description string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				keyOp: schema.StringAttribute{
					Computed:    true,
					Description: "Logical operator (AND or OR).",
				},
			},
			Blocks: map[string]schema.Block{
				keyCondition: computedConditionBlockSchema(),
				keyGroup: schema.ListNestedBlock{
					Description: "Nested filter group.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							keyOp: schema.StringAttribute{
								Computed:    true,
								Description: "Logical operator (AND or OR).",
							},
						},
						Blocks: map[string]schema.Block{
							keyCondition: computedConditionBlockSchema(),
						},
					},
				},
			},
		},
	}
}

func (d *dataSecurityPolicyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceDataSecurityPolicyDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "Data security policy ID (UUID).",
			},
			keyPolicyID: schema.StringAttribute{
				Optional:    true,
				Description: "Data security policy ID (UUID).",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "Name of the data security policy.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot(keyPolicyID)),
					isNotWhiteSpace(),
				},
			},
			keyDescription: schema.StringAttribute{
				Computed:    true,
				Description: "Description of the data security policy.",
			},
			keyCategory: schema.StringAttribute{
				Computed:    true,
				Description: "Category of the data security policy.",
			},
			keySeverity: schema.StringAttribute{
				Computed:    true,
				Description: "Severity of the data security policy.",
			},
			keyEnabled: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the data security policy is enabled.",
			},
			keyPredefined: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the data security policy is predefined.",
			},
		},
		Blocks: map[string]schema.Block{
			keyFilter:          computedFilterBlockSchema("Filter configuration."),
			keyThresholdFilter: computedFilterBlockSchema("Threshold filter configuration."),
		},
	}
}

func (d *dataSecurityPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *dataSecurityPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "dataSecurityPolicyDataSource.Read")

	var config dataSecurityPolicyModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var policy gqldspm.Policy
	if !config.PolicyID.IsNull() {
		id, err := uuid.Parse(config.PolicyID.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Invalid policy ID", err.Error())
			return
		}

		policy, err = dspm.Wrap(polarisClient).PolicyByID(ctx, id)
		if err != nil {
			res.Diagnostics.AddError("Failed to read data security policy", err.Error())
			return
		}
	} else {
		policy, err = dspm.Wrap(polarisClient).PolicyByName(ctx, config.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError("Failed to read data security policy", err.Error())
			return
		}
	}

	state := dataSecurityPolicyModel{
		ID:          types.StringValue(policy.ID.String()),
		PolicyID:    types.StringValue(policy.ID.String()),
		Name:        types.StringValue(policy.Name),
		Description: types.StringValue(policy.Description),
		Category:    types.StringValue(string(policy.Category)),
		Severity:    types.StringValue(string(policy.Severity)),
		Enabled:     types.BoolValue(policy.Enabled),
		Predefined:  types.BoolValue(policy.Predefined),
	}

	f, diags := fromGroupConfig(ctx, policy.Filter)
	res.Diagnostics.Append(diags...)
	if f != nil {
		state.Filter = []filterModel{*f}
	}

	tf, diags := fromGroupConfig(ctx, policy.ThresholdFilter)
	res.Diagnostics.Append(diags...)
	if tf != nil {
		state.ThresholdFilter = []filterModel{*tf}
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}
