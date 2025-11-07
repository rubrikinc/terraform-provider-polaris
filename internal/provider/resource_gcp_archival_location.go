// Copyright 2025 Rubrik, Inc.
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
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/gcp"
)

const resourceGCPArchivalLocationDescription = `
The ´polaris_gcp_archival_location´ resource creates an RSC archival location
for cloud native workloads. This resource requires that the GCP project has been
onboarded with the ´CLOUD_NATIVE_ARCHIVAL´ feature.

When creating an archival location, the region where the snapshots are stored
needs to be specified:
  * ´SOURCE_REGION´ - Store snapshots in the same region to minimize data
    transfer charges. This is the default behaviour when the ´region´ field is
    not specified.
  * ´SPECIFIC_REGION´ - Storing snapshots in another region can increase total
    data transfer charges. The ´region´ field specifies the region.

Custom storage encryption is enabled by specifying one or more customer managed
key blocks. For ´SPECIFIC_REGION´, a customer managed key block for the specific
region must be specified. For ´SOURCE_REGION´, a customer managed key block for
each source region should be specified, source regions not having a customer
managed key block will have its data encrypted with platform managed keys.

-> **Note:** When using ´SOURCE_REGION´ the GCP bucket isn't created until the
   first protected object is archived.
`

func resourceGcpArchivalLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateArchivalLocation,
		ReadContext:   gcpReadArchivalLocation,
		UpdateContext: gcpUpdateArchivalLocation,
		DeleteContext: gcpDeleteArchivalLocation,

		Description: description(resourceGCPArchivalLocationDescription),
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
				ValidateFunc: validation.IsUUID,
			},
			keyBucketLabels: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "GCP bucket labels. Each label will be added to the GCP bucket created by RSC.",
			},
			keyBucketPrefix: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "GCP bucket prefix. The prefix cannot be longer than 19 characters. Note that `rubrik-` " +
					"will always be prepended to the prefix. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringLenBetween(1, 19),
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the cloud native archival location.",
			},
			keyCustomerManagedKey: {
				Type:     schema.TypeSet,
				Elem:     gcpCustomerKeyResource(),
				Optional: true,
				Description: "Customer managed storage encryption. For `SPECIFIC_REGION`, a customer managed key " +
					"block for the specific region must be specified. For `SOURCE_REGION`, a customer managed key " +
					"block for each source region should be specified, source regions not having a customer managed " +
					"key block will have its data encrypted with platform managed keys.",
			},
			keyLocationTemplate: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "RSC location template. If a region was specified, it will be `SPECIFIC_REGION`, " +
					"otherwise `SOURCE_REGION`.",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the cloud native archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "GCP region to store the snapshots in (`SPECIFIC_REGION`). If not specified, the " +
					"snapshots will be stored in the same region as the workload (`SOURCE_REGION`). Changing this " +
					"forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(gcp.AllRegionNames(), false),
			},
			keyStorageClass: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD",
				Description: "AWS bucket storage class. Possible values are `ARCHIVE`, `COLDLINE`, " +
					"`NEARLINE`, `STANDARD` and `DURABLE_REDUCED_AVAILABILITY`. Default value is `STANDARD`.",
				ValidateFunc: validation.StringInSlice([]string{
					"ARCHIVE", "COLDLINE", "NEARLINE", "STANDARD", "DURABLE_REDUCED_AVAILABILITY",
				}, false),
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func gcpCreateArchivalLocation(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var bucketLabels *core.Tags
	if labels := toGCPBucketLabels(d.Get(keyBucketLabels).(map[string]any)); len(labels.TagList) > 0 {
		bucketLabels = &labels
	}

	// Create the GCP cloud native archival location.
	targetMappingID, err := archival.Wrap(client).CreateGCPStorageSetting(ctx, gqlarchival.CreateGCPStorageSettingParams{
		CloudAccountID: cloudAccountID,
		Name:           d.Get(keyName).(string),
		BucketPrefix:   d.Get(keyBucketPrefix).(string),
		Region:         gcp.RegionFromName(d.Get(keyRegion).(string)).ToRegionEnum(),
		BucketLabels:   bucketLabels,
		CMKInfo:        toGCPCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set)),
		StorageClass:   toGCPStorageClass(d.Get(keyStorageClass).(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMappingID.String())
	gcpReadArchivalLocation(ctx, d, m)
	return nil
}

func gcpReadArchivalLocation(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the GCP cloud native archival location. If the archival location
	// isn't found we remove it from the local state and return.
	targetMapping, err := archival.Wrap(client).GCPTargetMappingByID(ctx, targetMappingID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	targetTemplate := targetMapping.TargetTemplate
	if err := d.Set(keyBucketPrefix, strings.TrimPrefix(targetTemplate.BucketPrefix, implicitPrefix)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudAccountID, targetTemplate.CloudAccount.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyLocationTemplate, targetTemplate.LocTemplate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, targetMapping.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRegion, targetTemplate.Region.Name()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketLabels, fromGCPBucketLabels(targetTemplate.BucketLabels)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCustomerManagedKey, fromGCPCustomerManagedKeys(targetTemplate.CMKInfo)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageClass, fromGCPStorageClass(targetTemplate.StorageClass)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateArchivalLocation(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Update the GCP cloud native archival location. Note, the API doesn't
	// support updating all fields.
	err = archival.Wrap(client).UpdateGCPStorageSetting(ctx, targetMappingID, gqlarchival.UpdateGCPStorageSettingParams{
		Name:         d.Get(keyName).(string),
		BucketLabels: toGCPBucketLabels(d.Get(keyBucketLabels).(map[string]any)),
		CMKInfo:      toGCPCustomerManagedKeys(d.Get(keyCustomerManagedKey).(*schema.Set)),
		StorageClass: toGCPStorageClass(d.Get(keyStorageClass).(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpDeleteArchivalLocation(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete the GCP cloud native archival location.
	if err := archival.Wrap(client).DeleteTargetMapping(ctx, targetMappingID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// gcpCustomerKeyResource returns the schema for a GCP customer managed key
// resource.
func gcpCustomerKeyResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Key name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The region in which the key will be used.",
				ValidateFunc: validation.StringInSlice(gcp.AllRegionNames(), false),
			},
			keyRingName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Key ring name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

// toGCPCustomerManagedKeys converts from the customer managed keys field type
// to a slice of GCP customer keys.
func toGCPCustomerManagedKeys(keys *schema.Set) []gqlarchival.GCPCustomerKey {
	var customerKeys []gqlarchival.GCPCustomerKey
	for _, key := range keys.List() {
		key := key.(map[string]any)
		customerKeys = append(customerKeys, gqlarchival.GCPCustomerKey{
			Name:     key[keyName].(string),
			RingName: key[keyRingName].(string),
			Region:   gcp.RegionFromName(key[keyRegion].(string)).ToRegionEnum(),
		})
	}

	return customerKeys
}

// fromGCPCustomerManagedKeys converts to the customer managed keys field type
// from a slice of GCP customer keys.
func fromGCPCustomerManagedKeys(customerKeys []gqlarchival.GCPCustomerKey) *schema.Set {
	keys := &schema.Set{F: schema.HashResource(gcpCustomerKeyResource())}
	for _, key := range customerKeys {
		keys.Add(map[string]any{
			keyName:     key.Name,
			keyRingName: key.RingName,
			keyRegion:   key.Region.Name(),
		})
	}

	return keys
}

// toGCPBucketLabels converts from the bucket labels field type to the GCP tags
// type.
func toGCPBucketLabels(labels map[string]any) core.Tags {
	tagList := make([]core.Tag, 0, len(labels))
	for key, value := range labels {
		tagList = append(tagList, core.Tag{Key: key, Value: value.(string)})
	}

	return core.Tags{TagList: tagList}
}

// fromGCPBucketLabels converts to the bucket labels field type from the GCP
// tags type.
func fromGCPBucketLabels(tags []core.Tag) map[string]any {
	bucketLabels := make(map[string]any, len(tags))
	for _, tag := range tags {
		bucketLabels[tag.Key] = tag.Value
	}

	return bucketLabels
}

func toGCPStorageClass(storageClass string) string {
	return storageClass + "_GCP"
}

func fromGCPStorageClass(storageClass string) string {
	return strings.TrimSuffix(storageClass, "_GCP")
}
