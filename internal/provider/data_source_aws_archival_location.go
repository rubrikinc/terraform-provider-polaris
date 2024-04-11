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
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
)

func dataSourceAwsArchivalLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsArchivalLocationRead,

		Schema: map[string]*schema.Schema{
			"bucket_prefix": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS bucket prefix.",
			},
			"bucket_tags": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "AWS bucket tags.",
			},
			"connection_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the archival location.",
			},
			"archival_location_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"archival_location_id", "name"},
				Description:  "ID of the archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"kms_master_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "AWS KMS master key alias/ID.",
			},
			"location_template": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Location template. If a region was specified, it will be `SPECIFIC_REGION`, otherwise `SOURCE_REGION`.",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"archival_location_id", "name"},
				Description:  "Name of the archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS region to store the snapshots in. If not specified, the snapshots will be stored in the same region as the workload.",
			},
			"storage_class": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS bucket storage class.",
			},
		},
	}
}

func awsArchivalLocationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] awsArchivalLocationRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var targetMapping aws.TargetMapping
	if targetMappingID, ok := d.GetOk("archival_location_id"); ok {
		id, err := uuid.Parse(targetMappingID.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		// Read the AWS archival location using the target mapping ID.
		targetMapping, err = aws.Wrap(client).TargetMappingByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		targetMappingName := d.Get("name").(string)

		// Read the AWS archival location using the target mapping name.
		targetMapping, err = aws.Wrap(client).TargetMappingByName(ctx, targetMappingName)
		if err != nil {
			return diag.FromErr(err)
		}
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

	// Set the resource ID to the target mapping ID.
	d.SetId(targetMapping.ID.String())

	return nil
}
