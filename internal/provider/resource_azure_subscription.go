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
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAzureSubscriptionDescription = `
The ´polaris_azure_subscription´ resource adds an Azure subscription to RSC.
When the first subscription for an Azure tenant is added, a corresponding tenant
is created in RSC. The RSC tenant is automatically destroyed when it's last
subscription is removed.

Each feature's ´permissions´ field can be used with the
´polaris_azure_permissions´ data source to inform RSC about permission updates
when the Terraform configuration is applied.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

´AZURE_SQL_DB_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.
  * ´BACKUP_V2´ - Represents the set of permissions required for immutable
    backup V2 operations.

´AZURE_SQL_MI_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.
  * ´BACKUP_V2´ - Represents the set of permissions required for immutable
    backup V2 operations.

´CLOUD_NATIVE_ARCHIVAL´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´ENCRYPTION´ - Represents the set of permissions required for encryption
    operation.
  * ´SQL_ARCHIVAL´ - Represents the permissions required to enable Azure AD
    authorization to store Azure SQL and MI snapshots in an archival location.

´CLOUD_NATIVE_ARCHIVAL_ENCRYPTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´ENCRYPTION´ - Represents the set of permissions required for encryption
    operation.

´CLOUD_NATIVE_BLOB_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RECOVERY´ - Represents the set of permissions required for all recovery
    operations.

´CLOUD_NATIVE_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´EXPORT_AND_RESTORE´ - Represents the set of permissions required for export
    and restore operations.
  * ´FILE_LEVEL_RECOVERY´ - Represents the set of permissions required for
    file-level recovery operations.
  * ´SNAPSHOT_PRIVATE_ACCESS´ - Represents the set of permissions required for
    private access to disk snapshots.

´EXOCOMPUTE´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´PRIVATE_ENDPOINTS´ - Represents the set of permissions required for usage
    of private endpoints.
  * ´CUSTOMER_MANAGED_BASIC´ - Represents the permissions required to enable
    customer-managed Exocompute feature.

~> **Note:** Even though the ´resource_group_name´ and the
   ´resource_group_region´ fields are marked as optional you should always
   specify them. They are marked as optional to simplify the migration of
   existing Terraform configurations. If omitted, RSC will generate a unique
   resource group name but it will not create the actual resource group. Until
   the resource group is created, the RSC feature depending on the resource
   group will not function as expected.

~> **Note:** As mentioned in the documentation for each feature below, changing
   certain fields causes features to be re-onboarded. Take care when the
   subscription only has a single feature, as it could cause the tenant to be
   removed from RSC.

-> **Note:** As of now, ´sql_mi_protection´ does not support specifying an Azure
   resource group.
`

func resourceAzureSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateSubscription,
		ReadContext:   azureReadSubscription,
		UpdateContext: azureUpdateSubscription,
		DeleteContext: azureDeleteSubscription,

		Description: description(resourceAzureSubscriptionDescription),
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
				Description: "Enable the RSC Cloud Native Archival feature for the Azure subscription. Provides " +
					"archival of data from workloads for disaster recovery and long-term retention.",
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
				Description: "Enable the RSC Cloud Native Archival Encryption feature for the Azure subscription. " +
					"Allows cloud archival locations to be encrypted with customer managed keys.",
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
				Description: "Enable the RSC Cloud Native Protection feature for Azure Blob Storage. Provides " +
					"protection for Azure Blob Storage through the rules and policies of SLA Domains.",
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
				Description: "Enable the RSC Cloud Native Protection feature for the Azure subscription. Provides " +
					"protection for Azure virtual machines and managed disks through the rules and policies of SLA " +
					"Domains.",
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
				Description: "Enable the RSC Exocompute feature for the Azure subscription. Provides snapshot " +
					"indexing, file recovery, storage tiering, and application-consistent protection of Azure objects.",
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
						keyResourceGroupName: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keySQLDBProtection + ".0." + keyResourceGroupRegion,
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
								keySQLDBProtection + ".0." + keyResourceGroupName,
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
								keySQLDBProtection + ".0." + keyResourceGroupName,
								keySQLDBProtection + ".0." + keyResourceGroupRegion,
							},
							Description: "Tags to add to the Azure resource group. Changing this forces the RSC feature " +
								"to be re-onboarded.",
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
				Description: "Enable the RSC SQL DB Protection feature for the Azure subscription. Provides " +
					"centralized database backup management and recovery in an Azure SQL Database deployment.",
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
				Description: "Enable the RSC SQL MI Protection feature for the Azure subscription. Provides " +
					"centralized database backup management and recovery for an Azure SQL Managed Instance deployment.",
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
	tflog.Trace(ctx, "azureCreateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	featureKeys := make([]featureKey, 0, len(azureKeyFeatureMap))
	for key, feature := range azureKeyFeatureMap {
		featureKeys = append(featureKeys, featureKey{key: key, feature: feature.feature, order: feature.orderAdd})
	}
	slices.SortFunc(featureKeys, func(i, j featureKey) int {
		return cmp.Compare(i.order, j.order)
	})

	var accountID uuid.UUID
	for _, featureKey := range featureKeys {
		var block map[string]any
		if v, ok := d.GetOk(featureKey.key); ok {
			block = v.([]any)[0].(map[string]any)
		} else {
			continue
		}

		id, err := addAzureFeature(ctx, d, client, featureKey.feature, block)
		if err != nil {
			return diag.FromErr(err)
		}
		if accountID == uuid.Nil {
			accountID = id
		}
		if id != accountID {
			return diag.Errorf("feature %s added to wrong cloud account", featureKey.feature)
		}
	}

	d.SetId(accountID.String())
	azureReadSubscription(ctx, d, m)
	return nil
}

