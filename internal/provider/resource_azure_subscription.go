// Copyright 2021 Rubrik, Inc.
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
	"errors"
	"log"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceAzureSubscription defines the schema for the Azure subscription
// resource.
func resourceAzureSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateSubscription,
		ReadContext:   azureReadSubscription,
		UpdateContext: azureUpdateSubscription,
		DeleteContext: azureDeleteSubscription,

		Description: "The `polaris_azure_subscription` resource adds an Azure subscription to RSC. When the first " +
			"subscription for an Azure tenant is added, a corresponding tenant is created in RSC. The RSC tenant is " +
			"automatically destroyed when it's last subscription is removed.\n" +
			"\n" +
			"Any combination of different RSC features can be enabled for a subscription:\n" +
			"  1. `cloud_native_archival` - Provides archival of data from data center workloads for disaster recovery " +
			"     and long-term retention.\n" +
			"  2. `cloud_native_archival_encryption` - Allows cloud archival locations to be encrypted with customer " +
			"     managed keys.\n" +
			"  3. `cloud_native_protection` - Provides protection for Azure virtual machines and managed disks through " +
			"     the rules and policies of SLA Domains.\n" +
			"  4. `exocompute` - Provides snapshot indexing, file recovery, storage tiering, and application-consistent " +
			"     protection of Azure objects.\n" +
			"  5. `sql_db_protection` - Provides centralized database backup management and recovery in an Azure SQL " +
			"     Database deployment.\n" +
			"  6. `sql_mi_protection` - Provides centralized database backup management and recovery for an Azure SQL " +
			"     Managed Instance deployment.\n" +
			"\n" +
			"-> **Note:** As of now, `sql_db_protection` and `sql_mi_protection` does not support specifying an Azure " +
			"   resource group.\n",
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID.",
			},
			keyCloudNativeArchival: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions to enable the Cloud Native Archival feature in. Should be " +
								"specified in the standard Azure style, e.g. `eastus`.",
						},
						keyResourceGroupName: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Name of the Azure resource group where RSC places all resources created by the feature.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Region of the Azure resource group.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Tags to add to the Azure resource group.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Archival feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC Cloud Native Archival feature for the Azure subscription.",
			},
			keyCloudNativeArchivalEncryption: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions to enable the Cloud Native Archival Encryption feature in. " +
								"Should be specified in the standard Azure style, e.g. `eastus`.",
						},
						keyResourceGroupName: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Region of the Azure resource group.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Tags to add to the Azure resource group.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Archival Encryption feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC Cloud Native Archival Encryption feature for the Azure subscription.",
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions that RSC will monitor for resources to protect according to " +
								"SLA Domains. Should be specified in the standard Azure style, e.g. `eastus`.",
						},
						keyResourceGroupName: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Region of the Azure resource group.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Tags to add to the Azure resource group.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC Cloud Native Protection feature for the Azure subscription.",
			},
			keyDeleteSnapshotsOnDestroy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed. Default value is `false`.",
			},
			keyExocompute: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions to enable the Exocompute feature in. Should be specified in " +
								"the standard Azure style, e.g. `eastus`.",
						},
						keyResourceGroupName: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Name of the Azure resource group where RSC places all resources created by the feature.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Region of the Azure resource group.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Tags to add to the Azure resource group.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Exocompute feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC Exocompute feature for the Azure subscription.",
			},
			keySQLDBProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions to enable the SQL DB Protection feature in. Should be " +
								"specified in the standard Azure style, e.g. `eastus`.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the SQL DB Protection feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC SQL DB Protection feature for the Azure subscription.",
			},
			keySQLMIProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MinItems: 1,
							Required: true,
							Description: "Azure regions to enable the SQL MI Protection feature in. Should be " +
								"specified in the standard Azure style, e.g. `eastus`.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the SQL MI Protection feature.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudNativeArchival, keyCloudNativeArchivalEncryption, keyCloudNativeProtection, keyExocompute,
					keySQLDBProtection, keySQLMIProtection,
				},
				Description: "Enable the RSC SQL MI Protection feature for the Azure subscription.",
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure subscription ID.",
				ValidateFunc: validation.IsUUID,
			},
			keySubscriptionName: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Azure subscription name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTenantDomain: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure tenant primary domain.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},

		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureSubscriptionV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureSubscriptionStateUpgradeV0,
			Version: 0,
		}, {
			Type:    resourceAzureSubscriptionV1().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureSubscriptionStateUpgradeV1,
			Version: 1,
		}},
	}
}

