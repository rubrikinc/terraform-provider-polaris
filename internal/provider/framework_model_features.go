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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

type featureModel struct {
	Name             types.String `tfsdk:"name"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
}

func featureAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:             types.StringType,
		keyPermissionGroups: types.SetType{ElemType: types.StringType},
	}
}

func fromFeatures(features []core.Feature) (types.Set, diag.Diagnostics) {
	nullSet := types.SetNull(types.ObjectType{AttrTypes: featureAttrTypes()})

	featureValues := make([]attr.Value, 0, len(features))
	for _, feature := range features {
		groupValues := make([]attr.Value, 0, len(feature.PermissionGroups))
		for _, group := range feature.PermissionGroups {
			groupValues = append(groupValues, types.StringValue(string(group)))
		}
		groupSet, diags := types.SetValue(types.StringType, groupValues)
		if diags.HasError() {
			return nullSet, diags
		}

		object, diags := types.ObjectValue(featureAttrTypes(), map[string]attr.Value{
			keyName:             types.StringValue(feature.Name),
			keyPermissionGroups: groupSet,
		})
		if diags.HasError() {
			return nullSet, diags
		}

		featureValues = append(featureValues, object)
	}

	return types.SetValue(types.ObjectType{AttrTypes: featureAttrTypes()}, featureValues)
}

func toFeatures(ctx context.Context, featureSet types.Set) ([]core.Feature, diag.Diagnostics) {
	if featureSet.IsNull() {
		return nil, nil
	}

	var featureModels []featureModel
	diags := featureSet.ElementsAs(ctx, &featureModels, false)
	if diags.HasError() {
		return nil, diags
	}

	features := make([]core.Feature, 0, len(featureModels))
	for _, model := range featureModels {
		var groups []string
		diags.Append(model.PermissionGroups.ElementsAs(ctx, &groups, false)...)
		if diags.HasError() {
			return nil, diags
		}

		feature := core.Feature{Name: model.Name.ValueString()}
		for _, group := range groups {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group))
		}

		features = append(features, feature)
	}

	return features, diags
}

// featureWithPermissions extends Feature with the permissions signal.
type featureWithPermissions struct {
	core.Feature
	permissions string
}

// attachPermissions carries the permissions signal from featureWithPerms onto
// features, matched by name, returning the result. Features read from RSC never
// include the signal, so it is preserved from the prior state.
func attachPermissions(features []core.Feature, featureWithPerms []featureWithPermissions) []featureWithPermissions {
	m := make(map[string]string)
	for _, f := range featureWithPerms {
		m[f.Name] = f.permissions
	}

	fs := make([]featureWithPermissions, 0, len(features))
	for _, f := range features {
		fs = append(fs, featureWithPermissions{
			Feature: core.Feature{
				Name:             f.Name,
				PermissionGroups: f.PermissionGroups,
			},
			permissions: m[f.Name],
		})
	}

	return fs
}

// stripPermissions drops the permissions signal, returning the underlying
// features for use in SDK calls.
func stripPermissions(featureWithPerms []featureWithPermissions) []core.Feature {
	fs := make([]core.Feature, 0, len(featureWithPerms))
	for _, feature := range featureWithPerms {
		fs = append(fs, feature.Feature)
	}

	return fs
}

type featureWithPermissionsModel struct {
	Name             types.String `tfsdk:"name"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
	Permissions      types.String `tfsdk:"permissions"`
}

func featureWithPermissionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:             types.StringType,
		keyPermissionGroups: types.SetType{ElemType: types.StringType},
		keyPermissions:      types.StringType,
	}
}

func fromFeaturesWithPermissions(features []featureWithPermissions) (types.Set, diag.Diagnostics) {
	nullSet := types.SetNull(types.ObjectType{AttrTypes: featureWithPermissionsAttrTypes()})

	featureValues := make([]attr.Value, 0, len(features))
	for _, feature := range features {
		groupValues := make([]attr.Value, 0, len(feature.PermissionGroups))
		for _, group := range feature.PermissionGroups {
			groupValues = append(groupValues, types.StringValue(string(group)))
		}
		groupSet, diags := types.SetValue(types.StringType, groupValues)
		if diags.HasError() {
			return nullSet, diags
		}

		permissions := types.StringNull()
		if feature.permissions != "" {
			permissions = types.StringValue(feature.permissions)
		}
		object, d := types.ObjectValue(featureWithPermissionsAttrTypes(), map[string]attr.Value{
			keyName:             types.StringValue(feature.Name),
			keyPermissionGroups: groupSet,
			keyPermissions:      permissions,
		})
		if d.HasError() {
			return nullSet, d
		}

		featureValues = append(featureValues, object)
	}

	return types.SetValue(types.ObjectType{AttrTypes: featureWithPermissionsAttrTypes()}, featureValues)
}

func toFeaturesWithPermissions(ctx context.Context, featureSet types.Set) ([]featureWithPermissions, diag.Diagnostics) {
	if featureSet.IsNull() {
		return nil, nil
	}

	var models []featureWithPermissionsModel
	diags := featureSet.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	features := make([]featureWithPermissions, 0, len(models))
	for _, model := range models {
		var groupModels []string
		diags.Append(model.PermissionGroups.ElementsAs(ctx, &groupModels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		groups := make([]core.PermissionGroup, 0, len(groupModels))
		for _, group := range groupModels {
			groups = append(groups, core.PermissionGroup(group))
		}

		features = append(features, featureWithPermissions{
			Feature: core.Feature{
				Name:             model.Name.ValueString(),
				PermissionGroups: groups,
			},
			permissions: model.Permissions.ValueString(),
		})
	}

	return features, diags
}
