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

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
groups available for one or more RSC AWS features, along with the IAM action
statements that each permission group grants. It is intended for users of the
IAM-based onboarding flow who want to programmatically discover which permission
groups are available (for example, the ´BASIC´ and ´RECOVERY´ split on
´RDS_PROTECTION´) and the underlying actions, instead of hard-coding them.
`

var _ datasource.DataSource = &awsPermissionGroupsDataSource{}

type awsPermissionGroupsDataSource struct {
	client *client
}

type awsPermissionGroupsModel struct {
	ID           types.String `tfsdk:"id"`
	FeatureNames types.Set    `tfsdk:"feature_names"`
	Feature      types.List   `tfsdk:"feature"`
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
				Description: "SHA-256 hash of the feature names and permission groups returned.",
			},
			keyFeatureNames: schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "RSC feature names to look up permission groups for (e.g. `RDS_PROTECTION`).",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(isNotWhiteSpace()),
				},
			},
			keyFeature: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Permission group catalog grouped by RSC feature.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "RSC feature name.",
						},
						keyPermissionGroup: schema.ListNestedAttribute{
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
									keyPermissionStatement: schema.ListNestedAttribute{
										Computed:    true,
										Description: "IAM permission statements granted by the permission group.",
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												keyAction: schema.ListNestedAttribute{
													Computed:    true,
													Description: "IAM actions granted by the permission statement.",
													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															keyName: schema.StringAttribute{
																Computed:    true,
																Description: "IAM action.",
															},
															keyUseCases: schema.ListAttribute{
																Computed:    true,
																ElementType: types.StringType,
																Description: "Use cases for the IAM action.",
															},
														},
													},
												},
											},
										},
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

	var featureNames []string
	res.Diagnostics.Append(config.FeatureNames.ElementsAs(ctx, &featureNames, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	features := make([]core.Feature, 0, len(featureNames))
	for _, name := range featureNames {
		features = append(features, core.Feature{Name: name})
	}

	featurePerms, err := gqlaws.Wrap(polarisClient.GQL).AllFeaturePermissions(ctx, features)
	if err != nil {
		res.Diagnostics.AddError("Failed to read AWS permission groups", err.Error())
		return
	}

	slices.SortFunc(featurePerms, func(a, b gqlaws.FeaturePermissions) int {
		return cmp.Compare(a.Feature, b.Feature)
	})

	hash := sha256.New()
	featureValues := make([]attr.Value, 0, len(featurePerms))
	for _, fp := range featurePerms {
		hash.Write([]byte(fp.Feature))

		groupsSorted := slices.Clone(fp.PermissionsGroupPermissions)
		slices.SortFunc(groupsSorted, func(a, b gqlaws.PermissionsGroupPermissions) int {
			return cmp.Compare(string(a.PermissionsGroup), string(b.PermissionsGroup))
		})

		groupValues := make([]attr.Value, 0, len(groupsSorted))
		for _, pg := range groupsSorted {
			hash.Write([]byte(pg.PermissionsGroup))
			hash.Write([]byte(strconv.Itoa(pg.Version)))

			statementValues := make([]attr.Value, 0, len(pg.PermissionStatements))
			for _, stmt := range pg.PermissionStatements {
				actionsSorted := slices.Clone(stmt.Actions)
				slices.SortFunc(actionsSorted, func(a, b gqlaws.AWSActionWithUseCase) int {
					return cmp.Compare(a.Action, b.Action)
				})

				actionValues := make([]attr.Value, 0, len(actionsSorted))
				for _, act := range actionsSorted {
					hash.Write([]byte(act.Action))

					useCases, diags := types.ListValueFrom(ctx, types.StringType, act.UseCases)
					res.Diagnostics.Append(diags...)
					if res.Diagnostics.HasError() {
						return
					}

					actionValue, diags := types.ObjectValue(actionAttrTypes(), map[string]attr.Value{
						keyName:     types.StringValue(act.Action),
						keyUseCases: useCases,
					})
					res.Diagnostics.Append(diags...)
					if res.Diagnostics.HasError() {
						return
					}
					actionValues = append(actionValues, actionValue)
				}

				actionsList, diags := types.ListValue(types.ObjectType{AttrTypes: actionAttrTypes()}, actionValues)
				res.Diagnostics.Append(diags...)
				if res.Diagnostics.HasError() {
					return
				}

				statementValue, diags := types.ObjectValue(permissionStatementAttrTypes(), map[string]attr.Value{
					keyAction: actionsList,
				})
				res.Diagnostics.Append(diags...)
				if res.Diagnostics.HasError() {
					return
				}
				statementValues = append(statementValues, statementValue)
			}

			statementsList, diags := types.ListValue(types.ObjectType{AttrTypes: permissionStatementAttrTypes()}, statementValues)
			res.Diagnostics.Append(diags...)
			if res.Diagnostics.HasError() {
				return
			}

			groupValue, diags := types.ObjectValue(permissionGroupAttrTypes(), map[string]attr.Value{
				keyName:                types.StringValue(string(pg.PermissionsGroup)),
				keyVersion:             types.Int64Value(int64(pg.Version)),
				keyPermissionStatement: statementsList,
			})
			res.Diagnostics.Append(diags...)
			if res.Diagnostics.HasError() {
				return
			}
			groupValues = append(groupValues, groupValue)
		}

		groupsList, diags := types.ListValue(types.ObjectType{AttrTypes: permissionGroupAttrTypes()}, groupValues)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		featureValue, diags := types.ObjectValue(featurePermissionsAttrTypes(), map[string]attr.Value{
			keyName:            types.StringValue(fp.Feature),
			keyPermissionGroup: groupsList,
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		featureValues = append(featureValues, featureValue)
	}

	featuresList, diags := types.ListValue(types.ObjectType{AttrTypes: featurePermissionsAttrTypes()}, featureValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := awsPermissionGroupsModel{
		ID:           types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		FeatureNames: config.FeatureNames,
		Feature:      featuresList,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func actionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:     types.StringType,
		keyUseCases: types.ListType{ElemType: types.StringType},
	}
}

func permissionStatementAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyAction: types.ListType{ElemType: types.ObjectType{AttrTypes: actionAttrTypes()}},
	}
}

func permissionGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:                types.StringType,
		keyVersion:             types.Int64Type,
		keyPermissionStatement: types.ListType{ElemType: types.ObjectType{AttrTypes: permissionStatementAttrTypes()}},
	}
}

func featurePermissionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:            types.StringType,
		keyPermissionGroup: types.ListType{ElemType: types.ObjectType{AttrTypes: permissionGroupAttrTypes()}},
	}
}
