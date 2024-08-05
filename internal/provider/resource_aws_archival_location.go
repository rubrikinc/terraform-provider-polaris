// Copyright 2023 Rubrik, Inc.
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
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

const (
	implicitPrefix = "rubrik-"
)

const resourceAWSArchivalLocationDescription = `
The ´polaris_aws_archival_location´ resource creates an RSC archival location for
cloud-native workloads. This resource requires that the AWS account has been
onboarded with the ´CLOUD_NATIVE_ARCHIVAL´ feature.

When creating an archival location, the region where the snapshots are stored needs
to be specified:
  * ´SOURCE_REGION´ - Store snapshots in the same region to minimize data transfer
    charges. This is the default behaviour when the ´region´ field is not specified.
  * ´SPECIFIC_REGION´ - Storing snapshots in another region can increase total data
    transfer charges. The ´region´ field specifies the region.

-> **Note:** The AWS bucket holding the archived data is not created until the first
   protected object is archived.
`

func resourceAwsArchivalLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateArchivalLocation,
		ReadContext:   awsReadArchivalLocation,
		UpdateContext: awsUpdateArchivalLocation,
		DeleteContext: awsDeleteArchivalLocation,

		Description: description(resourceAWSArchivalLocationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud native archival location ID (UUID).",
			},
			keyAccountID: {
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
				Description: "AWS bucket prefix. The prefix cannot be longer than 19 characters. Note that `rubrik-` " +
					"will always be prepended to the prefix. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringLenBetween(1, 19),
			},
			keyBucketTags: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "AWS bucket tags. Each tag will be added to the bucket created by RSC.",
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the cloud native archival location.",
			},
			keyKMSMasterKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Default:      "aws/s3",
				Description:  "AWS KMS master key alias/ID. Default value is `aws/s3`.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
	}
}

func awsCreateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup and parse the cloud account ID argument. Note, if this argument
	// changes the resource will be recreated.
	accountID, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	bucketPrefix := d.Get(keyBucketPrefix).(string)
	kmsMasterKey := d.Get(keyKMSMasterKey).(string)
	name := d.Get(keyName).(string)
	region := d.Get(keyRegion).(string)
	storageClass := d.Get(keyStorageClass).(string)

	bucketTags, err := fromBucketTags(d.Get(keyBucketTags).(map[string]any))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create the AWS archival location.
	targetMappingID, err := aws.Wrap(client).CreateStorageSetting(
		ctx, aws.CloudAccountID(accountID), name, bucketPrefix, storageClass, region, kmsMasterKey, bucketTags)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMappingID.String())
	awsReadArchivalLocation(ctx, d, m)
	return nil
}

func awsReadArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the AWS archival location. If the archival location isn't found we
	// remove it from the local state and return.
	targetMapping, err := aws.Wrap(client).TargetMappingByID(ctx, targetMappingID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyBucketPrefix, strings.TrimPrefix(targetMapping.BucketPrefix, implicitPrefix)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyKMSMasterKey, targetMapping.KMSMasterKey); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyLocationTemplate, targetMapping.LocTemplate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, targetMapping.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRegion, targetMapping.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageClass, targetMapping.StorageClass); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketTags, toBucketTags(targetMapping.BucketTags)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	kmsMasterKey := d.Get(keyKMSMasterKey).(string)
	name := d.Get(keyName).(string)
	storageClass := d.Get(keyStorageClass).(string)
	bucketTags, err := fromBucketTags(d.Get(keyBucketTags).(map[string]any))
	if err != nil {
		return diag.FromErr(err)
	}

	// Update the AWS archival location. Note, the API doesn't support updating
	// all arguments.
	err = aws.Wrap(client).UpdateStorageSetting(ctx, targetMappingID, name, storageClass, kmsMasterKey, bucketTags)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsDeleteArchivalLocation(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteArchivalLocation")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete the AWS archival location.
	if err := aws.Wrap(client).DeleteTargetMapping(ctx, targetMappingID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// fromBucketTags converts from the bucket tags argument to a standard
// string-to-string map.
func fromBucketTags(bucketTags map[string]any) (map[string]string, error) {
	tags := make(map[string]string, len(bucketTags))
	for key, value := range bucketTags {
		value, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("bucket tag value for key %q is not a string", key)
		}
		tags[key] = value
	}

	return tags, nil
}

// toBucketTags converts to the bucket tags argument from a standard
// string-to-string map.
func toBucketTags(tags map[string]string) map[string]any {
	bucketTags := make(map[string]any, len(tags))
	for key, value := range tags {
		bucketTags[key] = value
	}

	return bucketTags
}
