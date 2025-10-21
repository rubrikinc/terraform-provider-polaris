// Copyright 2024 Rubrik, Inc.
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
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
)

const resourceAzureArchivalLocationDescription = `
The ´polaris_azure_archival_location´ resource creates an RSC archival location for
cloud-native workloads. This resource requires that the Azure subscription has been
onboarded with the ´cloud_native_archival´ feature.

When creating an archival location, the region where the snapshots are stored needs
to be specified:
  * ´SOURCE_REGION´ - Store snapshots in the same region to minimize data transfer
    charges. This is the default behaviour when the ´storage_account_region´ field is
    not specified.
  * ´SPECIFIC_REGION´ - Storing snapshots in another region can increase total data
    transfer charges. The ´storage_account_region´ field specifies the region.

Custom storage encryption is enabled by specifying one or more ´customer_managed_key´
blocks. Each ´customer_managed_key´ block specifies the encryption details to use for
a region. For other regions, data will be encrypted using platform managed keys.

-> **Note:** The Azure storage account is not created until the first protected object
   is archived to the location.
`

func resourceAzureArchivalLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateArchivalLocation,
		ReadContext:   azureReadArchivalLocation,
		UpdateContext: azureUpdateArchivalLocation,
		DeleteContext: azureDeleteArchivalLocation,

		Description: description(resourceAzureArchivalLocationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud native archival location ID (UUID).",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the cloud native archival location.",
			},
			keyContainerName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Azure storage container name.",
			},
			keyCustomerManagedKey: {
				Type:     schema.TypeSet,
				Elem:     customerKeyResource(),
				Optional: true,
				Description: "Customer managed storage encryption. Specify the regions and their respective " +
					"encryption details. For other regions, data will be encrypted using platform managed keys.",
			},
			keyLocationTemplate: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "RSC location template. If a storage account region was specified, it will be " +
					"`SPECIFIC_REGION`, otherwise `SOURCE_REGION`.",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Cloud native archival location name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRedundancy: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "LRS",
				Description: "Azure storage redundancy. Possible values are `GRS`, `GZRS`, `LRS`, `RA_GRS`, " +
					"`RA_GZRS` and `ZRS`. Default value is `LRS`. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice([]string{"GRS", "GZRS", "LRS", "RA_GRS", "RA_GZRS", "ZRS"}, false),
			},
			keyStorageAccountNamePrefix: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Azure storage account name prefix. The storage account name prefix cannot be longer " +
					"than 14 characters and can only consist of numbers and lower case letters. Changing this forces " +
					"a new resource to be created.",
				ValidateFunc: validation.All(validation.StringLenBetween(1, 14),
					validation.StringMatch(regexp.MustCompile("^[a-z0-9]*$"),
						"storage account name may only contain numbers and lowercase letters")),
			},
			keyStorageAccountRegion: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "Azure region to store the snapshots in. If not specified, the snapshots will be stored " +
					"in the same region as the workload. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(azure.AllRegionNames(), false),
			},
			keyStorageAccountTags: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Description: "Azure storage account tags. Each tag will be added to the storage account created by " +
					"RSC.",
			},
			keyStorageTier: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "COOL",
				Description:  "Azure storage tier. Possible values are `COOL` and `HOT`. Default value is `COOL`.",
				ValidateFunc: validation.StringInSlice([]string{"COOL", "HOT"}, false),
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func azureCreateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureCreateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var storageAccountTags *core.Tags
	if tags := toAzureStorageAccountTags(d.Get(keyStorageAccountTags).(map[string]any)); len(tags.TagList) > 0 {
		storageAccountTags = &tags
	}

	// Create the Azure cloud native archival location.
	targetMappingID, err := archival.Wrap(client).CreateAzureStorageSetting(ctx, gqlarchival.CreateAzureStorageSettingParams{
		CloudAccountID:       cloudAccountID,
		Name:                 d.Get(keyName).(string),
		Redundancy:           d.Get(keyRedundancy).(string),
		StorageTier:          d.Get(keyStorageTier).(string),
		StorageAccountName:   d.Get(keyStorageAccountNamePrefix).(string),
		StorageAccountRegion: azure.RegionFromName(d.Get(keyStorageAccountRegion).(string)).ToRegionEnum(),
		StorageAccountTags:   storageAccountTags,
		CMKInfo:              toAzureCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMappingID.String())
	azureReadArchivalLocation(ctx, d, m)
	return nil
}

func azureReadArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the Azure cloud native archival location. If the archival location
	// isn't found, we remove it from the local state and return.
	targetMapping, err := archival.Wrap(client).AzureTargetMappingByID(ctx, targetMappingID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	targetTemplate := targetMapping.TargetTemplate
	cloudNativeCompanion := targetTemplate.CloudNativeCompanion
	if err := d.Set(keyCloudAccountID, targetMapping.TargetTemplate.CloudAccount.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyContainerName, targetTemplate.ContainerNamePrefix); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCustomerManagedKey, fromAzureCustomerManagedKeys(cloudNativeCompanion.CMKInfo)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyLocationTemplate, cloudNativeCompanion.LocTemplate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, targetMapping.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRedundancy, cloudNativeCompanion.Redundancy); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountNamePrefix, targetTemplate.StorageAccountName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountRegion, cloudNativeCompanion.StorageAccountRegion.Name()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountTags, fromAzureStorageAccountTags(cloudNativeCompanion.StorageAccountTags)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageTier, cloudNativeCompanion.StorageTier); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureUpdateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureUpdateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Update the Azure cloud native archival location. Note, the API doesn't
	// support updating all arguments.
	err = archival.Wrap(client).UpdateAzureStorageSetting(ctx, targetMappingID, gqlarchival.UpdateAzureStorageSettingParams{
		Name:               d.Get(keyName).(string),
		StorageTier:        d.Get(keyStorageTier).(string),
		StorageAccountTags: toAzureStorageAccountTags(d.Get(keyStorageAccountTags).(map[string]any)),
		CMKInfo:            toAzureCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureDeleteArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete the Azure cloud native archival location.
	if err := archival.Wrap(client).DeleteTargetMapping(ctx, targetMappingID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// customerKeyResource returns the schema for a customer managed key resource.
func customerKeyResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Key name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Required: true,
				Description: "The region in which the key will be used. Regions without customer managed keys will " +
					"use platform managed keys.",
				ValidateFunc: validation.StringInSlice(azure.AllRegionNames(), false),
			},
			keyVaultName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Key vault name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

// toAzureCustomerManagedKeys converts from the customer managed keys field type
// to a slice of Azure customer keys.
func toAzureCustomerManagedKeys(keys *schema.Set) []gqlarchival.AzureCustomerKey {
	var customerKeys []gqlarchival.AzureCustomerKey
	for _, key := range keys.List() {
		key := key.(map[string]any)
		customerKeys = append(customerKeys, gqlarchival.AzureCustomerKey{
			Name:      key[keyName].(string),
			VaultName: key[keyVaultName].(string),
			Region:    azure.RegionFromName(key[keyRegion].(string)).ToRegionEnum(),
		})
	}

	return customerKeys
}

// fromAzureCustomerManagedKeys converts to the customer managed keys field type from
// a slice of Azure customer keys.
func fromAzureCustomerManagedKeys(customerKeys []gqlarchival.AzureCustomerKey) *schema.Set {
	keys := &schema.Set{F: schema.HashResource(customerKeyResource())}
	for _, key := range customerKeys {
		keys.Add(map[string]any{
			keyName:      key.Name,
			keyVaultName: key.VaultName,
			keyRegion:    key.Region,
		})
	}

	return keys
}

// toAzureStorageAccountTags converts from the storage account tags field type
// to the Azure tags type.
func toAzureStorageAccountTags(tags map[string]any) core.Tags {
	tagList := make([]core.Tag, 0, len(tags))
	for key, value := range tags {
		tagList = append(tagList, core.Tag{Key: key, Value: value.(string)})
	}

	return core.Tags{TagList: tagList}
}

// fromAzureStorageAccountTags converts to the storage account tags field type
// from the Azure tags type.
func fromAzureStorageAccountTags(storageAccountTags []core.Tag) map[string]any {
	tags := make(map[string]any, len(storageAccountTags))
	for _, tag := range storageAccountTags {
		tags[tag.Key] = tag.Value
	}

	return tags
}
