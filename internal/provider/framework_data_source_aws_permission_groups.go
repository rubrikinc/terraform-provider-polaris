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
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAWSPermissionGroupsDescription = `
The ´polaris_aws_permission_groups´ data source retrieves the latest permission
groups available for a single RSC AWS feature, along with the IAM action
statements that the feature requires. It is intended for users of the IAM-based
onboarding flow who want to programmatically discover which permission groups
are available (for example, the ´BASIC´ and ´RECOVERY´ split on
´RDS_PROTECTION´) and the underlying actions, instead of hard-coding them.

To look up multiple features at once, use ´for_each´ on the data source.
`

var _ datasource.DataSource = &awsPermissionGroupsDataSource{}

type awsPermissionGroupsDataSource struct {
	client *client
}

type awsPermissionGroupsModel struct {
	ID                   types.String `tfsdk:"id"`
	Feature              types.String `tfsdk:"feature"`
	PermissionGroups     types.List   `tfsdk:"permission_groups"`
	PermissionStatements types.List   `tfsdk:"permission_statements"`
}

func newAwsPermissionGroupsDataSource() datasource.DataSource {
	return &awsPermissionGroupsDataSource{}
}

func (d *awsPermissionGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "awsPermissionGroupsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_aws_permission_groups"
}

func (d *awsPermissionGroupsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "awsPermissionGroupsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAWSPermissionGroupsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the permission groups and statements returned.",
			},
			keyFeature: schema.StringAttribute{
				Required:    true,
				Description: "RSC feature name to look up permission groups for (e.g. `RDS_PROTECTION`).",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyPermissionGroups: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Permission groups available for the feature, sorted by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Permission group name.",
						},
						keyVersion: schema.Int64Attribute{
							Computed:    true,
							Description: "Permission group version.",
						},
					},
				},
			},
			keyPermissionStatements: schema.ListNestedAttribute{
				Computed: true,
				Description: "Flat list of IAM action statements required by the feature, merged across all " +
					"permission groups and exploded so each `(action, use_case)` pair is its own entry. " +
					"Sorted by `name` then `use_case`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "IAM action.",
						},
						keyUseCase: schema.StringAttribute{
							Computed:    true,
							Description: "Use case the IAM action is required for.",
						},
					},
				},
			},
		},
	}
}

func (d *awsPermissionGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "awsPermissionGroupsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *awsPermissionGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "awsPermissionGroupsDataSource.Read")

	var config awsPermissionGroupsModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	featureName := config.Feature.ValueString()
	featurePerms, err := gqlaws.Wrap(polarisClient.GQL).AllFeaturePermissions(ctx, []core.Feature{{Name: featureName}})
	if err != nil {
		res.Diagnostics.AddError("Failed to read AWS permission groups", err.Error())
		return
	}
	if len(featurePerms) != 1 {
		res.Diagnostics.AddError(
			"Unexpected RSC response for AWS permission groups",
			fmt.Sprintf("expected exactly 1 feature in response for %q, got %d", featureName, len(featurePerms)),
		)
		return
	}

	groups := slices.Clone(featurePerms[0].PermissionsGroupPermissions)
	slices.SortFunc(groups, func(a, b gqlaws.PermissionsGroupPermissions) int {
		return cmp.Compare(string(a.PermissionsGroup), string(b.PermissionsGroup))
	})

	hash := sha256.New()
	hash.Write([]byte(featureName))

	groupValues := make([]attr.Value, 0, len(groups))
	type stmtKey struct{ name, useCase string }
	stmtSet := make(map[stmtKey]struct{})
	for _, pg := range groups {
		hash.Write([]byte(pg.PermissionsGroup))
		hash.Write([]byte(strconv.Itoa(pg.Version)))

		groupValue, diags := types.ObjectValue(permissionGroupAttrTypes(), map[string]attr.Value{
			keyName:    types.StringValue(string(pg.PermissionsGroup)),
			keyVersion: types.Int64Value(int64(pg.Version)),
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		groupValues = append(groupValues, groupValue)

		for _, stmt := range pg.PermissionStatements {
			for _, act := range stmt.Actions {
				// RSC currently leaves usecase empty for AWS actions; emit
				// the action once with use_case = "" so it is still surfaced.
				if len(act.UseCases) == 0 {
					stmtSet[stmtKey{name: act.Action}] = struct{}{}
					continue
				}
				for _, uc := range act.UseCases {
					stmtSet[stmtKey{name: act.Action, useCase: uc}] = struct{}{}
				}
			}
		}
	}

	stmts := make([]stmtKey, 0, len(stmtSet))
	for k := range stmtSet {
		stmts = append(stmts, k)
	}
	slices.SortFunc(stmts, func(a, b stmtKey) int {
		if r := cmp.Compare(a.name, b.name); r != 0 {
			return r
		}
		return cmp.Compare(a.useCase, b.useCase)
	})

	stmtValues := make([]attr.Value, 0, len(stmts))
	for _, s := range stmts {
		hash.Write([]byte(s.name))
		hash.Write([]byte(s.useCase))

		stmtValue, diags := types.ObjectValue(permissionStatementAttrTypes(), map[string]attr.Value{
			keyName:    types.StringValue(s.name),
			keyUseCase: types.StringValue(s.useCase),
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		stmtValues = append(stmtValues, stmtValue)
	}

	groupsList, diags := types.ListValue(types.ObjectType{AttrTypes: permissionGroupAttrTypes()}, groupValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	stmtsList, diags := types.ListValue(types.ObjectType{AttrTypes: permissionStatementAttrTypes()}, stmtValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := awsPermissionGroupsModel{
		ID:                   types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		Feature:              config.Feature,
		PermissionGroups:     groupsList,
		PermissionStatements: stmtsList,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func permissionGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:    types.StringType,
		keyVersion: types.Int64Type,
	}
}

func permissionStatementAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:    types.StringType,
		keyUseCase: types.StringType,
	}
}