// azureReadSubscription run the Read operation for the Azure subscription
// resource. This reads the remote state of the Azure subscription in RSC.
func azureReadSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadSubscription")

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
		feature, ok := account.Feature(feature.feature)
		if !ok {
			if err := d.Set(key, nil); err != nil {
				return diag.FromErr(err)
			}
			continue
		}
		if err := updateAzureFeatureState(d, key, feature); err != nil {
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
	tflog.Trace(ctx, "azureUpdateSubscription")

	client := m.(*client)
	polarisClient, err := client.polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Break the update into a series of update operations sequenced in the
	// correct order.
	const (
		opAddFeature = iota
		opRemoveFeature
		opTemporaryRemoveFeature
		opUpdateSubscription
		opUpdatePermissions
	)
	type updateOp struct {
		feature core.Feature
		op      int
		block   map[string]any
		order   int
	}
	var updates []updateOp
	for key, feature := range azureKeyFeatureMap {
		if !d.HasChange(key) {
			continue
		}

		switch oldBlock, newBlock := d.GetChange(key); {
		case len(oldBlock.([]any)) == 0 && len(newBlock.([]any)) != 0:
			updates = append(updates, updateOp{
				op:      opAddFeature,
				feature: feature.feature,
				block:   newBlock.([]any)[0].(map[string]any),
				order:   feature.orderAdd,
			})

		case len(oldBlock.([]any)) != 0 && len(newBlock.([]any)) == 0:
			updates = append(updates, updateOp{
				op:      opRemoveFeature,
				feature: feature.feature,
				order:   feature.orderRemove,
			})

		case len(oldBlock.([]any)) != 0 && len(newBlock.([]any)) != 0:
			oldBlock := oldBlock.([]any)[0].(map[string]any)
			newBlock := newBlock.([]any)[0].(map[string]any)

			// Try to upgrade the Azure SQL DB Protection feature to use a
			// resource group.
			if feature.feature.Equal(core.FeatureAzureSQLDBProtection) {
				ok, err := upgradeSQLDBFeatureToUseResourceGroup(ctx, client, accountID, newBlock)
				if err != nil {
					return diag.FromErr(err)
				}
				if ok {
					continue
				}
			}

			// Changes in resource group or managed identity requires the
			// feature to be re-onboarded, any other changes to the feature will
			// be updated when the feature is re-onboarded.
			if diffAzureFeatureResourceGroup(oldBlock, newBlock) || diffAzureUserAssignedManagedIdentity(oldBlock, newBlock) {
				updates = append(updates, updateOp{
					op:      opAddFeature,
					feature: feature.feature,
					block:   newBlock,
					order:   feature.orderSplitAdd,
				})
				updates = append(updates, updateOp{
					op:      opTemporaryRemoveFeature,
					feature: feature.feature,
					order:   feature.orderSplitRemove,
				})
				continue
			}
			if diffAzureFeaturePermissionGroups(oldBlock, newBlock) || diffAzureFeatureRegions(oldBlock, newBlock) {
				updates = append(updates, updateOp{
					op:      opUpdateSubscription,
					feature: feature.feature,
					block:   newBlock,
				})
			}
			if diffAzureFeaturePermissions(newBlock, oldBlock) {
				updates = append(updates, updateOp{
					op:      opUpdatePermissions,
					feature: feature.feature,
				})
			}
		}
	}
	slices.SortFunc(updates, func(i, j updateOp) int {
		return cmp.Compare(i.order, j.order)
	})

	// Apply the update operations in the correct order.
	for _, update := range updates {
		feature := update.feature

		switch update.op {
		case opAddFeature:
			id, err := addAzureFeature(ctx, d, polarisClient, feature, update.block)
			if err != nil {
				return diag.FromErr(err)
			}
			if id != accountID {
				return diag.Errorf("feature %s added to the wrong cloud account", feature)
			}
		case opRemoveFeature, opTemporaryRemoveFeature:
			deleteSnapshots := false
			if update.op == opRemoveFeature {
				deleteSnapshots = d.Get(keyDeleteSnapshotsOnDestroy).(bool)
			}
			if err := azure.Wrap(polarisClient).RemoveSubscription(ctx, azure.CloudAccountID(accountID), feature, deleteSnapshots); err != nil {
				return diag.FromErr(err)
			}
		case opUpdateSubscription:
			for _, permGroup := range update.block[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(permGroup.(string)))
			}
			var opts []azure.OptionFunc
			for _, region := range update.block[keyRegions].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}
			if err := azure.Wrap(polarisClient).UpdateSubscription(ctx, azure.CloudAccountID(accountID), feature, opts...); err != nil {
				return diag.FromErr(err)
			}
		case opUpdatePermissions:
			if err := azure.Wrap(polarisClient).PermissionsUpdated(ctx, azure.CloudAccountID(accountID), []core.Feature{feature}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keySubscriptionName) {
		opts := []azure.OptionFunc{azure.Name(d.Get(keySubscriptionName).(string))}
		if err = azure.Wrap(polarisClient).UpdateSubscription(ctx, azure.CloudAccountID(accountID), core.FeatureAll, opts...); err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadSubscription(ctx, d, m)
	return nil
}

// azureDeleteSubscription run the Delete operation for the Azure subscription
// resource. This removes the Azure subscription from RSC.
func azureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Remove features in the correct order.
	featureKeys := make([]featureKey, 0, len(azureKeyFeatureMap))
	for key, feature := range azureKeyFeatureMap {
		featureKeys = append(featureKeys, featureKey{key: key, feature: feature.feature, order: feature.orderRemove})
	}
	slices.SortFunc(featureKeys, func(i, j featureKey) int {
		return cmp.Compare(i.order, j.order)
	})

	for _, featureKey := range featureKeys {
		if _, ok := d.GetOk(featureKey.key); !ok {
			continue
		}

		deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
		err = azure.Wrap(client).RemoveSubscription(ctx, azure.CloudAccountID(accountID), featureKey.feature, deleteSnapshots)
		if err != nil && !errors.Is(err, graphql.ErrNotFound) {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

// featureKey maps a Terraform configuration key to an RSC feature along with
// order information.
type featureKey struct {
	key     string
	feature core.Feature
	order   int
}

// orderedFeature holds the feature and order information for the feature.
// The split order information is used when a feature needs to be re-onboarded
// due to a change in the configuration.
type orderedFeature struct {
	feature          core.Feature
	orderAdd         int
	orderRemove      int
	orderSplitAdd    int
	orderSplitRemove int
}

// azureKeyFeatureMap maps the subscription's Terraform keys to the RSC features
// and the feature's order information.
//
// Adds are performed first, to reduce the risk of tenant being removed due to
// the last RSC feature being removed. Next, we perform updates. An update can
// result in a feature being removed and added again. Lastly, feature removals
// are performed.
//
// Note, all operations must be performed in the correct order, due to the
// implicit relationship between CLOUD_NATIVE_ARCHIVAL and
// CLOUD_NATIVE_ARCHIVAL_ENCRYPTION.
var azureKeyFeatureMap = map[string]orderedFeature{
	keyCloudNativeArchival: {
		feature:          core.FeatureCloudNativeArchival,
		orderAdd:         100,
		orderRemove:      301,
		orderSplitAdd:    202,
		orderSplitRemove: 201,
	},
	keyCloudNativeArchivalEncryption: {
		feature:          core.FeatureCloudNativeArchivalEncryption,
		orderAdd:         101,
		orderRemove:      300,
		orderSplitAdd:    203,
		orderSplitRemove: 200,
	},
	keyCloudNativeBlobProtection: {
		feature:          core.FeatureCloudNativeBlobProtection,
		orderAdd:         102,
		orderRemove:      302,
		orderSplitAdd:    205,
		orderSplitRemove: 204,
	},
	keyCloudNativeProtection: {
		feature:          core.FeatureCloudNativeProtection,
		orderAdd:         103,
		orderRemove:      303,
		orderSplitAdd:    207,
		orderSplitRemove: 206,
	},
	keyExocompute: {
		feature:          core.FeatureExocompute,
		orderAdd:         104,
		orderRemove:      304,
		orderSplitAdd:    209,
		orderSplitRemove: 208,
	},
	keySQLDBProtection: {
		feature:          core.FeatureAzureSQLDBProtection,
		orderAdd:         105,
		orderRemove:      305,
		orderSplitAdd:    211,
		orderSplitRemove: 210,
	},
	keySQLMIProtection: {
		feature:          core.FeatureAzureSQLMIProtection,
		orderAdd:         106,
		orderRemove:      306,
		orderSplitAdd:    213,
		orderSplitRemove: 212,
	},
}

// addAzureFeature onboards the RSC feature for the Azure subscription.
func addAzureFeature(ctx context.Context, d *schema.ResourceData, client *polaris.Client, feature core.Feature, block map[string]any) (uuid.UUID, error) {
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

	if permGroups, ok := block[keyPermissionGroups]; ok {
		for _, permGroup := range permGroups.(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(permGroup.(string)))
		}
	}

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

	permGroups := schema.Set{F: schema.HashString}
	for _, permGroup := range feature.PermissionGroups {
		permGroups.Add(string(permGroup))
	}
	block[keyPermissionGroups] = &permGroups

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
	slices.SortFunc(oldRegions, func(i, j string) int {
		return cmp.Compare(i, j)
	})
	slices.SortFunc(newRegions, func(i, j string) int {
		return cmp.Compare(i, j)
	})

	return !slices.Equal(oldRegions, newRegions)
}

// diffAzureFeaturePermissionGroups returns true if the old and new permission
// groups blocks are different.
func diffAzureFeaturePermissionGroups(oldBlock, newBlock map[string]any) bool {
	var oldPermGroups []string
	if v, ok := oldBlock[keyPermissionGroups]; ok {
		for _, permGroup := range v.(*schema.Set).List() {
			oldPermGroups = append(oldPermGroups, permGroup.(string))
		}
	}
	var newPermGroups []string
	if v, ok := newBlock[keyPermissionGroups]; ok {
		for _, permGroup := range v.(*schema.Set).List() {
			newPermGroups = append(newPermGroups, permGroup.(string))
		}
	}
	slices.SortFunc(oldPermGroups, func(i, j string) int {
		return cmp.Compare(i, j)
	})
	slices.SortFunc(newPermGroups, func(i, j string) int {
		return cmp.Compare(i, j)
	})

	return !slices.Equal(oldPermGroups, newPermGroups)
}

// diffAzureFeaturePermissionGroups returns true if the old and new permissions
// strings are different.
func diffAzureFeaturePermissions(oldBlock, newBlock map[string]any) bool {
	return oldBlock[keyPermissions].(string) != newBlock[keyPermissions].(string)
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

// azureFeatureResourceGroup returns the resource group from the feature block.
func azureFeatureResourceGroup(block map[string]any) (*gqlazure.ResourceGroup, bool) {
	var name string
	if v, ok := block[keyResourceGroupName]; ok {
		name = v.(string)
	}

	var region gqlazure.Region
	if v, ok := block[keyResourceGroupRegion]; ok {
		region = gqlazure.RegionFromName(v.(string))
	}

	tagList := make([]core.Tag, 0)
	if v, ok := block[keyResourceGroupTags]; ok {
		for k, v := range v.(map[string]any) {
			tagList = append(tagList, core.Tag{Key: k, Value: v.(string)})
		}
	}

	if name == "" || region == gqlazure.RegionUnknown {
		return nil, false
	}

	return &gqlazure.ResourceGroup{
		Name:    name,
		Region:  region.ToCloudAccountRegionEnum(),
		TagList: gqlazure.TagList{Tags: tagList},
	}, true
}

// upgradeSQLDBFeatureToUseResourceGroup upgrades the Azure SQL DB Protection
// feature to use a resource group.
func upgradeSQLDBFeatureToUseResourceGroup(ctx context.Context, client *client, cloudAccountID uuid.UUID, block map[string]any) (bool, error) {
	polarisClient, err := client.polaris()
	if err != nil {
		return false, err
	}

	// Check if the SQL DB Copy Backup feature flag is enabled for the account.
	// We only need to upgrade accounts which has the feature flag enabled.
	if !client.flags["CNP_AZURE_SQL_DB_COPY_BACKUP"] {
		tflog.Debug(ctx, "skipping Azure SQL DB Protection feature upgrade: feature flag CNP_AZURE_SQL_DB_COPY_BACKUP is not enabled")
		return false, nil
	}

	// Read the subscription and check if the Azure SQL DB Protection feature
	// already has a resource group set. If the Azure SQL DB feature hasn't been
	// onboarded or already has a resource group set, we don't need to upgrade.
	account, err := azure.Wrap(polarisClient).Subscription(ctx, azure.CloudAccountID(cloudAccountID), core.FeatureAll)
	if err != nil {
		return false, err
	}
	feature, ok := account.Feature(core.FeatureAzureSQLDBProtection)
	if !ok {
		return false, nil
	}
	if feature.ResourceGroup.Name != "" || feature.ResourceGroup.Region != "" {
		tflog.Debug(ctx, "skipping Azure SQL DB Protection feature upgrade: feature already upgraded")
		return false, nil
	}

	// Fetch the resource group from the Azure SQL DB block in the Terraform
	// configuration. If the resource group is not set, we cannot upgrade.
	rg, ok := azureFeatureResourceGroup(block)
	if !ok {
		return false, nil
	}

	// Upgrade the Azure SQL DB feature to use a resource group.
	tflog.Info(ctx, "upgrading Azure SQL DB Protection feature to use resource group", map[string]any{
		"resource_group": rg.Name,
	})
	if err := gqlazure.Wrap(polarisClient.GQL).UpgradeCloudAccountPermissionsWithoutOAuth(ctx, cloudAccountID, feature.Feature, rg); err != nil {
		return false, err
	}

	return true, nil
}
