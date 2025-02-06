// Copyright 2021 Rubrik, Inc.
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
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/exocompute"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
)

const resourceAzureExocomputeDescription = `
The ´polaris_azure_exocompute´ resource creates an RSC Exocompute configuration for
Azure workloads.

There are 2 types of Exocompute configurations:
 1. *Host* - When a host configuration is created, RSC will automatically deploy the
    necessary resources in the specified Azure region to run the Exocompute service.
    A host configuration can be used by both the host cloud account and application
    cloud accounts mapped to the host account.
 2. *Application* - An application configuration is created by mapping the application
    cloud account to a host cloud account. The application cloud account will leverage
    the Exocompute resources deployed for the host configuration.

Item 1 above requires that the Azure subscription has been onboarded with the
´exocompute´ feature.

Since there are 2 types of Exocompute configurations, there are 2 ways to create a
´polaris_azure_exocompute´ resource:
 1. Using the ´cloud_account_id´, ´region´, ´subnet´ and ´pod_overlay_network_cidr´
    fields. This creates a host configuration.
 2. Using the ´cloud_account_id´ and ´host_cloud_account_id´ fields. This creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   ´pod_overlay_network_cidr´ field, this is discouraged and should only be done for
   backwards compatibility reasons.

-> **Note:** Customer-managed Exocompute is sometimes referred to as Bring Your Own
   Kubernetes (BYOK). Using both host and application Exocompute configurations is
   sometimes referred to as shared Exocompute.
`

func resourceAzureExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateExocompute,
		ReadContext:   azureReadExocompute,
		DeleteContext: azureDeleteExocompute,

		Description: description(resourceAzureExocomputeDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Exocompute configuration ID (UUID).",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyCloudAccountID, keySubscriptionID},
				Description: "RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for " +
					"which the Exocompute service runs. Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyHostCloudAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{keyHostCloudAccountID, keyRegion},
				Description: "RSC cloud account ID of the shared exocompute host account. Changing this forces a new " +
					"resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyPodOverlayNetworkCIDR: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The CIDR range assigned to pods when launching Exocompute with the CNI overlay network " +
					"plugin mode. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "Azure region to run the exocompute service in. Should be specified in the standard " +
					"Azure style, e.g. `eastus`. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
			},
			keySubnet: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "Azure subnet ID of the cluster subnet corresponding to the Exocompute configuration. " +
					"This subnet will be used to allocate IP addresses to the nodes of the cluster. Changing this forces " +
					"a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keySubscriptionID: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for " +
					"which the Exocompute service runs. Changing this forces a new resource to be created. " +
					"**Deprecated:** use `cloud_account_id` instead.",
				Deprecated:   "use `cloud_account_id` instead.",
				ValidateFunc: validation.IsUUID,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureExocomputeV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureExocomputeStateUpgradeV0,
			Version: 0,
		}},
	}
}

func azureCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Get(keyCloudAccountID).(string)
	if id == "" {
		id = d.Get(keySubscriptionID).(string)
	}
	accountID, err := uuid.Parse(id)
	if err != nil {
		return diag.FromErr(err)
	}

	if hostCloudAccount, ok := d.GetOk(keyHostCloudAccountID); ok {
		hostCloudAccountID, err := uuid.Parse(hostCloudAccount.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		err = exocompute.Wrap(client).MapAzureCloudAccount(ctx, accountID, hostCloudAccountID)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(appCloudAccountPrefix + accountID.String())
	} else {
		var exoConfig exocompute.AzureConfigurationFunc
		region := gqlazure.RegionFromName(d.Get(keyRegion).(string))
		if podOverlayNetworkCIDR, ok := d.GetOk(keyPodOverlayNetworkCIDR); ok {
			exoConfig = exocompute.AzureManagedWithOverlayNetwork(region, d.Get(keySubnet).(string),
				podOverlayNetworkCIDR.(string))
		} else if subnet, ok := d.GetOk(keySubnet); ok {
			exoConfig = exocompute.AzureManaged(region, subnet.(string))
		} else {
			exoConfig = exocompute.AzureBYOKCluster(region)
		}
		exoConfigID, err := exocompute.Wrap(client).AddAzureConfiguration(ctx, accountID, exoConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(exoConfigID.String())
	}

	azureReadExocompute(ctx, d, m)
	return nil
}

func azureReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if id := d.Id(); strings.HasPrefix(id, appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}

		hostID, err := exocompute.Wrap(client).AzureHostCloudAccount(ctx, appID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyHostCloudAccountID, hostID.String()); err != nil {
			return diag.FromErr(err)
		}
	} else {
		exoConfigID, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}

		exoConfig, err := exocompute.Wrap(client).AzureConfigurationByID(ctx, exoConfigID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyRegion, exoConfig.Region.Name()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keySubnet, exoConfig.SubnetID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyPodOverlayNetworkCIDR, exoConfig.PodOverlayNetworkCIDR); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func azureDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if id := d.Id(); strings.HasPrefix(id, appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}
		err = exocompute.Wrap(client).UnmapAzureCloudAccount(ctx, appID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		exoConfigID, err := uuid.Parse(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		err = exocompute.Wrap(client).RemoveAzureConfiguration(ctx, exoConfigID)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}
