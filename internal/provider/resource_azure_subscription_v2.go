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
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
)

func resourceAzureSubscriptionV2() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyCloudNativeArchival: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "ENCRYPTION", "SQL_ARCHIVAL",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Archival feature. " +
								"Possible values are `BASIC`, `ENCRYPTION` and `SQL_ARCHIVAL`.",
						},
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
							Description: "Region of the Azure resource group. Should be specified in the standard " +
								"Azure style, e.g. `eastus`. Changing this forces the RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
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
					keyCloudNativeBlobProtection,
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
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "ENCRYPTION",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Archival Encryption " +
								"feature. Possible values are `BASIC` and `ENCRYPTION`.",
						},
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
							Description: "Region of the Azure resource group. Should be specified in the standard " +
								"Azure style, e.g. `eastus`. Changing this forces the RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
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
							Type:     schema.TypeString,
							Required: true,
							Description: "User-assigned managed identity region. Should be specified in the " +
								"standard Azure style, e.g. `eastus`.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
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
			keyCloudNativeBlobProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "RECOVERY",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Blob Protection feature. " +
								"Possible values are `BASIC` and `RECOVERY`.",
						},
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
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Blob Protection feature.",
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
				Description: "Enable the RSC Cloud Native Protection feature for Azure Blob Storage.",
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY", "CLOUD_CLUSTER_ES",
									"SNAPSHOT_PRIVATE_ACCESS",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Protection feature. " +
								"Possible values are `BASIC`, `EXPORT_AND_RESTORE`, `FILE_LEVEL_RECOVERY`, " +
								"`CLOUD_CLUSTER_ES` and `SNAPSHOT_PRIVATE_ACCESS`.",
						},
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
							Description: "Region of the Azure resource group. Should be specified in the standard " +
								"Azure style, e.g. `eastus`. Changing this forces the RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
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
					keyCloudNativeBlobProtection,
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
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "PRIVATE_ENDPOINTS", "CUSTOMER_MANAGED_BASIC",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Exocompute feature. Possible values " +
								"are `BASIC`, `PRIVATE_ENDPOINTS` and `CUSTOMER_MANAGED_BASIC`.",
						},
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
							Description: "Region of the Azure resource group. Should be specified in the standard " +
								"Azure style, e.g. `eastus`. Changing this forces the RSC feature to be re-onboarded.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
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
					keyCloudNativeBlobProtection,
					keyCloudNativeProtection,
					keySQLDBProtection,
					keySQLMIProtection,
				},
				Description: "Enable the RSC Exocompute feature for the Azure subscription.",
			},
			keySQLDBProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "RECOVERY", "BACKUP_V2",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the SQL DB Protection feature. " +
								"Possible values are `BASIC`, `RECOVERY` and `BACKUP_V2`.",
						},
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
					keyCloudNativeBlobProtection,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLMIProtection,
				},
				Description: "Enable the RSC SQL DB Protection feature for the Azure subscription.",
			},
			keySQLMIProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "RECOVERY", "BACKUP_V2",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the SQL MI Protection feature. " +
								"Possible values are `BASIC`, `RECOVERY` and `BACKUP_V2`.",
						},
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
					keyCloudNativeBlobProtection,
					keyCloudNativeProtection,
					keyExocompute,
					keySQLDBProtection,
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
	}
}

// resourceAzureSubscriptionStateUpgradeV1 introduces a cloud native protection
// feature block.
func resourceAzureSubscriptionStateUpgradeV2(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	log.Print("[TRACE] azureSubscriptionStateUpgradeV2")
	return state, nil
}
