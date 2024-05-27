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
	"fmt"
	"log"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

func resourceAzureArchivalLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateArchivalLocation,
		ReadContext:   azureReadArchivalLocation,
		UpdateContext: azureUpdateArchivalLocation,
		DeleteContext: azureDeleteArchivalLocation,

		Description: "The `polaris_azure_archival_location` resource creates an RSC archival location for cloud-native " +
			"workloads.\n" +
			"\n" +
			"When creating an archival location, the region where the snapshots are stored needs to be specified:\n" +
			"  * *Source Region* - Store snapshots in the same region to minimize data transfer charges. This is the " +
			"    default behaviour when the `storage_account_region` field is not specified.\n" +
			"  * *Specific region* - Storing snapshots in another region can increase total data transfer charges. " +
			"    The `storage_account_region` field specifies the region.\n" +
			"\n" +
			"Custom storage encryption is enabled by specifying one or more `customer_managed_key` blocks. Each " +
			"`customer_managed_key` block specifies the encryption details to use for a region. For other regions, " +
			"data will be encrypted using platform managed keys. \n" +
			"\n" +
			"-> **Note:** The Azure storage account is not created until the first protected object is archived to the" +
			"   location.\n" +
			"\n",
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud native archival location ID.",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID.",
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
				Description: "Customer managed storage encryption. Specify the regions and their respective encryption " +
					"details. For other regions, data will be encrypted using platform managed keys.",
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
				Description: "Azure storage redundancy. Possible values are `GRS`, `GZRS`, `LRS`, `RA_GRS`, `RA_GZRS` " +
					"and `ZRS`. Default value is `LRS`.",
				ValidateFunc: validation.StringInSlice([]string{"GRS", "GZRS", "LRS", "RA_GRS", "RA_GZRS", "ZRS"}, false),
			},
			keyStorageAccountNamePrefix: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Azure storage account name prefix. The storage account name prefix cannot be longer than " +
					"14 characters and can only consist of numbers and lower case letters.",
				ValidateFunc: validation.All(validation.StringLenBetween(1, 14),
					validation.StringMatch(regexp.MustCompile("^[a-z0-9]*$"), "storage account name may only contain numbers and lowercase letters")),
			},
			keyStorageAccountRegion: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "Azure region to store the snapshots in. If not specified, the snapshots will be stored " +
					"in the same region as the workload.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyStorageAccountTags: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Azure storage account tags. Each tag will be added to the storage account created by RSC.",
			},
			keyStorageTier: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "COOL",
				Description:  "Azure storage tier. Possible values are `COOL` and `HOT`. Default value is `COOL`.",
				ValidateFunc: validation.StringInSlice([]string{"COOL", "HOT"}, false),
			},
		},
	}
}

func azureCreateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	customerKeys := fromCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set))
	name := d.Get(keyName).(string)
	redundancy := d.Get(keyRedundancy).(string)
	storageAccountName := d.Get(keyStorageAccountNamePrefix).(string)
	storageAccountRegion := d.Get(keyStorageAccountRegion).(string)
	storageAccountTags, err := fromBucketTags(d.Get(keyStorageAccountTags).(map[string]any))
	if err != nil {
		return diag.FromErr(err)
	}
	storageTier := d.Get(keyStorageTier).(string)

	// Create the archival location.
	targetMappingID, err := azure.Wrap(client).CreateStorageSetting(
		ctx, azure.CloudAccountID(accountID), name, redundancy, storageTier, storageAccountName, storageAccountRegion, storageAccountTags, customerKeys)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMappingID.String())
	azureReadArchivalLocation(ctx, d, m)
	return nil
}

func azureReadArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the archival location. If the archival location isn't found, we
	// remove it from the local state and return.
	targetMapping, err := azure.Wrap(client).TargetMappingByID(ctx, targetMappingID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyContainerName, targetMapping.ContainerName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCustomerManagedKey, toCustomerManagedKeys(targetMapping.CustomerKeys)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyLocationTemplate, targetMapping.LocTemplate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, targetMapping.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRedundancy, targetMapping.Redundancy); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountNamePrefix, targetMapping.StorageAccountName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountRegion, targetMapping.StorageAccountRegion); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageAccountTags, toStorageAccountTags(targetMapping.StorageAccountTags)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageTier, targetMapping.StorageTier); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureUpdateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	customerKeys := fromCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set))
	name := d.Get(keyName).(string)
	storageAccountTags, err := fromStorageAccountTags(d.Get(keyStorageAccountTags).(map[string]any))
	if err != nil {
		return diag.FromErr(err)
	}
	storageTier := d.Get(keyStorageTier).(string)

	// Update the archival location. Note, the API doesn't support updating
	// all arguments.
	err = azure.Wrap(client).UpdateStorageSetting(ctx, targetMappingID, name, storageTier, storageAccountTags, customerKeys)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureDeleteArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete the archival location.
	if err := azure.Wrap(client).DeleteTargetMapping(ctx, targetMappingID); err != nil {
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
				ValidateFunc: validation.StringIsNotWhiteSpace,
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

// fromCustomerManagedKeys converts from the customer managed keys field type
// to a customer key slice.
func fromCustomerManagedKeys(customerManagedKeys *schema.Set) []azure.CustomerKey {
	var customerKeys []azure.CustomerKey
	for _, key := range customerManagedKeys.List() {
		key := key.(map[string]any)
		customerKeys = append(customerKeys, azure.CustomerKey{
			Name:      key[keyName].(string),
			Region:    key[keyRegion].(string),
			VaultName: key[keyVaultName].(string),
		})
	}

	return customerKeys
}

// toStorageAccountTags converts to the customer managed keys field type from
// a customer key slice.
func toCustomerManagedKeys(customerKeys []azure.CustomerKey) *schema.Set {
	customerManagedKeys := &schema.Set{F: schema.HashResource(customerKeyResource())}
	for _, key := range customerKeys {
		customerManagedKeys.Add(map[string]any{
			keyName:      key.Name,
			keyRegion:    key.Region,
			keyVaultName: key.VaultName,
		})
	}

	return customerManagedKeys
}

// fromStorageAccountTags converts from the storage account tags field type to
// a standard string-to-string map.
func fromStorageAccountTags(storageAccountTags map[string]any) (map[string]string, error) {
	tags := make(map[string]string, len(storageAccountTags))
	for key, value := range storageAccountTags {
		value, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("storage account tag value for key %q is not a string", key)
		}
		tags[key] = value
	}

	return tags, nil
}

// toStorageAccountTags converts to the storage account tags field type from a
// standard string-to-string map.
func toStorageAccountTags(tags map[string]string) map[string]any {
	storageAccountTags := make(map[string]any, len(tags))
	for key, value := range tags {
		storageAccountTags[key] = value
	}

	return storageAccountTags
}
