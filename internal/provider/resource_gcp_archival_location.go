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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
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

-> **Note:** The GCP bucket holding the archived data is not created until the
   first protected object is archived.
`

func resourceGCPArchivalLocation() *schema.Resource {
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
			keyBucketPrefix: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "GCP bucket prefix. The prefix cannot be longer than 19 characters. Note that `rubrik-` " +
					"will always be prepended to the prefix. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringLenBetween(1, 19),
			},
			keyBucketLabels: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "GCP bucket labels. Each label will be added to the bucket created by RSC.",
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
				Description: "Customer managed storage encryption. Specify the regions and their respective " +
					"encryption details. For other regions, data will be encrypted using platform managed keys.",
			},
			keyLocationTemplate: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Location template. If a region was specified, it will be `SPECIFIC_REGION`, otherwise " +
					"`SOURCE_REGION`.",
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
				Description: "AWS region to store the snapshots in. If not specified, the snapshots will be " +
					"stored in the same region as the workload. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(aws.AllRegionNames(), false),
			},
			keyStorageClass: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD_IA",
				Description: "AWS bucket storage class. Possible values are `STANDARD`, `STANDARD_IA`, `ONEZONE_IA`, " +
					"`GLACIER_INSTANT_RETRIEVAL`, `GLACIER_DEEP_ARCHIVE` and `GLACIER_FLEXIBLE_RETRIEVAL`. Default " +
					"value is `STANDARD_IA`.",
				ValidateFunc: validation.StringInSlice([]string{
					"STANDARD", "STANDARD_IA", "ONEZONE_IA", "GLACIER_INSTANT_RETRIEVAL", "GLACIER_DEEP_ARCHIVE",
					"GLACIER_FLEXIBLE_RETRIEVAL",
				}, false),
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func gcpCreateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create the AWS cloud native archival location.
	targetMappingID, err := archival.Wrap(client).CreateAWSStorageSetting(ctx, gqlarchival.CreateAWSStorageSettingParams{
		CloudAccountID: cloudAccountID,
		Name:           d.Get(keyName).(string),
		BucketPrefix:   d.Get(keyBucketPrefix).(string),
		StorageClass:   d.Get(keyStorageClass).(string),
		Region:         aws.RegionFromName(d.Get(keyRegion).(string)).ToRegionEnum(),
		KmsMasterKey:   d.Get(keyKMSMasterKey).(string),
		BucketTags:     toAWSBucketTags(d.Get(keyBucketTags).(map[string]any)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMappingID.String())
	awsReadArchivalLocation(ctx, d, m)
	return nil
}

func gcpReadArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the AWS cloud native archival location. If the archival location
	// isn't found we remove it from the local state and return.
	targetMapping, err := archival.Wrap(client).AWSTargetMappingByID(ctx, targetMappingID)
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
	if err := d.Set(keyAccountID, targetTemplate.CloudAccount.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyKMSMasterKey, targetTemplate.KMSMasterKey); err != nil {
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
	if err := d.Set(keyStorageClass, targetTemplate.StorageClass); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketTags, fromAWSBucketTags(targetTemplate.BucketTags)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	bucketTags := toAWSBucketTags(d.Get(keyBucketTags).(map[string]any))

	// Update the AWS cloud native archival location. Note, the API doesn't
	// support updating all arguments.
	err = archival.Wrap(client).UpdateAWSStorageSetting(ctx, targetMappingID, gqlarchival.UpdateAWSStorageSettingParams{
		Name:                d.Get(keyName).(string),
		StorageClass:        d.Get(keyStorageClass).(string),
		KmsMasterKey:        d.Get(keyKMSMasterKey).(string),
		DeleteAllBucketTags: bucketTags == nil,
		BucketTags:          bucketTags,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpDeleteArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete the AWS cloud native archival location.
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
				Type:     schema.TypeString,
				Required: true,
				Description: "The region in which the key will be used. Regions without customer managed keys will " +
					"use platform managed keys.",
				ValidateFunc: validation.StringInSlice(azure.AllRegionNames(), false),
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
// to a slice of Azure customer keys.
func toGCPCustomerManagedKeys(keys *schema.Set) []gqlarchival.GCPCustomerKey {
	var customerKeys []gqlarchival.GCPCustomerKey
	for _, key := range keys.List() {
		key := key.(map[string]any)
		customerKeys = append(customerKeys, gqlarchival.GCPCustomerKey{
			KeyName:     key[keyName].(string),
			KeyRingName: key[keyRingName].(string),
			Region:      gcp.RegionFromName(key[keyRegion].(string)).ToRegionEnum(),
		})
	}

	return customerKeys
}

// fromGCPCustomerManagedKeys converts to the customer managed keys field type from
// a slice of GCP customer keys.
func fromGCPCustomerManagedKeys(customerKeys []gqlarchival.GCPCustomerKey) *schema.Set {
	keys := &schema.Set{F: schema.HashResource(gcpCustomerKeyResource())}
	for _, key := range customerKeys {
		keys.Add(map[string]any{
			keyName:     key.KeyName,
			keyRingName: key.KeyRingName,
			keyRegion:   key.Region,
		})
	}

	return keys
}

// toGCPBucketLabels converts from the bucket labels argument to archival GCP
// tags. If the bucket labels argument is empty, nil is returned.
func toGCPBucketLabels(tags map[string]any) *gqlarchival.AWSTags {
	tagList := make([]core.Tag, 0, len(tags))
	for key, value := range tags {
		tagList = append(tagList, core.Tag{Key: key, Value: value.(string)})
	}
	if len(tagList) > 0 {
		return &gqlarchival.AWSTags{TagList: tagList}
	}

	return nil
}

// fromGCPBucketLabels converts to the bucket labels argument from a slice
// of GCP tags.
func fromGCPBucketLabels(tags []core.Tag) map[string]any {
	bucketTags := make(map[string]any, len(tags))
	for _, tag := range tags {
		bucketTags[tag.Key] = tag.Value
	}

	return bucketTags
}
