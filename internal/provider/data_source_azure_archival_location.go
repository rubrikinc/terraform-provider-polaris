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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
)

const dataSourceAzureArchivalLocationDescription = `
The ´polaris_azure_archival_location´ data source is used to access information about
an Azure archival location. An archival location is looked up using either the ID or
the name.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceAzureArchivalLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: azureArchivalLocationRead,

		Description: description(dataSourceAzureArchivalLocationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID, keyArchivalLocationID, keyName},
				Description:  "Cloud native archival location ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyArchivalLocationID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID, keyArchivalLocationID, keyName},
				Description:  "Cloud native archival location ID (UUID). **Deprecated:** use `id` instead.",
				ValidateFunc: validation.IsUUID,
				Deprecated:   "Use `id` instead.",
			},
			keyCloudAccountID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
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
				Computed: true,
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
				Optional:     true,
				ExactlyOneOf: []string{keyID, keyArchivalLocationID, keyName},
				Description:  "Cloud native archival location name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRedundancy: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Azure storage redundancy. Possible values are `GRS`, `GZRS`, `LRS`, `RA_GRS`, " +
					"`RA_GZRS` and `ZRS`. Default value is `LRS`.",
			},
			keyStorageAccountNamePrefix: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Azure storage account name prefix. The storage account name prefix cannot be longer " +
					"than 14 characters and can only consist of numbers and lower case letters.",
			},
			keyStorageAccountRegion: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Azure region to store the snapshots in. If not specified, the snapshots will be " +
					"stored in the same region as the workload.",
			},
			keyStorageAccountTags: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "Azure storage account tags. Each tag will be added to the storage account created by " +
					"RSC.",
			},
			keyStorageTier: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Azure storage tier. Possible values are `COOL` and `HOT`. Default value is `COOL`.",
			},
		},
	}
}

func azureArchivalLocationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureArchivalLocationRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the archival location using either the ID or the name.
	var targetMapping gqlarchival.AzureTargetMapping
	targetMappingID := d.Get(keyID).(string)
	if targetMappingID == "" {
		targetMappingID = d.Get(keyArchivalLocationID).(string)
	}
	if targetMappingID != "" {
		id, err := uuid.Parse(targetMappingID)
		if err != nil {
			return diag.FromErr(err)
		}
		targetMapping, err = archival.Wrap(client).AzureTargetMappingByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		targetMapping, err = archival.Wrap(client).AzureTargetMappingByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	targetTemplate := targetMapping.TargetTemplate
	cloudNativeCompanion := targetMapping.TargetTemplate.CloudNativeCompanion
	if err := d.Set(keyArchivalLocationID, targetMapping.ID.String()); err != nil {
		return diag.FromErr(err)
	}
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

	d.SetId(targetMapping.ID.String())
	return nil
}
