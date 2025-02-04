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
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/cdm"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceCDMRegistrationDescription = `
The ´polaris_cdm_registration´ resource registers a Rubrik cluster with the
Rubrik Security Cloud (RSC).

~> **Note:** The Terraform provider can only register clusters, it cannot
   un-register clusters or read the state of a cluster registration. Destroying
   the resource only removes it from the local state.
`

func resourceCDMRegistration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCDMRegistrationCreate,
		ReadContext:   resourceCDMRegistrationRead,
		DeleteContext: resourceCDMRegistrationDelete,

		Description: description(resourceCDMRegistrationDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster name.",
			},
			keyAdminPassword: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Sensitive:    true,
				Description:  "Password for the cluster admin account.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyClusterNodeIPAddress: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The IP address of the cluster node to connect to.",
				ValidateFunc: validation.IsIPAddress,
			},
			keyClusterName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Cluster name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegistrationMode: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster registration mode.",
			},
		},
	}
}

func resourceCDMRegistrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMRegistrationCreate")

	adminPassword := d.Get(keyAdminPassword).(string)
	nodeIP := d.Get(keyClusterNodeIPAddress).(string)
	cdmClient, err := cdm.NewClientFromCredentials(nodeIP, "admin", adminPassword, true)
	if err != nil {
		return diag.FromErr(err)
	}

	polarisClient, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterDetails, err := cdmClient.OfflineEntitle(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var regConfig []core.NodeRegistrationConfig
	for _, nodeDetails := range clusterDetails {
		regConfig = append(regConfig, nodeDetails.ToNodeRegistrationConfig())
	}
	authToken, _, err := core.Wrap(polarisClient.GQL).RegisterCluster(ctx, true, regConfig, true)
	if err != nil {
		log.Fatal(err)
	}

	mode, err := cdmClient.SetRegisteredMode(ctx, authToken)
	if err != nil {
		log.Fatal(err)
	}

	d.SetId(d.Get(keyClusterName).(string))
	if err := d.Set(keyRegistrationMode, mode); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCDMRegistrationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMRegistrationRead")

	return nil
}

// Once a Cluster has been registered it cannot be un-registered through the
// resource, delete simply removes the resource from the local state.
func resourceCDMRegistrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMRegistrationDelete")
	d.SetId("")
	return nil
}
