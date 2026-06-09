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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceObjectDescription = `
The ´polaris_object´ data source is used to look up an RSC hierarchy object by
name and type. This is useful for finding the ID of an object when only its
name and type are known.

Supported object types:
  * ´AwsNativeAccount´ - AWS Native Account
  * ´AwsNativeEbsVolume´ - AWS Native EBS Volume
  * ´AwsNativeEc2Instance´ - AWS Native EC2 Instance
  * ´AwsNativeRdsInstance´ - AWS Native RDS Instance
  * ´AzureNativeResourceGroup´ - Azure Native Resource Group (requires ´subscription_id´)
  * ´AzureNativeSubscription´ - Azure Native Subscription
  * ´AzureNativeVirtualMachine´ - Azure Native Virtual Machine
`

func dataSourceObject() *schema.Resource {
	return &schema.Resource{
		ReadContext: objectRead,

		// The read timeout is used by the AwsNativeAccount retry loop which
		// polls the hierarchy until an active account appears. Other object
		// types return immediately and are unaffected by this timeout.
		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Minute),
		},

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
				Description: "Object type. Possible values are `AwsNativeAccount`, `AwsNativeEbsVolume`, `AwsNativeEc2Instance`, `AwsNativeRdsInstance`, `AzureNativeResourceGroup`, `AzureNativeSubscription` and `AzureNativeVirtualMachine`.",
				ValidateFunc: validation.StringInSlice([]string{
					"AwsNativeAccount",
					"AwsNativeEbsVolume",
					"AwsNativeEc2Instance",
					"AwsNativeRdsInstance",
					"AzureNativeResourceGroup",
					"AzureNativeSubscription",
					"AzureNativeVirtualMachine",
				}, false),
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "RSC cloud account ID of the parent Azure subscription (UUID). Required when `object_type` is `AzureNativeResourceGroup`; ignored for other object types.",
				ValidateFunc: validation.IsUUID,
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

	// Filters for workload-level object types. Unlike container-level types
	// (e.g. AwsNativeAccount, AzureNativeSubscription), workload objects do not
	// carry RSC feature-status metadata, so activity is determined via these
	// server-side filters rather than inspecting the returned feature list.
	activeFilters := []hierarchy.Filter{
		{Field: "IS_RELIC", Texts: []string{"false"}},
		{Field: "IS_GHOST", Texts: []string{"false"}},
		{Field: "IS_ACTIVE", Texts: []string{"true"}},
		{Field: "IS_ARCHIVED", Texts: []string{"false"}},
	}

	var objects []hierarchy.Object
	switch objectType {
	case hierarchy.ObjectType("AwsNativeAccount"):
		// Container-level type: the API can return multiple entries for the
		// same account name (e.g. an account added to RSC more than once).
		// Activity is determined by inspecting the RSC feature status on each
		// result rather than using server-side filters.
		//
		// A newly onboarded AWS account may not appear in the hierarchy
		// immediately after creation because the polaris_aws_cnp_account
		// resource only registers the account while the hierarchy object is
		// created asynchronously after the polaris_aws_cnp_account_attachments
		// resource finalizes the account setup. When polaris_object depends on
		// the account, it can run before the hierarchy has caught up. We retry
		// until an active account is found or the read timeout is reached.
		for {
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
			if len(objects) > 0 {
				break
			}

			tflog.Debug(ctx, "no active account found in hierarchy, retrying", map[string]any{
				"name": name,
			})

			select {
			case <-ctx.Done():
				return diag.Errorf("timed out waiting for active object with name %q and type %q: %d result(s) returned, none active", name, objectType, len(results))
			case <-time.After(10 * time.Second):
			}
		}
	case hierarchy.ObjectType("AwsNativeEbsVolume"):
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeEBSVolume](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case hierarchy.ObjectType("AwsNativeEc2Instance"):
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeEC2Instance](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case hierarchy.ObjectType("AwsNativeRdsInstance"):
		results, err := hierarchy.ObjectsByName[hierarchy.AWSNativeRDSInstance](ctx, api, name, hierarchy.WorkloadAllSubHierarchyType, activeFilters...)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case hierarchy.ObjectType("AzureNativeSubscription"):
		// Container-level type: same feature-status strategy as AwsNativeAccount.
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
	case hierarchy.ObjectType("AzureNativeVirtualMachine"):
		results, err := hierarchy.ObjectsByName[hierarchy.AzureNativeVirtualMachine](ctx, api, name, hierarchy.WorkloadAzureVM, activeFilters...)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, r := range results {
			objects = append(objects, r.Object)
		}
	case hierarchy.ObjectType("AzureNativeResourceGroup"):
		// Azure resource groups are not exposed via the hierarchy inventory
		// query used for the other object types, so route through the
		// dedicated NativeResourceGroups SDK wrapper. Resource group names are
		// unique within a subscription, so a (subscription, name) tuple
		// resolves to at most one resource group.
		subIDStr := d.Get(keySubscriptionID).(string)
		if subIDStr == "" {
			return diag.Errorf("subscription_id is required when object_type is %q", objectType)
		}
		subID, err := uuid.Parse(subIDStr)
		if err != nil {
			return diag.Errorf("invalid subscription_id %q: %s", subIDStr, err)
		}

		// nameSubstring is a substring filter server-side, so a query for
		// "foo" can return "foo" and "foobar". Pick the exact-name match.
		rgs, err := azure.Wrap(client).NativeResourceGroups(ctx, []uuid.UUID{subID}, name)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, rg := range rgs {
			if rg.Name == name {
				d.SetId(rg.ID)
				return nil
			}
		}
		return diag.Errorf("no object found with name %q and type %q in subscription %q", name, objectType, subIDStr)
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
