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
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/exocompute"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlgcp "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/gcp"
)

const resourceGCPExocomputeDescription = `
The ´polaris_gcp_exocompute´ resource creates an RSC Exocompute configuration
for GCP workloads. This resource should only be used with customer managed
networking. Customer managed networking is used when the ´EXOCOMPUTE´ feature
of the GCP project was onboarded without the ´AUTOMATED_NETWORKING_SETUP´
permission group. If the GCP project was onboarded with the
´AUTOMATED_NETWORKING_SETUP´ permission group, RSC will automatically create
and manage the networking resources for Exocompute.
`

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceGcpExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateExocompute,
		ReadContext:   gcpReadExocompute,
		UpdateContext: gcpUpdateExocompute,
		DeleteContext: gcpDeleteExocompute,

		Description: description(resourceGCPExocomputeDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC Cloud Account ID (UUID).",
			},
			keyCloudAccountID: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "RSC cloud account ID. This is the ID of the `polaris_gcp_project` resource for " +
					"which the Exocompute service runs. Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyRegionalConfig: {
				Type:        schema.TypeSet,
				Elem:        gcpRegionalConfigResource(),
				Required:    true,
				Description: "Regional configuration for the Exocompute service.",
			},
			keyTriggerHealthCheck: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Trigger a health check for the Exocompute configuration.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func gcpRegionalConfigResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyRegion: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "GCP region to run the exocompute service in. Should be specified in the standard GCP " +
					"style, e.g. `us-east1`.",
				ValidateFunc: validation.StringInSlice(gqlgcp.AllRegionNames(), false),
			},
			keySubnetName: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Name of the GCP subnet to run the exocompute service in.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyVPCName: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Name of the GCP VPC to run the exocompute service in.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func gcpCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	healthCheck := d.Get(keyTriggerHealthCheck).(bool)
	err = exocompute.Wrap(client).UpdateGCPConfiguration(ctx, cloudAccountID, fromRegionalConfig(d), healthCheck)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cloudAccountID.String())
	gcpReadExocompute(ctx, d, m)
	return nil
}

func gcpReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	exoConfigs, err := exocompute.Wrap(client).GCPConfigurationsByCloudAccountID(ctx, cloudAccountID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRegionalConfig, toRegionalConfig(exoConfigs)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	healthCheck := d.Get(keyTriggerHealthCheck).(bool)
	err = exocompute.Wrap(client).UpdateGCPConfiguration(ctx, cloudAccountID, fromRegionalConfig(d), healthCheck)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = exocompute.Wrap(client).RemoveGCPConfiguration(ctx, cloudAccountID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func fromRegionalConfig(d *schema.ResourceData) []exocompute.RegionalConfig {
	var configs []exocompute.RegionalConfig
	for _, config := range d.Get(keyRegionalConfig).(*schema.Set).List() {
		config := config.(map[string]any)
		configs = append(configs, exocompute.RegionalConfig{
			Region:         gqlgcp.RegionFromName(config[keyRegion].(string)),
			SubnetName:     config[keySubnetName].(string),
			VPCNetworkName: config[keyVPCName].(string),
		})
	}

	return configs
}

func toRegionalConfig(exoConfigs []exocompute.GCPConfiguration) *schema.Set {
	configs := &schema.Set{F: schema.HashResource(gcpRegionalConfigResource())}
	for _, exoConfig := range exoConfigs {
		configs.Add(map[string]any{
			keyRegion:     exoConfig.Config.Region.Name(),
			keySubnetName: exoConfig.Config.SubnetName,
			keyVPCName:    exoConfig.Config.VPCNetworkName,
		})
	}

	return configs
}
