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
	"fmt"
	"log"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
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
			"Each feature's `permissions` field can be used with the `polaris_azure_permissions` data source to inform " +
			"RSC about permission updates when the Terraform configuration is applied.\n" +
			"\n" +
			"~> **Note:** Even though the `resource_group_name` and the `resource_group_region` fields are marked as " +
			"   optional you should always specify them. They are marked as optional to simplify the migration of " +
			"   existing Terraform configurations. If omitted, RSC will generate a unique resource group name but it " +
			"   will not create the actual resource group. Until the resource group is created, the RSC feature " +
			"   depending on the resource group will not function as expected.\n" +
			"\n" +
			"~> **Note:** As mentioned in the documentation for each feature below, changing certain fields causes " +
			"   features to re-onboarded. Take care when the subscription only has a single feature, as it could cause " +
			"   the tenant to be removed from RSC.\n" +
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
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeArchival + ".0." + keyResourceGroupRegion,
							},
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature. RSC assumes the resource group already exists. Changing this forces the " +
								"RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeArchival + ".0." + keyResourceGroupName,
							},
							Description: "Region of the Azure resource group. Changing this forces the RSC feature to " +
								"be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeArchival + ".0." + keyResourceGroupName,
								keyCloudNativeArchival + ".0." + keyResourceGroupRegion,
							},
							Description: "Tags to add to the Azure resource group. Changing this forces the RSC feature " +
								"to be re-onboarded.",
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
					keyCloudNativeArchival,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
					keySQLMIProtection,
				},
				Description: "Enable the RSC Cloud Native Archival feature for the Azure subscription.",
			},
			keyCloudNativeArchivalEncryption: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
							RequiredWith: []string{
								keyCloudNativeArchivalEncryption + ".0." + keyResourceGroupRegion,
							},
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature. RSC assumes the resource group already exists. Changing this forces the " +
								"RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeArchivalEncryption + ".0." + keyResourceGroupName,
							},
							Description: "Region of the Azure resource group. Changing this forces the RSC feature to " +
								"be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeArchivalEncryption + ".0." + keyResourceGroupName,
								keyCloudNativeArchivalEncryption + ".0." + keyResourceGroupRegion,
							},
							Description: "Tags to add to the Azure resource group. Changing this forces the RSC feature " +
								"to be re-onboarded.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Archival Encryption feature.",
						},
						keyUserAssignedManagedIdentityName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "User-assigned managed identity name.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUserAssignedManagedIdentityPrincipalID: {
							Type:     schema.TypeString,
							Required: true,
							Description: "ID of the service principal object associated with the user-assigned managed " +
								"identity.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUserAssignedManagedIdentityRegion: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "User-assigned managed identity region.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUserAssignedManagedIdentityResourceGroupName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "User-assigned managed identity resource group name.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				RequiredWith: []string{
					keyCloudNativeArchival,
				},
				Description: "Enable the RSC Cloud Native Archival Encryption feature for the Azure subscription.",
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
							RequiredWith: []string{
								keyCloudNativeProtection + ".0." + keyResourceGroupRegion,
							},
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature. RSC assumes the resource group already exists. Changing this forces the " +
								"RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeProtection + ".0." + keyResourceGroupName,
							},
							Description: "Region of the Azure resource group. Changing this forces the RSC feature to " +
								"be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
							RequiredWith: []string{
								keyCloudNativeProtection + ".0." + keyResourceGroupName,
								keyCloudNativeProtection + ".0." + keyResourceGroupRegion,
							},
							Description: "Tags to add to the Azure resource group. Changing this forces the RSC feature " +
								"to be re-onboarded.",
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
					keyCloudNativeArchival,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
					keySQLMIProtection,
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
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyExocompute + ".0." + keyResourceGroupRegion,
							},
							Description: "Name of the Azure resource group where RSC places all resources created by " +
								"the feature. RSC assumes the resource group already exists. Changing this forces the " +
								"RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupRegion: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyExocompute + ".0." + keyResourceGroupName,
							},
							Description: "Region of the Azure resource group. Changing this forces the RSC feature to " +
								"be re-onboarded.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupTags: {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional: true,
							RequiredWith: []string{
								keyExocompute + ".0." + keyResourceGroupName,
								keyExocompute + ".0." + keyResourceGroupRegion,
							},
							Description: "Tags to add to the Azure resource group. Changing this forces the RSC feature " +
								"to be re-onboarded.",
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
					keyCloudNativeArchival,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
					keySQLMIProtection,
				},
				Description: "Enable the RSC Exocompute feature for the Azure subscription.",
			},
			keySQLDBProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
					keyCloudNativeArchival,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
					keySQLMIProtection,
				},
				Description: "Enable the RSC SQL DB Protection feature for the Azure subscription.",
			},
			keySQLMIProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissions: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Permissions updated signal. When this field changes, the provider will notify " +
								"RSC that the permissions for the feature has been updated. Use this field with the " +
								"`polaris_azure_permissions` data source.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
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
					keyCloudNativeArchival,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
					keySQLMIProtection,
				},
				Description: "Enable the RSC SQL MI Protection feature for the Azure subscription.",
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure subscription ID. Changing this forces a new resource to be created.",
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
				Description:  "Azure tenant primary domain. Changing this forces a new resource to be created.",
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
func azureCreateSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] azureCreateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var accountID uuid.UUID
	for key := range azureKeyFeatureMap {
		var block map[string]any
		if v, ok := d.GetOk(key); ok {
			block = v.([]any)[0].(map[string]any)
		} else {
			continue
		}

		id, err := addAzureFeature(ctx, d, client, key, block)
		if err != nil {
			return diag.FromErr(err)
		}
		if accountID == uuid.Nil {
			accountID = id
		}
		if id != accountID {
			return diag.Errorf("feature %s added to wrong cloud account", azureKeyFeatureMap[key])
		}
	}

	d.SetId(accountID.String())
	azureReadSubscription(ctx, d, m)
	return nil
}

