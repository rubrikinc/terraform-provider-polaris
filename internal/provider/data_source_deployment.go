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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceDeploymentDescription = `
The ´polaris_deployment´ data source is used to access information about the RSC
deployment.
`

func dataSourceDeployment() *schema.Resource {
	return &schema.Resource{
		ReadContext: deploymentRead,

		Description: description(dataSourceDeploymentDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the fields in order.",
			},
			keyIPAddresses: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Deployment IP addresses.",
			},
			keyVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deployment version.",
			},
		},
	}
}

func deploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "deploymentRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	ipAddresses, err := core.Wrap(client.GQL).DeploymentIPAddresses(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	version, err := client.GQL.DeploymentVersion(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	ipAddressesAttr := &schema.Set{F: schema.HashString}
	for _, ipAddress := range ipAddresses {
		ipAddressesAttr.Add(ipAddress)
	}
	if err := d.Set(keyIPAddresses, ipAddressesAttr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyVersion, version); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, ipAddress := range ipAddresses {
		hash.Write([]byte(ipAddress))
	}
	hash.Write([]byte(version))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
