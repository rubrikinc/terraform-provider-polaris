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
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceWorkloadsDescription = `
The ´polaris_workloads´ data source is used to access information about the
valid workload hierarchy types (snappable types) that can be used in the RSC account.
`

func dataSourceWorkloads() *schema.Resource {
	return &schema.Resource{
		ReadContext: workloadsRead,

		Description: description(dataSourceWorkloadsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the workloads.",
			},
			keyWorkloads: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Valid workload hierarchy types (snappable types) that can be used in the RSC account.",
			},
		},
	}
}

func workloadsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "workloadsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	coreClient := core.Wrap(client.GQL)

	workloads, err := coreClient.ValuesByEnum(ctx, "WorkloadLevelHierarchy")
	if err != nil {
		return diag.FromErr(err)
	}

	slices.Sort(workloads)

	var workloadAttr []string
	for _, workload := range workloads {
		workloadAttr = append(workloadAttr, string(workload))
	}
	if err := d.Set(keyWorkloads, workloadAttr); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, workload := range workloads {
		hash.Write([]byte(workload))
	}
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
