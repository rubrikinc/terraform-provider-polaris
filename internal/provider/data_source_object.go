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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceObjectDescription = `
The ´polaris_object´ data source is used to look up an RSC hierarchy object by
name and type. This is useful for finding the ID of an object when only its
name and type are known.

Supported object types:
  * ´AwsNativeAccount´ - AWS Native Account
  * ´AzureNativeSubscription´ - Azure Native Subscription
`

func dataSourceObject() *schema.Resource {
	return &schema.Resource{
		ReadContext: objectRead,

		Description: description(dataSourceObjectDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Object ID (UUID).",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Exact object name to search for.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyObjectType: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Object type (e.g., 'AwsNativeAccount', 'AzureNativeSubscription').",
				ValidateFunc: validation.StringInSlice([]string{
					"AwsNativeAccount",
					"AzureNativeSubscription",
				}, false),
			},
		},
	}
}

func objectRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "objectRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get(keyName).(string)
	objectType := hierarchy.ObjectType(d.Get(keyObjectType).(string))

	api := hierarchy.Wrap(client.GQL)

	// Call the appropriate ObjectsByName based on the object type
	var objects []hierarchy.Object
	switch objectType {
	case hierarchy.ObjectType("AwsNativeAccount"):
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeAccount](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			var active bool
			for _, feature := range r.Features {
				switch feature.Status {
				case hierarchy.StatusAdded, hierarchy.StatusRefreshed, hierarchy.StatusRefreshing:
					active = true
				default:
					tflog.Debug(ctx, "skipping account because it is not active", map[string]any{
						"account": r.Object.Name,
						"status":  feature.Status,
					})
				}
				if active {
					objects = append(objects, r.Object)
					break
				}
			}
		}
	case hierarchy.ObjectType("AzureNativeSubscription"):
		results, err := hierarchy.ObjectsByName[hierarchy.AzureNativeSubscription](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			var active bool
			for _, feature := range r.Features {
				switch feature.Status {
				case hierarchy.StatusAdded, hierarchy.StatusRefreshed, hierarchy.StatusRefreshing:
					active = true
				default:
					tflog.Debug(ctx, "skipping subscription because it is not active", map[string]any{
						"subscription": r.Object.Name,
						"status":       feature.Status,
					})
				}
				if active {
					objects = append(objects, r.Object)
					break
				}
			}
		}
	}

	if len(objects) == 0 {
		return diag.Errorf("no object found with name %q and type %q", name, objectType)
	}
	if len(objects) > 1 {
		return diag.Errorf("multiple objects found with name %q and type %q", name, objectType)
	}

	d.SetId(objects[0].ID.String())

	return nil
}
