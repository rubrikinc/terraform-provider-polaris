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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceDeployment defines the schema for the RSC deployment data source.
func dataSourceDeployment() *schema.Resource {
	return &schema.Resource{
		ReadContext: deploymentRead,

		Schema: map[string]*schema.Schema{
			"ip_addresses": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Deployment IP addresses.",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deployment version.",
			},
		},
	}
}

// deploymentRead run the Read operation for the deployment data source. Returns
// details about the RSC deployment.
func deploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] deploymentRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Request deployment details.
	ipAddresses, err := core.Wrap(client.GQL).DeploymentIPAddresses(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	version, err := client.GQL.DeploymentVersion(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	ipAddressesAttr := &schema.Set{F: schema.HashString}
	for _, ipAddress := range ipAddresses {
		ipAddressesAttr.Add(ipAddress)
	}
	if err := d.Set("ip_addresses", ipAddressesAttr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("version", version); err != nil {
		return diag.FromErr(err)
	}

	// Generate an ID for the data source.
	hash := sha256.New()
	for _, ipAddress := range ipAddresses {
		hash.Write([]byte(ipAddress))
	}
	hash.Write([]byte(version))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
