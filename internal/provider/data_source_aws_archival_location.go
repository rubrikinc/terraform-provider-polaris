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
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
)

const dataSourceAWSArchivalLocationDescription = `
The ´polaris_aws_archival_location´ data source is used to access information about an
AWS archival location. An archival location is looked up using either the ID or the name.
`

func dataSourceAwsArchivalLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsArchivalLocationRead,

		Description: description(dataSourceAWSArchivalLocationDescription),
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
			keyBucketPrefix: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS bucket prefix. Note, `rubrik-` will always be prepended to the prefix.",
			},
			keyBucketTags: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "AWS bucket tags.",
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the archival location.",
			},
			keyKMSMasterKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "AWS KMS master key alias/ID.",
			},
			keyLocationTemplate: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "RSC location template. If a region was specified, it will be `SPECIFIC_REGION`, " +
					"otherwise `SOURCE_REGION`.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID, keyArchivalLocationID, keyName},
				Description:  "Name of the cloud native archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "AWS region to store the snapshots in. If not specified, the snapshots will be stored " +
					"in the same region as the workload.",
			},
			keyStorageClass: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "AWS bucket storage class. Possible values are `STANDARD`, `STANDARD_IA`, `ONEZONE_IA`, " +
					"`GLACIER_INSTANT_RETRIEVAL`, `GLACIER_DEEP_ARCHIVE` and `GLACIER_FLEXIBLE_RETRIEVAL`. Default " +
					"value is `STANDARD_IA`.",
			},
		},
	}
}

func awsArchivalLocationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsArchivalLocationRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the archival location using either the ID or the name.
	var targetMapping gqlarchival.AWSTargetMapping
	targetMappingID := d.Get(keyID).(string)
	if targetMappingID == "" {
		targetMappingID = d.Get(keyArchivalLocationID).(string)
	}
	if targetMappingID != "" {
		id, err := uuid.Parse(targetMappingID)
		if err != nil {
			return diag.FromErr(err)
		}
		targetMapping, err = archival.Wrap(client).AWSTargetMappingByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		targetMapping, err = archival.Wrap(client).AWSTargetMappingByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	targetTemplate := targetMapping.TargetTemplate
	if err := d.Set(keyArchivalLocationID, targetMapping.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketPrefix, strings.TrimPrefix(targetTemplate.BucketPrefix, "rubrik-")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus); err != nil {
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
	if err := d.Set(keyRegion, targetTemplate.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageClass, targetTemplate.StorageClass); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketTags, fromAWSBucketTags(targetTemplate.BucketTags)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMapping.ID.String())
	return nil
}
