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

func resourceAwsArchivalLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateArchivalLocation,
		ReadContext:   awsReadArchivalLocation,
		UpdateContext: awsUpdateArchivalLocation,
		DeleteContext: awsDeleteArchivalLocation,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"bucket_prefix": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "AWS bucket prefix. Note that `rubrik-` will always be prepended to the prefix.",
				ValidateFunc: validation.StringLenBetween(1, 19),
			},
			"bucket_tags": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "AWS bucket tags. Each tag will be added to the bucket created by RSC.",
			},
			"connection_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the archival location.",
			},
			"kms_master_key": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Default:      "aws/s3",
				Description:  "AWS KMS master key alias/ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"location_template": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Location template. If a region was specified, it will be `SPECIFIC_REGION`, otherwise `SOURCE_REGION`.",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"region": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "AWS region to store the snapshots in. If not specified, the snapshots will be stored in the same region as the workload.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"storage_class": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "STANDARD_IA",
				Description:  "AWS bucket storage class.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
	accountID, err := uuid.Parse(d.Get("account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the resource string arguments.
	bucketPrefix := d.Get("bucket_prefix").(string)
	kmsMasterKey := d.Get("kms_master_key").(string)
	name := d.Get("name").(string)
	region := d.Get("region").(string)
	storageClass := d.Get("storage_class").(string)

	// Lookup the resource bucket tags argument.
	bucketTags, err := fromBucketTags(d.Get("bucket_tags").(map[string]any))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create the AWS archival location.
	targetMappingID, err := aws.Wrap(client).CreateStorageSetting(
		ctx, aws.CloudAccountID(accountID), name, bucketPrefix, storageClass, region, kmsMasterKey, bucketTags)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set the resource ID to the target mapping ID.
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

	// Lookup and parse the target mapping ID from the resource ID.
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

	// Set the resource string arguments.
	if err := d.Set("bucket_prefix", strings.TrimPrefix(targetMapping.BucketPrefix, "rubrik-")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("connection_status", targetMapping.ConnectionStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("kms_master_key", targetMapping.KMSMasterKey); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("location_template", targetMapping.LocTemplate); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", targetMapping.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", targetMapping.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("storage_class", targetMapping.StorageClass); err != nil {
		return diag.FromErr(err)
	}

	// Set the resource bucket tags argument.
	if err := d.Set("bucket_tags", toBucketTags(targetMapping.BucketTags)); err != nil {
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

	// Lookup and parse the target mapping ID from the resource ID.
	targetMappingID, err := uuid.Parse(d.Id())
	if err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	// Lookup the resource string arguments.
	kmsMasterKey := d.Get("kms_master_key").(string)
	name := d.Get("name").(string)
	storageClass := d.Get("storage_class").(string)

	// Update the AWS archival location. Note, the API doesn't support updating
	// all arguments.
	err = aws.Wrap(client).UpdateStorageSetting(ctx, targetMappingID, name, storageClass, kmsMasterKey)
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

	// Lookup and parse the target mapping ID from the resource ID.
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

// fromBucketTags converts from the bucket tags argument to a standard string to
// string map.
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

// toBucketTags converts to the bucket tags argument from a standard string to
// string map.
func toBucketTags(tags map[string]string) map[string]any {
	bucketTags := make(map[string]any, len(tags))
	for key, value := range tags {
		bucketTags[key] = value
	}

	return bucketTags
}
