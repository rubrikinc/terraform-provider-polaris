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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
)

const dataSourceNCDArchivalLocationDescription = `
The ´polaris_ncd_archival_location´ data source is used to access information about
a NAS Cloud Direct (NCD) archival location for a specific cluster. An archival location
is looked up using the cluster ID and archival location name.

Use this data source to get the archival location ID for use with ´ncd_config´ backup
location fields in an SLA domain resource.
`

func dataSourceNCDArchivalLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: ncdArchivalLocationRead,

		Description: description(dataSourceNCDArchivalLocationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Archival location ID (UUID).",
			},
			keyClusterID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Cluster ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the archival location.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the archival location.",
			},
			keyTargetType: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of the archival target.",
			},
			keyActive: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the archival location is active.",
			},
			keyClusterName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the cluster.",
			},
			keyClusterStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the cluster.",
			},
			keyClusterVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Version of the cluster.",
			},
		},
	}
}

func ncdArchivalLocationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "ncdArchivalLocationRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := uuid.Parse(d.Get(keyClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	name := d.Get(keyName).(string)

	locations, err := archival.Wrap(client).NCDArchivalLocationByName(ctx, clusterID, name)
	if err != nil {
		return diag.FromErr(err)
	}

	// Filter for exact name match since the API uses "contains" matching
	inx := -1
	for i, loc := range locations {
		if loc.Name == name {
			if inx != -1 {
				return diag.Errorf("multiple NCD archival locations found for cluster %s with name %s", clusterID, name)
			}
			inx = i
		}
	}
	if inx == -1 {
		return diag.Errorf("no NCD archival location found for cluster %s with name %s", clusterID, name)
	}
	location := locations[inx]

	if err := d.Set(keyName, location.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStatus, location.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTargetType, location.TargetType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyActive, location.IsActive); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyClusterName, location.Cluster.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyClusterStatus, location.Cluster.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyClusterVersion, location.Cluster.Version); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(location.ID.String())
	return nil
}
