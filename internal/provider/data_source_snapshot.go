// Copyright 2026 Rubrik, Inc.
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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const dataSourceSnapshotDescription = `
The ´polaris_snapshot´ data source looks up the closest snapshot to a given
point in time for a workload object.

Exactly one of ´before_time´ or ´after_time´ must be specified:

- ´before_time´ — returns the latest snapshot with date <= the given timestamp.
- ´after_time´ — returns the earliest snapshot with date >= the given timestamp.

Only clean snapshots are returned by default. Set ´exclude_quarantined´ or
´exclude_anomalous´ to ´false´ to include those snapshot types.
`

func dataSourceSnapshot() *schema.Resource {
	return &schema.Resource{
		ReadContext: snapshotRead,

		Description: description(dataSourceSnapshotDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Snapshot ID (UUID).",
			},
			keyWorkloadID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Workload ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyBeforeTime: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyBeforeTime, keyAfterTime},
				Description: "Return the latest snapshot with date <= this RFC 3339 timestamp " +
					"(e.g. 2025-01-01T00:00:00Z). Mutually exclusive with ´after_time´.",
				ValidateFunc: validation.IsRFC3339Time,
			},
			keyAfterTime: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyBeforeTime, keyAfterTime},
				Description: "Return the earliest snapshot with date >= this RFC 3339 timestamp " +
					"(e.g. 2025-01-01T00:00:00Z). Mutually exclusive with ´before_time´.",
				ValidateFunc: validation.IsRFC3339Time,
			},
			keyExcludeQuarantined: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Exclude quarantined snapshots.",
			},
			keyExcludeAnomalous: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Exclude anomalous snapshots.",
			},
			keyDate: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Snapshot timestamp.",
			},
		},
	}
}

func snapshotRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "snapshotRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	params := hierarchy.ClosestSnapshotParams{
		WorkloadID:         d.Get(keyWorkloadID).(string),
		ExcludeQuarantined: d.Get(keyExcludeQuarantined).(bool),
		ExcludeAnomalous:   d.Get(keyExcludeAnomalous).(bool),
	}

	if v := d.Get(keyAfterTime).(string); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return diag.FromErr(err)
		}
		params.AfterTime = &t
	} else {
		t, err := time.Parse(time.RFC3339, d.Get(keyBeforeTime).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		params.BeforeTime = &t
	}

	result, err := hierarchy.Wrap(client.GQL).ClosestSnapshot(ctx, params)
	if err != nil {
		return diag.FromErr(err)
	}

	if result.Snapshot == nil {
		return diag.Errorf("no snapshot found for workload %s", params.WorkloadID)
	}

	d.SetId(result.Snapshot.ID)
	if err := d.Set(keyDate, result.Snapshot.Date.Format(time.RFC3339)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
