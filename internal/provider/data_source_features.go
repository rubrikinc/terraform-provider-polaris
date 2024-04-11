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
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"slices"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceFeatures defines the schema for the RSC features data source.
func dataSourceFeatures() *schema.Resource {
	return &schema.Resource{
		ReadContext: featuresRead,

		Schema: map[string]*schema.Schema{
			"features": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Enabled features.",
			},
		},
	}
}

// featuresRead run the Read operation for the RSC features data source. Returns
// all RSC features enabled for the current RSC account.
func featuresRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] featuresRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Request features.
	features, err := core.Wrap(client.GQL).EnabledFeaturesForAccount(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	slices.SortFunc(features, func(lhs, rhs core.Feature) int {
		return cmp.Compare(lhs.Name, rhs.Name)
	})

	// Set attributes.
	var featuresAttr []string
	for _, feature := range features {
		featuresAttr = append(featuresAttr, feature.Name)
	}
	if err := d.Set("features", featuresAttr); err != nil {
		return diag.FromErr(err)
	}

	// Generate an ID for the data source.
	hash := sha256.New()
	for _, feature := range features {
		hash.Write([]byte(feature.Name))
	}
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