// azureCreateSubscription run the Create operation for the Azure subscription
// resource. This adds the Azure subscription to the RSC platform.
func azureCreateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionID, err := uuid.Parse(d.Get(keySubscriptionID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	tenantDomain := d.Get(keyTenantDomain).(string)

	var opts []azure.OptionFunc
	if name, ok := d.GetOk(keySubscriptionName); ok {
		opts = append(opts, azure.Name(name.(string)))
	}

	var accountID uuid.UUID
	for key, feature := range azureKeyFeatureMap {
		block, ok := d.GetOk(key)
		if ok {
			featureBlock := block.([]any)[0].(map[string]any)

			var featureOpts []azure.OptionFunc
			for _, region := range featureBlock[keyRegions].(*schema.Set).List() {
				featureOpts = append(featureOpts, azure.Region(region.(string)))
			}
			if rgOpt, ok := resourceGroup(featureBlock); ok {
				featureOpts = append(featureOpts, rgOpt)
			}

			// sql_db_protection and sql_mi_protection do not support resource
			// groups.
			if rgOpt, ok := resourceGroup(featureBlock); ok {
				featureOpts = append(featureOpts, rgOpt)
			}

			featureOpts = append(featureOpts, opts...)
			accountID, err = azure.Wrap(client).AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain),
				feature, featureOpts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId(accountID.String())
	azureReadSubscription(ctx, d, m)
	return nil
}

// azureReadSubscription run the Read operation for the Azure subscription
// resource. This reads the remote state of the Azure subscription in RSC.
func azureReadSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := azure.Wrap(client).Subscription(ctx, azure.CloudAccountID(accountID), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	} else if err != nil {
		return diag.FromErr(err)
	}

	for key, feature := range azureKeyFeatureMap {
		feature, ok := account.Feature(feature)
		if !ok {
			if err := d.Set(key, nil); err != nil {
				return diag.FromErr(err)
			}
			continue
		}

		regions := schema.Set{F: schema.HashString}
		for _, region := range feature.Regions {
			regions.Add(region)
		}
		status := string(feature.Status)
		if feature.SupportResourceGroup() {
			tags := make(map[string]any, len(feature.ResourceGroup.Tags))
			for key, value := range feature.ResourceGroup.Tags {
				tags[key] = value
			}
			err = d.Set(key, []any{
				map[string]any{
					keyRegions:             &regions,
					keyResourceGroupName:   feature.ResourceGroup.Name,
					keyResourceGroupRegion: feature.ResourceGroup.Region,
					keyResourceGroupTags:   tags,
					keyStatus:              status,
				},
			})
		} else {
			err = d.Set(key, []any{
				map[string]any{
					keyRegions: &regions,
					keyStatus:  status,
				},
			})
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keySubscriptionID, account.NativeID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySubscriptionName, account.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTenantDomain, account.TenantDomain); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// azureUpdateSubscription run the Update operation for the Azure subscription
// resource. This updates the Azure subscription in RSC.
func azureUpdateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Classify the feature changes as either add, remove or update.
	// The classification determines the order in which the changes are applied.
	// Add changes must be applied before remove changes, otherwise there is a
	// risk that he tenant is removed during the update.
	type change struct {
		feature core.Feature
		block   map[string]any
		order   int
	}
	changes := make([]change, 0)
	for key, feature := range azureKeyFeatureMap {
		if !d.HasChange(key) {
			continue
		}
		switch oldBlock, newBlock := d.GetChange(key); {
		case len(oldBlock.([]any)) == 0:
			changes = append(changes, change{feature: feature, block: newBlock.([]any)[0].(map[string]any), order: 1})
		case len(newBlock.([]any)) == 0:
			changes = append(changes, change{feature: feature, order: 2})
		default:
			changes = append(changes, change{feature: feature, block: newBlock.([]any)[0].(map[string]any), order: 3})
		}
	}
	slices.SortFunc(changes, func(i, j change) int {
		return cmp.Compare(i.order, j.order)
	})

	// Apply the changes in the correct order.
	for _, change := range changes {
		switch change.order {
		case 1:
			subscriptionID, err := uuid.Parse(d.Get(keySubscriptionID).(string))
			if err != nil {
				return diag.FromErr(err)
			}
			var opts []azure.OptionFunc
			for _, region := range change.block[keyRegions].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}
			if rgOpt, ok := resourceGroup(change.block); ok {
				opts = append(opts, rgOpt)
			}
			_, err = azure.Wrap(client).AddSubscription(ctx,
				azure.Subscription(subscriptionID, d.Get(keyTenantDomain).(string)), change.feature, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		case 2:
			err := azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), change.feature,
				d.Get(keyDeleteSnapshotsOnDestroy).(bool))
			if err != nil {
				return diag.FromErr(err)
			}
		case 3:
			var opts []azure.OptionFunc
			for _, region := range change.block[keyRegions].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}
			err := azure.Wrap(client).UpdateSubscription(ctx, azure.CloudAccountID(accountID), change.feature, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keySubscriptionName) {
		opts := []azure.OptionFunc{azure.Name(d.Get(keySubscriptionName).(string))}
		err = azure.Wrap(client).UpdateSubscription(ctx, azure.CloudAccountID(accountID),
			core.FeatureCloudNativeProtection, opts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadSubscription(ctx, d, m)
	return nil
}

// azureDeleteSubscription run the Delete operation for the Azure subscription
// resource. This removes the Azure subscription from RSC.
func azureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	for key, feature := range azureKeyFeatureMap {
		if _, ok := d.GetOk(key); !ok {
			continue
		}

		err = azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), feature,
			d.Get(keyDeleteSnapshotsOnDestroy).(bool))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

// azureKeyFeatureMap maps the subscription resource's Terraform keys to RSC
// features.
var azureKeyFeatureMap = map[string]core.Feature{
	keyCloudNativeArchival:           core.FeatureCloudNativeArchival,
	keyCloudNativeArchivalEncryption: core.FeatureCloudNativeArchivalEncryption,
	keyCloudNativeProtection:         core.FeatureCloudNativeProtection,
	keyExocompute:                    core.FeatureExocompute,
	keySQLDBProtection:               core.FeatureAzureSQLDBProtection,
	keySQLMIProtection:               core.FeatureAzureSQLMIProtection,
}

// resourceGroup extracts the resource group information from the feature block.
func resourceGroup(block map[string]any) (azure.OptionFunc, bool) {
	name, nameOk := block[keyResourceGroupName]
	region, regionOk := block[keyResourceGroupRegion]

	tags := make(map[string]string)
	if rgTags, ok := block[keyResourceGroupTags]; ok {
		for key, value := range rgTags.(map[string]any) {
			tags[key] = value.(string)
		}
	}

	// If any part of the resource group is set, it's considered to be a valid
	// resource group. Proper validation will be handled by the backend.
	if nameOk || regionOk || len(tags) > 0 {
		return azure.ResourceGroup(name.(string), region.(string), tags), true
	}

	return nil, false
}
