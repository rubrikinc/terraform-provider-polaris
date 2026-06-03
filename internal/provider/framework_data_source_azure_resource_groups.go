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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
)

const dataSourceAzureResourceGroupsDescription = `
The ´polaris_azure_resource_groups´ data source lists Azure resource groups
that are visible to RSC, optionally narrowed to a set of subscriptions. It is
intended for users who need to enumerate resource groups under one or more
RSC-managed Azure subscriptions (for example, to drive ´for_each´ over the
resulting list).

If ´subscription_ids´ is omitted, resource groups from every managed
subscription are returned. Resource group names are unique only within a
subscription, so consumers that key on name should also branch on
´subscription_id´.
`

var _ datasource.DataSource = &azureResourceGroupsDataSource{}

type azureResourceGroupsDataSource struct {
	client *client
}

type azureResourceGroupsModel struct {
	ID              types.String `tfsdk:"id"`
	SubscriptionIDs types.Set    `tfsdk:"subscription_ids"`
	Name            types.String `tfsdk:"name"`
	ResourceGroups  types.List   `tfsdk:"resource_groups"`
}

func newAzureResourceGroupsDataSource() datasource.DataSource {
	return &azureResourceGroupsDataSource{}
}

func (d *azureResourceGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "azureResourceGroupsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_azure_resource_groups"
}

func (d *azureResourceGroupsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "azureResourceGroupsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceAzureResourceGroupsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the resource groups returned, used as a stable identifier.",
			},
			keySubscriptionIDs: schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "RSC cloud account IDs of the Azure subscriptions to filter by. Omit to list resource groups across all managed subscriptions.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(isUUID()),
				},
			},
			keyName: schema.StringAttribute{
				Optional:    true,
				Description: "Exact resource group name to filter by. Resource group names are unique within a subscription, so this yields at most one entry per subscription. Omit to skip name filtering.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
			keyResourceGroups: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Resource groups visible to RSC, sorted by `(subscription_id, name)`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyID: schema.StringAttribute{
							Computed:    true,
							Description: "RSC ID of the resource group.",
						},
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Name of the resource group.",
						},
						keySubscriptionID: schema.StringAttribute{
							Computed:    true,
							Description: "RSC ID of the parent Azure subscription.",
						},
						keySubscriptionName: schema.StringAttribute{
							Computed:    true,
							Description: "Name of the parent Azure subscription.",
						},
						keySLAAssignment: schema.StringAttribute{
							Computed:    true,
							Description: "How the SLA domain is assigned to the resource group (e.g. `Direct`, `Derived`, `Unassigned`).",
						},
						keyLogicalPath: schema.ListNestedAttribute{
							Computed:    true,
							Description: "Logical hierarchy path to the resource group within RSC.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: pathNodeSchemaAttrs(),
							},
						},
						keyPhysicalPath: schema.ListNestedAttribute{
							Computed:    true,
							Description: "Physical hierarchy path to the resource group within RSC.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: pathNodeSchemaAttrs(),
							},
						},
					},
				},
			},
		},
	}
}

func (d *azureResourceGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "azureResourceGroupsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *azureResourceGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "azureResourceGroupsDataSource.Read")

	var config azureResourceGroupsModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	var subIDs []uuid.UUID
	if !config.SubscriptionIDs.IsNull() && !config.SubscriptionIDs.IsUnknown() {
		var subIDStrings []string
		res.Diagnostics.Append(config.SubscriptionIDs.ElementsAs(ctx, &subIDStrings, false)...)
		if res.Diagnostics.HasError() {
			return
		}
		subIDs = make([]uuid.UUID, 0, len(subIDStrings))
		for _, s := range subIDStrings {
			id, err := uuid.Parse(s)
			if err != nil {
				res.Diagnostics.AddError("Invalid subscription_id", fmt.Sprintf("%q: %s", s, err))
				return
			}
			subIDs = append(subIDs, id)
		}
	}

	exactName := config.Name.ValueString()
	groups, err := azure.Wrap(polarisClient).NativeResourceGroups(ctx, subIDs, exactName)
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure resource groups", err.Error())
		return
	}

	// The server-side filter is a substring match, so a query for "foo" can
	// return "foo" and "foobar". Drop everything that isn't an exact match.
	if exactName != "" {
		n := 0
		for _, g := range groups {
			if g.Name == exactName {
				groups[n] = g
				n++
			}
		}
		groups = groups[:n]
	}

	slices.SortFunc(groups, func(a, b gqlazure.NativeResourceGroup) int {
		if r := cmp.Compare(a.Subscription.ID, b.Subscription.ID); r != 0 {
			return r
		}
		return cmp.Compare(a.Name, b.Name)
	})

	hash := sha256.New()
	rgValues := make([]attr.Value, 0, len(groups))
	for _, g := range groups {
		hash.Write([]byte(g.Subscription.ID))
		hash.Write([]byte{0})
		hash.Write([]byte(g.ID))
		hash.Write([]byte{0})

		logicalPath, diags := pathToList(g.LogicalPath)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		physicalPath, diags := pathToList(g.PhysicalPath)
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}

		rgValue, diags := types.ObjectValue(resourceGroupAttrTypes(), map[string]attr.Value{
			keyID:               types.StringValue(g.ID),
			keyName:             types.StringValue(g.Name),
			keySubscriptionID:   types.StringValue(g.Subscription.ID),
			keySubscriptionName: types.StringValue(g.Subscription.Name),
			keySLAAssignment:    types.StringValue(string(g.SLAAssignment)),
			keyLogicalPath:      logicalPath,
			keyPhysicalPath:     physicalPath,
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		rgValues = append(rgValues, rgValue)
	}

	rgList, diags := types.ListValue(types.ObjectType{AttrTypes: resourceGroupAttrTypes()}, rgValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := azureResourceGroupsModel{
		ID:              types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		SubscriptionIDs: config.SubscriptionIDs,
		Name:            config.Name,
		ResourceGroups:  rgList,
	}
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func pathNodeSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		keyFID: schema.StringAttribute{
			Computed:    true,
			Description: "FID of the path node.",
		},
		keyName: schema.StringAttribute{
			Computed:    true,
			Description: "Name of the path node.",
		},
		keyObjectType: schema.StringAttribute{
			Computed:    true,
			Description: "Object type of the path node.",
		},
	}
}

func pathNodeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyFID:        types.StringType,
		keyName:       types.StringType,
		keyObjectType: types.StringType,
	}
}

func resourceGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyID:               types.StringType,
		keyName:             types.StringType,
		keySubscriptionID:   types.StringType,
		keySubscriptionName: types.StringType,
		keySLAAssignment:    types.StringType,
		keyLogicalPath:      types.ListType{ElemType: types.ObjectType{AttrTypes: pathNodeAttrTypes()}},
		keyPhysicalPath:     types.ListType{ElemType: types.ObjectType{AttrTypes: pathNodeAttrTypes()}},
	}
}

func pathToList(nodes []gqlazure.PathNode) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	pathType := types.ObjectType{AttrTypes: pathNodeAttrTypes()}

	values := make([]attr.Value, 0, len(nodes))
	for _, n := range nodes {
		v, d := types.ObjectValue(pathNodeAttrTypes(), map[string]attr.Value{
			keyFID:        types.StringValue(n.FID),
			keyName:       types.StringValue(n.Name),
			keyObjectType: types.StringValue(n.ObjectType),
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(pathType), diags
		}
		values = append(values, v)
	}
	list, d := types.ListValue(pathType, values)
	diags.Append(d...)
	return list, diags
}
