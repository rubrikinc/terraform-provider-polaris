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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

const (
	appCloudAccountPrefix = "app-"
)

// resourceAzureExocompute defines the schema for the Azure exocompute resource.
func resourceAzureExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateExocompute,
		ReadContext:   azureReadExocompute,
		DeleteContext: azureDeleteExocompute,

		Description: "The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration.\n" +
			"\n" +
			"There are 2 types of Exocompute configurations:\n" +
			" 1. *Host* - When a host configuration is created, RSC will automatically deploy the necessary resources " +
			"    in the specified Azure region to run the Exocompute service. A host configuration can be used by both " +
			"    the host cloud account and application cloud accounts mapped to the host account.\n" +
			" 2. *Application* - An application configuration is created by mapping the application cloud account to a " +
			"    host cloud account. The application cloud account will leverage the Exocompute resources deployed for " +
			"    the host configuration.\n" +
			"\n" +
			"Since there are 2 types of Exocompute configurations, there are 2 ways to create a `polaris_azure_exocompute` " +
			"resource:\n" +
			" 1. Using the `cloud_account_id`, `region`, `subnet` and `pod_overlay_network_cidr` fields. This creates a " +
			"    host configuration.\n" +
			" 2. Using the `cloud_account_id` and `host_cloud_account_id` fields. This creates an application " +
			"    configuration.\n" +
			"\n" +
			"~> **Note:** A host configuration can be created without specifying the `pod_overlay_network_cidr` field, " +
			"   this is discouraged and should only be done for backwards compatibility reasons.\n" +
			"\n" +
			"-> **Note:** Using both host and application Exocompute configurations is sometimes referred to as shared " +
			"   Exocompute.",
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Exocompute configuration ID.",
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
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
				ValidateFunc: validation.StringIsNotWhiteSpace,
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

// azureCreateExocompute run the Create operation for the Azure exocompute
// resource. This enables the exocompute feature and adds an exocompute config
// to the RSC cloud account.
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
		err = azure.Wrap(client).MapExocompute(ctx, azure.CloudAccountID(hostCloudAccountID), azure.CloudAccountID(accountID))
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(appCloudAccountPrefix + accountID.String())
	} else {
		var exoConfig azure.ExoConfigFunc
		if podOverlayNetworkCIDR, ok := d.GetOk(keyPodOverlayNetworkCIDR); ok {
			exoConfig = azure.ManagedWithOverlayNetwork(d.Get(keyRegion).(string), d.Get(keySubnet).(string),
				podOverlayNetworkCIDR.(string))
		} else {
			exoConfig = azure.Managed(d.Get(keyRegion).(string), d.Get(keySubnet).(string))
		}
		exoConfigID, err := azure.Wrap(client).AddExocomputeConfig(ctx, azure.CloudAccountID(accountID), exoConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(exoConfigID.String())
	}

	azureReadExocompute(ctx, d, m)
	return nil
}

// azureReadExocompute run the Read operation for the Azure exocompute
// resource. This reads the remote state of the exocompute config in RSC.
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

		hostID, err := azure.Wrap(client).ExocomputeHostAccount(ctx, azure.CloudAccountID(appID))
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

		exoConfig, err := azure.Wrap(client).ExocomputeConfig(ctx, exoConfigID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyRegion, exoConfig.Region); err != nil {
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

// azureDeleteExocompute run the Delete operation for the Azure exocompute
// resource. This removes the exocompute config from RSC.
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
		err = azure.Wrap(client).UnmapExocompute(ctx, azure.CloudAccountID(appID))
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		exoConfigID, err := uuid.Parse(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		err = azure.Wrap(client).RemoveExocomputeConfig(ctx, exoConfigID)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}
