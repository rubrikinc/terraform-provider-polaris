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
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
)

const dataSourceGCPArchivalLocationDescription = `
The ´polaris_gcp_archival_location´ data source is used to access information
about a GCP archival location. An archival location is looked up using either
the ID or the name.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceGcpArchivalLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: gcpArchivalLocationRead,

		Description: description(dataSourceGCPArchivalLocationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyName},
				Description:  "Cloud native archival location ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyBucketLabels: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "GCP bucket labels.",
			},
			keyBucketPrefix: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "GCP bucket prefix. Note, `rubrik-` will always be prepended to the prefix.",
			},
			keyCloudAccountID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status of the archival location.",
			},
			keyCustomerManagedKey: {
				Type:     schema.TypeSet,
				Elem:     gcpCustomerKeyResource(),
				Computed: true,
				Description: "Customer managed storage encryption. For `SPECIFIC_REGION`, a customer managed key " +
					"for the specific region will be returned. For `SOURCE_REGION`, a customer managed key for each " +
					"source region will be returned, for other regions, data will be encrypted using platform " +
					"managed keys.",
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
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyName},
				Description:  "Name of the cloud native archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "GCP region to store the snapshots in (`SPECIFIC_REGION`). If not specified, the " +
					"snapshots will be stored in the same region as the workload (`SOURCE_REGION`).",
			},
			keyStorageClass: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "GCP bucket storage class. Possible values are `ARCHIVE`, `COLDLINE`, " +
					"`NEARLINE`, `STANDARD` and `DURABLE_REDUCED_AVAILABILITY`.",
			},
		},
	}
}

func gcpArchivalLocationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpArchivalLocationRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the archival location using either the ID or the name.
	var targetMapping gqlarchival.GCPTargetMapping
	if id := d.Get(keyID).(string); id != "" {
		id, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		targetMapping, err = archival.Wrap(client).GCPTargetMappingByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		targetMapping, err = archival.Wrap(client).GCPTargetMappingByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	targetTemplate := targetMapping.TargetTemplate
	if err := d.Set(keyBucketLabels, fromGCPBucketLabels(targetTemplate.BucketLabels)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketPrefix, strings.TrimPrefix(targetTemplate.BucketPrefix, "rubrik-")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudAccountID, targetTemplate.CloudAccount.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, targetMapping.ConnectionStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCustomerManagedKey, fromGCPCustomerManagedKeys(targetTemplate.CMKInfo)); err != nil {
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
	if err := d.Set(keyStorageClass, fromGCPStorageClass(targetTemplate.StorageClass)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(targetMapping.ID.String())
	return nil
}
