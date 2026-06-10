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
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAzurePermissionGroupsDescription = `
The ´polaris_azure_permission_groups´ data source returns the permission groups
available for a single RSC Azure feature, along with the Azure RBAC actions
and data actions that each permission group requires. It exposes the same
catalog used by RSC itself, so configurations can discover the available
groups (for example, the ´BASIC´ and ´RECOVERY´ split on
´AZURE_SQL_DB_PROTECTION´) at plan time.

Each statement carries the scope it applies to and the kind of operation it
authorises. Azure RBAC distinguishes management-plane operations (´actions´)
from data-plane operations (´data_actions´), and many features require
permissions at both the subscription and resource group scopes.

The statements returned are informational. To get the permission set RSC
actually requires for a specific ´(feature, permission_groups)´ combination,
feed the discovered groups into the ´polaris_azure_permissions´ data source —
it returns the policy-shaped ´subscription_actions´,
´subscription_data_actions´, ´resource_group_actions´ and
´resource_group_data_actions´ lists ready to drive an Azure role definition.

~> **Note:** RSC follows a least-privilege model: a permission group should be
opted into only when its capabilities are required. For example, ´RECOVERY´
grants the elevated Azure permissions needed to perform recovery operations
and should be configured only on accounts that need to perform recoveries.
Hard-coding a known set of permission groups is a valid choice when it keeps
the granted permissions to the minimum required.

To look up multiple features at once, use ´for_each´ on the data source.
`

var _ datasource.DataSource = &azurePermissionGroupsDataSource{}

type azurePermissionGroupsDataSource struct {
	client *client
}

type azurePermissionGroupsModel struct {
	ID               types.String `tfsdk:"id"`
	Feature          types.String `tfsdk:"feature"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
}

func newAzurePermissionGroupsDataSource() datasource.DataSource {
	return &azurePermissionGroupsDataSource{}
}

func (d *azurePermissionGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azurePermissionGroupsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_azure_permission_groups"
}

func (d *azurePermissionGroupsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azurePermissionGroupsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzurePermissionGroupsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the permission groups and statements returned.",
			},
			keyFeature: schema.StringAttribute{
				Required:    true,
				Description: "RSC feature name to look up permission groups for (e.g. `CLOUD_NATIVE_PROTECTION`).",
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
							Description: "Azure RBAC permissions required by this permission group, one " +
								"entry per `(scope, kind, permission, use_case)` tuple.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									keyScope: schema.StringAttribute{
										Computed: true,
										Description: "Azure RBAC scope at which the permission applies. " +
											"One of `subscription` or `resource_group`.",
									},
									keyKind: schema.StringAttribute{
										Computed: true,
										Description: "Azure RBAC permission kind. One of `action` " +
											"(management plane) or `data_action` (data plane).",
									},
									keyPermission: schema.StringAttribute{
										Computed:    true,
										Description: "Azure RBAC permission string (e.g. `Microsoft.Compute/virtualMachines/read`).",
									},
									keyUseCase: schema.StringAttribute{
										Computed:    true,
										Description: "Use case the permission is required for. May be empty.",
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

func (d *azurePermissionGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azurePermissionGroupsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azurePermissionGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azurePermissionGroupsDataSource.Read")

	var config azurePermissionGroupsModel
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
	featurePerms, err := gqlazure.Wrap(polarisClient.GQL).AllPermissionsGroupsByFeature(ctx, []core.Feature{{Name: featureName}})
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure permission groups", err.Error())
		return
	}
	if len(featurePerms) != 1 {
		res.Diagnostics.AddError(
			"Unexpected RSC response for Azure permission groups",
			fmt.Sprintf("expected exactly 1 feature in response for %q, got %d", featureName, len(featurePerms)),
		)
		return
	}

	groups := slices.Clone(featurePerms[0].PermissionGroups)
	slices.SortFunc(groups, func(a, b gqlazure.PermissionGroupInfo) int {
		return cmp.Compare(string(a.PermissionGroup), string(b.PermissionGroup))
	})

	hash := sha256.New()
	hash.Write([]byte(featureName))

	type stmtKey struct{ scope, kind, permission, useCase string }
	groupValues := make([]attr.Value, 0, len(groups))
	for _, pg := range groups {
		hash.Write([]byte(pg.PermissionGroup))
		hash.Write([]byte(strconv.Itoa(pg.Version)))

		stmtSet := make(map[stmtKey]struct{})
		collect := func(scope string, scopePerms []gqlazure.ScopePermissions) {
			for _, sp := range scopePerms {
				for _, act := range sp.IncludedActionsWithUseCase {
					stmtSet[stmtKey{scope: scope, kind: keyAction, permission: act.Permission, useCase: act.UseCase}] = struct{}{}
				}
				for _, act := range sp.IncludedDataActionsWithUseCase {
					stmtSet[stmtKey{scope: scope, kind: keyDataAction, permission: act.Permission, useCase: act.UseCase}] = struct{}{}
				}
			}
		}
		collect(keySubscription, pg.SubscriptionPermissions)
		collect(keyResourceGroup, pg.ResourceGroupPermissions)

		stmts := make([]stmtKey, 0, len(stmtSet))
		for k := range stmtSet {
			stmts = append(stmts, k)
		}
		slices.SortFunc(stmts, func(a, b stmtKey) int {
			if r := cmp.Compare(a.scope, b.scope); r != 0 {
				return r
			}
			if r := cmp.Compare(a.kind, b.kind); r != 0 {
				return r
			}
			if r := cmp.Compare(a.permission, b.permission); r != 0 {
				return r
			}
			return cmp.Compare(a.useCase, b.useCase)
		})

		stmtValues := make([]attr.Value, 0, len(stmts))
		for _, s := range stmts {
			hash.Write([]byte(s.scope))
			hash.Write([]byte(s.kind))
			hash.Write([]byte(s.permission))
			hash.Write([]byte(s.useCase))

			stmtValue, diags := types.ObjectValue(azureStatementAttrTypes(), map[string]attr.Value{
				keyScope:      types.StringValue(s.scope),
				keyKind:       types.StringValue(s.kind),
				keyPermission: types.StringValue(s.permission),
				keyUseCase:    types.StringValue(s.useCase),
			})
			res.Diagnostics.Append(diags...)
			if res.Diagnostics.HasError() {
				return
			}
			stmtValues = append(stmtValues, stmtValue)
		}

		stmtsSet, diags := types.SetValue(types.ObjectType{AttrTypes: azureStatementAttrTypes()}, stmtValues)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		groupValue, diags := types.ObjectValue(azurePermissionGroupAttrTypes(), map[string]attr.Value{
			keyName:       types.StringValue(string(pg.PermissionGroup)),
			keyVersion:    types.Int64Value(int64(pg.Version)),
			keyStatements: stmtsSet,
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		groupValues = append(groupValues, groupValue)
	}

	groupsSet, diags := types.SetValue(types.ObjectType{AttrTypes: azurePermissionGroupAttrTypes()}, groupValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := azurePermissionGroupsModel{
		ID:               types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		Feature:          config.Feature,
		PermissionGroups: groupsSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func azureStatementAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyScope:      types.StringType,
		keyKind:       types.StringType,
		keyPermission: types.StringType,
		keyUseCase:    types.StringType,
	}
}

func azurePermissionGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyName:       types.StringType,
		keyVersion:    types.Int64Type,
		keyStatements: types.SetType{ElemType: types.ObjectType{AttrTypes: azureStatementAttrTypes()}},
	}
}
