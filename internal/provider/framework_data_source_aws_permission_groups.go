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
The ´polaris_aws_permission_groups´ data source returns the permission groups
available for a single RSC AWS feature, along with the IAM action statements
that each permission group requires. It exposes the same catalog used by RSC
itself, so configurations can discover the available groups (for example, the
´BASIC´ and ´RECOVERY´ split on ´RDS_PROTECTION´) at plan time.

The IAM action statements returned are informational. To generate the IAM roles
and policies needed for the IAM-based onboarding flow, use the
´polaris_aws_cnp_artifacts´ and ´polaris_aws_cnp_permissions´ data sources,
which emit the artifacts and policy documents in the shape RSC expects.

~> **Note:** RSC follows a least-privilege model: a permission group should be
opted into only when its capabilities are required. For example, ´RECOVERY´
grants the elevated AWS permissions needed to perform recovery operations and
should be configured only on accounts that need to perform recoveries.
Hard-coding a known set of permission groups is a valid choice when it keeps
the granted permissions to the minimum required.

To look up multiple features at once, use ´for_each´ on the data source.
`

var _ datasource.DataSource = &awsPermissionGroupsDataSource{}

type awsPermissionGroupsDataSource struct {
	client *client
}

type awsPermissionGroupsModel struct {
	ID               types.String `tfsdk:"id"`
	Feature          types.String `tfsdk:"feature"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
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
			keyPermissionGroups: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Permission groups available for the feature.",
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
						keyStatements: schema.SetNestedAttribute{
							Computed: true,
							Description: "IAM actions required by this permission group, one entry per " +
								"`(action, use_case)` pair.",
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

	type stmtKey struct{ name, useCase string }
	groupValues := make([]attr.Value, 0, len(groups))
	for _, pg := range groups {
		hash.Write([]byte(pg.PermissionsGroup))
		hash.Write([]byte(strconv.Itoa(pg.Version)))

		stmtSet := make(map[stmtKey]struct{})
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

			stmtValue, diags := types.ObjectValue(statementAttrTypes(), map[string]attr.Value{
				keyName:    types.StringValue(s.name),
				keyUseCase: types.StringValue(s.useCase),
			})
			res.Diagnostics.Append(diags...)
			if res.Diagnostics.HasError() {
				return
			}
			stmtValues = append(stmtValues, stmtValue)
		}

		stmtsSet, diags := types.SetValue(types.ObjectType{AttrTypes: statementAttrTypes()}, stmtValues)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		groupValue, diags := types.ObjectValue(permissionGroupAttrTypes(), map[string]attr.Value{
			keyName:       types.StringValue(string(pg.PermissionsGroup)),
			keyVersion:    types.Int64Value(int64(pg.Version)),
			keyStatements: stmtsSet,
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		groupValues = append(groupValues, groupValue)
	}

	groupsSet, diags := types.SetValue(types.ObjectType{AttrTypes: permissionGroupAttrTypes()}, groupValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := awsPermissionGroupsModel{
		ID:               types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		Feature:          config.Feature,
		PermissionGroups: groupsSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func statementAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:    types.StringType,
		keyUseCase: types.StringType,
	}
}

func permissionGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:       types.StringType,
		keyVersion:    types.Int64Type,
		keyStatements: types.SetType{ElemType: types.ObjectType{AttrTypes: statementAttrTypes()}},
	}
}
