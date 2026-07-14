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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
)

const dataSourceObjectsDescription = `
The ´polaris_objects´ data source is used to look up all RSC hierarchy objects
of a given type. Unlike ´polaris_object´, it does not filter by name — it
returns every matching object.

Supported object types:
  * ´AzureNativeResourceGroup´ - Azure Native Resource Group (optionally
    scoped to a single subscription with ´subscription_id´; omitting it
    searches across all subscriptions managed by RSC)

Additional object types will be added in future releases.
`

var _ datasource.DataSource = &objectsDataSource{}

type objectsDataSource struct {
	client *client
}

type objectsModel struct {
	ID             types.String `tfsdk:"id"`
	ObjectType     types.String `tfsdk:"object_type"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	Objects        types.Set    `tfsdk:"objects"`
}

func newObjectsDataSource() datasource.DataSource {
	return &objectsDataSource{}
}

func (d *objectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	tflog.Trace(ctx, "objectsDataSource.Metadata")

	res.TypeName = req.ProviderTypeName + "_objects"
}

func (d *objectsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	tflog.Trace(ctx, "objectsDataSource.Schema")

	res.Schema = schema.Schema{
		Description: description(dataSourceObjectsDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the object type, subscription ID and objects returned.",
			},
			keyObjectType: schema.StringAttribute{
				Required:    true,
				Description: "Object type. The only value currently supported is `AzureNativeResourceGroup`.",
				Validators: []validator.String{
					stringvalidator.OneOf("AzureNativeResourceGroup"),
				},
			},
			keySubscriptionID: schema.StringAttribute{
				Optional: true,
				Description: "RSC cloud account ID of an Azure subscription (UUID) to scope the search to. " +
					"When omitted, resource groups across all subscriptions managed by RSC are returned. " +
					"Only used when `object_type` is `AzureNativeResourceGroup`.",
				Validators: []validator.String{
					isUUID(),
				},
			},
			keyObjects: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Objects matching `object_type` (and `subscription_id`, if specified).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						keyID: schema.StringAttribute{
							Computed:    true,
							Description: "Object ID (UUID).",
						},
						keyName: schema.StringAttribute{
							Computed:    true,
							Description: "Object name.",
						},
						keySubscriptionID: schema.StringAttribute{
							Computed:    true,
							Description: "RSC cloud account ID of the parent Azure subscription (UUID).",
						},
					},
				},
			},
		},
	}
}

func (d *objectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "objectsDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client)
}

func (d *objectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "objectsDataSource.Read")

	var config objectsModel
	res.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if res.Diagnostics.HasError() {
		return
	}

	polarisClient, err := d.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	objectType := config.ObjectType.ValueString()
	subIDStr := config.SubscriptionID.ValueString()

	var subIDs []uuid.UUID
	if subIDStr != "" {
		subID, err := uuid.Parse(subIDStr)
		if err != nil {
			res.Diagnostics.AddError("Invalid subscription_id", err.Error())
			return
		}
		subIDs = []uuid.UUID{subID}
	}

	// Passing an empty nameSubstring disables the RSC substring filter, so
	// every resource group in scope is returned.
	rgs, err := azure.Wrap(polarisClient).NativeResourceGroups(ctx, subIDs, "")
	if err != nil {
		res.Diagnostics.AddError("Failed to read Azure native resource groups", err.Error())
		return
	}

	slices.SortFunc(rgs, func(a, b gqlazure.NativeResourceGroup) int {
		return cmp.Compare(a.ID, b.ID)
	})

	hash := sha256.New()
	hash.Write([]byte(objectType))
	hash.Write([]byte(subIDStr))

	objectValues := make([]attr.Value, 0, len(rgs))
	for _, rg := range rgs {
		hash.Write([]byte(rg.ID))
		hash.Write([]byte(rg.Name))
		hash.Write([]byte(rg.Subscription.ID))

		objectValue, diags := types.ObjectValue(objectAttrTypes(), map[string]attr.Value{
			keyID:             types.StringValue(rg.ID),
			keyName:           types.StringValue(rg.Name),
			keySubscriptionID: types.StringValue(rg.Subscription.ID),
		})
		res.Diagnostics.Append(diags...)
		if res.Diagnostics.HasError() {
			return
		}
		objectValues = append(objectValues, objectValue)
	}

	objectsSet, diags := types.SetValue(types.ObjectType{AttrTypes: objectAttrTypes()}, objectValues)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	state := objectsModel{
		ID:             types.StringValue(fmt.Sprintf("%x", hash.Sum(nil))),
		ObjectType:     config.ObjectType,
		SubscriptionID: config.SubscriptionID,
		Objects:        objectsSet,
	}

	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

func objectAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		keyID:             types.StringType,
		keyName:           types.StringType,
		keySubscriptionID: types.StringType,
	}
}