// azureReadSubscription run the Read operation for the Azure subscription
// resource. This reads the remote state of the Azure subscription in RSC.
func azureReadSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
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

	for featureKey, feature := range azureKeyFeatureMap {
		feature, ok := account.Feature(feature)
		if !ok {
			if err := d.Set(featureKey, nil); err != nil {
				return diag.FromErr(err)
			}
			continue
		}
		if err := updateAzureFeatureState(d, featureKey, feature); err != nil {
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
func azureUpdateSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
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
	// Add-changes must be applied before remove-changes, otherwise there is a
	// risk that he tenant is removed during the update.
	const (
		add    = 1
		remove = 2
		update = 3
	)
	type change struct {
		key      string
		oldBlock map[string]any
		newBlock map[string]any
		order    int
	}
	var changes []change
	for key := range azureKeyFeatureMap {
		if !d.HasChange(key) {
			continue
		}
		switch oldBlock, newBlock := d.GetChange(key); {
		case len(oldBlock.([]any)) == 0 && len(newBlock.([]any)) != 0:
			newBlock := newBlock.([]any)[0].(map[string]any)
			changes = append(changes, change{key: key, newBlock: newBlock, order: add})
		case len(oldBlock.([]any)) != 0 && len(newBlock.([]any)) == 0:
			changes = append(changes, change{key: key, order: remove})
		case len(oldBlock.([]any)) != 0 && len(newBlock.([]any)) != 0:
			oldBlock := oldBlock.([]any)[0].(map[string]any)
			newBlock := newBlock.([]any)[0].(map[string]any)
			changes = append(changes, change{key: key, oldBlock: oldBlock, newBlock: newBlock, order: update})
		default:
			return diag.Errorf("")
		}
	}
	slices.SortFunc(changes, func(i, j change) int {
		return cmp.Compare(i.order, j.order)
	})

	// Apply changes in order.
	for _, change := range changes {
		feature := azureKeyFeatureMap[change.key]

		switch change.order {
		case add:
			id, err := addAzureFeature(ctx, d, client, change.key, change.newBlock)
			if err != nil {
				return diag.FromErr(err)
			}
			if id != accountID {
				return diag.Errorf("feature %s added to wrong cloud account", feature)
			}
		case remove:
			deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
			if err := azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), feature, deleteSnapshots); err != nil {
				return diag.FromErr(err)
			}
		case update:
			if err := updateAzureFeature(ctx, d, client, accountID, change.key, change.oldBlock, change.newBlock); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keySubscriptionName) {
		opts := []azure.OptionFunc{azure.Name(d.Get(keySubscriptionName).(string))}
		if err = azure.Wrap(client).UpdateSubscription(ctx, azure.CloudAccountID(accountID), core.FeatureAll, opts...); err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadSubscription(ctx, d, m)
	return nil
}

// azureDeleteSubscription run the Delete operation for the Azure subscription
// resource. This removes the Azure subscription from RSC.
func azureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	for key, feature := range azureKeyFeatureMapReverse {
		if _, ok := d.GetOk(key); !ok {
			continue
		}

		deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
		if err = azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), feature, deleteSnapshots); err != nil {
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

// azureKeyFeatureMapReverse maps the subscription resource's Terraform keys to
// RSC features, but with the Cloud Native Archival and Cloud Native Archival
// Encryption features reversed.
var azureKeyFeatureMapReverse = map[string]core.Feature{
	keyCloudNativeArchivalEncryption: core.FeatureCloudNativeArchivalEncryption,
	keyCloudNativeArchival:           core.FeatureCloudNativeArchival,
	keyCloudNativeProtection:         core.FeatureCloudNativeProtection,
	keyExocompute:                    core.FeatureExocompute,
	keySQLDBProtection:               core.FeatureAzureSQLDBProtection,
	keySQLMIProtection:               core.FeatureAzureSQLMIProtection,
}

// updateAzureFeature updates the remote RSC feature with the local state
// information.
func updateAzureFeature(ctx context.Context, d *schema.ResourceData, client *polaris.Client, accountID uuid.UUID, key string, oldBlock, newBlock map[string]any) error {
	feature := azureKeyFeatureMap[key]

	// Both of these diffs requires the feature to be re-onboarded. Note, we
	// never delete the snapshots when a feature gets re-onboarded because of
	// a configuration change.
	if diffAzureFeatureResourceGroup(oldBlock, newBlock) || diffAzureUserAssignedManagedIdentity(oldBlock, newBlock) {
		if err := azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), feature, false); err != nil {
			return err
		}
		id, err := addAzureFeature(ctx, d, client, key, newBlock)
		if err != nil {
			return err
		}
		if id != accountID {
			return fmt.Errorf("feature %s added to wrong cloud account", feature)
		}

		return nil
	}

	// Region diffs can be updated in place.
	if diffAzureFeatureRegions(oldBlock, newBlock) {
		var opts []azure.OptionFunc
		for _, region := range newBlock[keyRegions].(*schema.Set).List() {
			opts = append(opts, azure.Region(region.(string)))
		}
		if err := azure.Wrap(client).UpdateSubscription(ctx, azure.CloudAccountID(accountID), feature, opts...); err != nil {
			return err
		}
	}

	if newBlock[keyPermissions] != oldBlock[keyPermissions] {
		if err := azure.Wrap(client).PermissionsUpdated(ctx, azure.CloudAccountID(accountID), []core.Feature{feature}); err != nil {
			return err
		}
	}

	return nil
}

