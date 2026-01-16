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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/cluster"
)

const dataSourceSLASourceClusterDescription = `
The ´polaris_sla_source_cluster´ data source is used to access information about
an SLA source cluster in RSC. A source cluster is looked up using the cluster name.
`

func dataSourceSLASourceCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: slaSourceClusterRead,

		Description: description(dataSourceSLASourceClusterDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster ID (UUID).",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Cluster name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster version.",
			},
		},
	}
}

func slaSourceClusterRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "slaSourceClusterRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterName := d.Get(keyName).(string)
	clusterData, err := cluster.Wrap(client).SLASourceClusterByName(ctx, clusterName)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyName, clusterData.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyVersion, clusterData.Version); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterData.ID.String())
	return nil
}
