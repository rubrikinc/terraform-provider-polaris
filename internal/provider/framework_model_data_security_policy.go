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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	gqldspm "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/dspm"
)

// conditionModel maps to a single filter condition in the schema.
type conditionModel struct {
	FilterType   types.String `tfsdk:"filter_type"`
	Relationship types.String `tfsdk:"relationship"`
	Values       types.List   `tfsdk:"values"`
}

// groupModel maps to a nested filter group in the schema.
type groupModel struct {
	Operator  types.String     `tfsdk:"op"`
	Condition []conditionModel `tfsdk:"condition"`
}

// filterModel maps to the top-level filter block in the schema.
type filterModel struct {
	Operator  types.String     `tfsdk:"op"`
	Condition []conditionModel `tfsdk:"condition"`
	Group     []groupModel     `tfsdk:"group"`
}

// toGroupConfig converts a filterModel to the SDK GroupConfig type.
func toGroupConfig(ctx context.Context, model filterModel) (gqldspm.GroupConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	gc := gqldspm.GroupConfig{
		Op: gqldspm.LogicalOp(model.Operator.ValueString()),
	}

	for _, c := range model.Condition {
		node, d := toConditionNode(ctx, c)
		diags.Append(d...)
		if diags.HasError() {
			return gqldspm.GroupConfig{}, diags
		}
		gc.Filters = append(gc.Filters, node)
	}

	for _, g := range model.Group {
		var subGC gqldspm.GroupConfig
		subGC.Op = gqldspm.LogicalOp(g.Operator.ValueString())
		for _, c := range g.Condition {
			node, d := toConditionNode(ctx, c)
			diags.Append(d...)
			if diags.HasError() {
				return gqldspm.GroupConfig{}, diags
			}
			subGC.Filters = append(subGC.Filters, node)
		}
		gc.Filters = append(gc.Filters, gqldspm.Node{GroupConfig: &subGC})
	}

	return gc, diags
}

func toConditionNode(ctx context.Context, c conditionModel) (gqldspm.Node, diag.Diagnostics) {
	var values []string
	diags := c.Values.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return gqldspm.Node{}, diags
	}

	return gqldspm.Node{
		Config: &gqldspm.Config{
			Type:         gqldspm.FilterType(c.FilterType.ValueString()),
			Relationship: gqldspm.Relationship(c.Relationship.ValueString()),
			Values:       values,
		},
	}, diags
}

// fromGroupConfig converts a *GroupConfig into a filterModel.
func fromGroupConfig(ctx context.Context, gc *gqldspm.GroupConfig) (*filterModel, diag.Diagnostics) {
	if gc == nil {
		return nil, nil
	}

	var diags diag.Diagnostics

	model := &filterModel{
		Operator: types.StringValue(string(gc.Op)),
	}

	for _, node := range gc.Filters {
		switch {
		case node.Config != nil:
			values, d := types.ListValueFrom(ctx, types.StringType, node.Config.Values)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			model.Condition = append(model.Condition, conditionModel{
				FilterType:   types.StringValue(string(node.Config.Type)),
				Relationship: types.StringValue(string(node.Config.Relationship)),
				Values:       values,
			})
		case node.GroupConfig != nil:
			g := groupModel{
				Operator: types.StringValue(string(node.GroupConfig.Op)),
			}
			for _, subNode := range node.GroupConfig.Filters {
				if subNode.Config == nil {
					diags.AddError("Unsupported filter nesting",
						"Filter contains a nested group deeper than two levels, which is not supported.")
					return nil, diags
				}
				values, d := types.ListValueFrom(ctx, types.StringType, subNode.Config.Values)
				diags.Append(d...)
				if diags.HasError() {
					return nil, diags
				}
				g.Condition = append(g.Condition, conditionModel{
					FilterType:   types.StringValue(string(subNode.Config.Type)),
					Relationship: types.StringValue(string(subNode.Config.Relationship)),
					Values:       values,
				})
			}
			model.Group = append(model.Group, g)
		}
	}

	return model, diags
}