// addAzureFeature onboards the RSC feature for the Azure subscription.
func addAzureFeature(ctx context.Context, d *schema.ResourceData, client *polaris.Client, key string, block map[string]any) (uuid.UUID, error) {
	id, err := uuid.Parse(d.Get(keySubscriptionID).(string))
	if err != nil {
		return uuid.Nil, err
	}

	var opts []azure.OptionFunc
	if name, ok := d.GetOk(keySubscriptionName); ok {
		opts = append(opts, azure.Name(name.(string)))
	}

	if regions, ok := block[keyRegions]; ok {
		for _, region := range regions.(*schema.Set).List() {
			opts = append(opts, azure.Region(region.(string)))
		}
	}
	if rgOpt, ok := fromAzureResourceGroup(block); ok {
		opts = append(opts, rgOpt)
	}
	if miOpt, ok := fromAzureUserAssignedManagedIdentity(block); ok {
		opts = append(opts, miOpt)
	}

	feature := azureKeyFeatureMap[key]
	return azure.Wrap(client).AddSubscription(ctx, azure.Subscription(id, d.Get(keyTenantDomain).(string)), feature, opts...)
}

// updateAzureFeatureState updates the local state with the feature information.
func updateAzureFeatureState(d *schema.ResourceData, key string, feature azure.Feature) error {
	var block map[string]any
	if v, ok := d.GetOk(key); ok {
		block = v.([]any)[0].(map[string]any)
	} else {
		block = make(map[string]any)
	}

	regions := schema.Set{F: schema.HashString}
	for _, region := range feature.Regions {
		regions.Add(region)
	}
	block[keyRegions] = &regions
	block[keyStatus] = string(feature.Status)

	if feature.SupportResourceGroup() {
		tags := make(map[string]any, len(feature.ResourceGroup.Tags))
		for key, value := range feature.ResourceGroup.Tags {
			tags[key] = value
		}
		block[keyResourceGroupName] = feature.ResourceGroup.Name
		block[keyResourceGroupRegion] = feature.ResourceGroup.Region
		block[keyResourceGroupTags] = tags
	}

	if err := d.Set(key, []any{block}); err != nil {
		return err
	}

	return nil
}

// fromAzureResourceGroup returns an OptionFunc holding the resource group
// information.
func fromAzureResourceGroup(block map[string]any) (azure.OptionFunc, bool) {
	var name string
	if v, ok := block[keyResourceGroupName]; ok {
		name = v.(string)
	}
	var region string
	if v, ok := block[keyResourceGroupRegion]; ok {
		region = v.(string)
	}
	tags := make(map[string]string)
	if rgTags, ok := block[keyResourceGroupTags]; ok {
		for key, value := range rgTags.(map[string]any) {
			tags[key] = value.(string)
		}
	}

	if name != "" || region != "" || len(tags) > 0 {
		return azure.ResourceGroup(name, region, tags), true
	}

	return nil, false
}

