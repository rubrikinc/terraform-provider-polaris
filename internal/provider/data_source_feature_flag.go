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
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceFeatureFlagDescription = `
The ´polaris_feature_flag´ data source is used to check if a feature flag is enabled for
the RSC account.
`

func dataSourceFeatureFlag() *schema.Resource {
	return &schema.Resource{
		ReadContext: featureFlagRead,

		Description: description(dataSourceFeatureFlagDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the feature flag name.",
			},
			keyName: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Feature flag name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			keyEnabled: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the feature flag is enabled for the RSC account.",
			},
		},
	}
}

func featureFlagRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "featureFlagRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get(keyName).(string)
	flag, err := core.Wrap(client.GQL).FeatureFlag(ctx, core.FeatureFlagName(name))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyEnabled, flag.Enabled); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	hash.Write([]byte(name))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
