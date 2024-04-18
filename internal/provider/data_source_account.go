// Copyright 2024 Rubrik, Inc.
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
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceAccount defines the schema for the RSC account data source.
func dataSourceAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: accountRead,

		Description: "The `polaris_account` data source is used to access information about the RSC account.\n" +
			"\n" +
			"-> **Note:** The `fqdn` and `name` fields are read from the local RSC credentials and not from RSC.",
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the fields in order.",
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Features enabled for the RSC account.",
			},
			keyFQDN: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified domain name of the RSC account.",
			},
			keyName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC account name.",
			},
		},
	}
}

// accountRead run the Read operation for the account data source. Returns
// details about the RSC account.
func accountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] accountRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Request deployment details.
	accountFeatures, err := core.Wrap(client.GQL).EnabledFeaturesForAccount(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	accountFQDN := strings.ToLower(client.Account.AccountFQDN())
	accountName := strings.ToLower(client.Account.AccountName())

	// Set attributes.
	accountFeaturesAttr := &schema.Set{F: schema.HashString}
	for _, accountFeature := range accountFeatures {
		accountFeaturesAttr.Add(accountFeature.Name)
	}
	if err := d.Set(keyFeatures, accountFeaturesAttr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyFQDN, accountFQDN); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, accountName); err != nil {
		return diag.FromErr(err)

	}

	// Generate an ID for the data source.
	hash := sha256.New()
	for _, accountFeature := range accountFeatures {
		hash.Write([]byte(accountFeature.Name))
	}
	hash.Write([]byte(accountFQDN))
	hash.Write([]byte(accountName))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
