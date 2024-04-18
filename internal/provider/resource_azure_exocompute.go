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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

// resourceAzureExocompute defines the schema for the Azure exocompute resource.
func resourceAzureExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateExocompute,
		ReadContext:   azureReadExocompute,
		DeleteContext: azureDeleteExocompute,

		Description: "The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration. When an " +
			"Exocompute configuration is created, RSC will automatically deploy the necessary resources in the " +
			"specified Azure region to run the Exocompute service.",
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
					"which the Exocompute service runs.",
				ValidateFunc: validation.IsUUID,
			},
			keySubscriptionID: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for " +
					"which the Exocompute service runs. **Deprecated:** use `cloud_account_id` instead.",
				Deprecated:   "use `cloud_account_id` instead.",
				ValidateFunc: validation.IsUUID,
			},
			keyRegion: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Azure region to run the exocompute service in. Should be specified in the standard " +
					"Azure style, e.g. `eastus`.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keySubnet: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure subnet id.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
	region := d.Get(keyRegion).(string)

	exoConfig := azure.Managed(region, d.Get(keySubnet).(string))
	exoConfigID, err := azure.Wrap(client).AddExocomputeConfig(ctx, azure.CloudAccountID(accountID), exoConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(exoConfigID.String())
	awsReadExocompute(ctx, d, m)
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

	exoConfigID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	exoConfig, err := azure.Wrap(client).ExocomputeConfig(ctx, exoConfigID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	} else if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyRegion, exoConfig.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySubnet, exoConfig.SubnetID); err != nil {
		return diag.FromErr(err)
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

	exoConfigID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = azure.Wrap(client).RemoveExocomputeConfig(ctx, exoConfigID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