// fromAzureUserAssignedManagedIdentity returns an OptionFunc holding the
// user-assigned managed identity information.
func fromAzureUserAssignedManagedIdentity(block map[string]any) (azure.OptionFunc, bool) {
	var name string
	if v, ok := block[keyUserAssignedManagedIdentityName]; ok {
		name = v.(string)
	}
	var principalID string
	if v, ok := block[keyUserAssignedManagedIdentityPrincipalID]; ok {
		principalID = v.(string)
	}
	var region string
	if v, ok := block[keyUserAssignedManagedIdentityRegion]; ok {
		region = v.(string)
	}
	var rgName string
	if v, ok := block[keyUserAssignedManagedIdentityResourceGroupName]; ok {
		rgName = v.(string)
	}

	if name != "" || rgName != "" || principalID != "" || region != "" {
		return azure.ManagedIdentity(name, rgName, principalID, region), true
	}

	return nil, false
}

// diffAzureFeatureRegions returns true if the old and new regions are
// different.
func diffAzureFeatureRegions(oldBlock, newBlock map[string]any) bool {
	var oldRegions []string
	if v, ok := oldBlock[keyRegions]; ok {
		for _, region := range v.(*schema.Set).List() {
			oldRegions = append(oldRegions, region.(string))
		}
	}
	var newRegions []string
	if v, ok := newBlock[keyRegions]; ok {
		for _, region := range v.(*schema.Set).List() {
			newRegions = append(newRegions, region.(string))
		}
	}
	if !slices.Equal(oldRegions, newRegions) {
		return true
	}

	return false
}

// diffAzureFeatureResourceGroup returns true if the old and new resource group
// blocks are different.
func diffAzureFeatureResourceGroup(oldBlock, newBlock map[string]any) bool {
	var oldName string
	if v, ok := oldBlock[keyResourceGroupName]; ok {
		oldName = v.(string)
	}
	var newName string
	if v, ok := newBlock[keyResourceGroupName]; ok {
		newName = v.(string)
	}
	if newName != oldName {
		return true
	}

	var oldRegion string
	if v, ok := oldBlock[keyResourceGroupRegion]; ok {
		oldRegion = v.(string)
	}
	var newRegion string
	if v, ok := newBlock[keyResourceGroupRegion]; ok {
		newRegion = v.(string)
	}
	if newRegion != oldRegion {
		return true
	}

	oldTags := make(map[string]string)
	if v, ok := oldBlock[keyResourceGroupTags]; ok {
		for k, v := range v.(map[string]any) {
			oldTags[k] = v.(string)
		}
	}
	newTags := make(map[string]string)
	if v, ok := newBlock[keyResourceGroupTags]; ok {
		for k, v := range v.(map[string]any) {
			newTags[k] = v.(string)
		}
	}
	if !maps.Equal(oldTags, newTags) {
		return true
	}

	return false
}

// diffAzureUserAssignedManagedIdentity returns true if the old and new
// user-assigned managed identities blocks are different.
func diffAzureUserAssignedManagedIdentity(oldBlock, newBlock map[string]any) bool {
	var oldName string
	if v, ok := oldBlock[keyUserAssignedManagedIdentityName]; ok {
		oldName = v.(string)
	}
	var newName string
	if v, ok := newBlock[keyUserAssignedManagedIdentityName]; ok {
		newName = v.(string)
	}
	if newName != oldName {
		return true
	}

	var oldRGName string
	if v, ok := oldBlock[keyUserAssignedManagedIdentityResourceGroupName]; ok {
		oldRGName = v.(string)
	}
	var newRGName string
	if v, ok := newBlock[keyUserAssignedManagedIdentityResourceGroupName]; ok {
		newRGName = v.(string)
	}
	if newRGName != oldRGName {
		return true
	}

	var oldPrincipalID string
	if v, ok := oldBlock[keyUserAssignedManagedIdentityPrincipalID]; ok {
		oldPrincipalID = v.(string)
	}
	var newPrincipalID string
	if v, ok := newBlock[keyUserAssignedManagedIdentityPrincipalID]; ok {
		newPrincipalID = v.(string)
	}
	if newPrincipalID != oldPrincipalID {
		return true
	}

	var oldRegion string
	if v, ok := oldBlock[keyUserAssignedManagedIdentityRegion]; ok {
		oldRegion = v.(string)
	}
	var newRegion string
	if v, ok := newBlock[keyUserAssignedManagedIdentityRegion]; ok {
		newRegion = v.(string)
	}
	if newRegion != oldRegion {
		return true
	}

	return false
}
