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
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	graphql_azure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// validateAzureRegion verifies that the name is a valid Azure region name.
func validateAzureRegion(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := graphql_azure.ParseRegion(m.(string))
	return diag.FromErr(err)
}

// resourceAzureSubscription defines the schema for the Azure subscription
// resource.
func resourceAzureSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateSubscription,
		ReadContext:   azureReadSubscription,
		UpdateContext: azureUpdateSubscription,
		DeleteContext: azureDeleteSubscription,

		Schema: map[string]*schema.Schema{
			"cloud_native_protection": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateAzureRegion,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions that Polaris will monitor for instances to automatically protect.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
					},
				},
				MaxItems:    1,
				Required:    true,
				Description: "Enable the Cloud Native Protection feature for the GCP project.",
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"exocompute": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateAzureRegion,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the exocompute feature in.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Exocompute feature.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the exocompute feature for the account.",
			},
			"subscription_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Subscription id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"subscription_name": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Subscription name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_domain": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Tenant directory/domain name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},

		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureSubscriptionV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureSubscriptionStateUpgradeV0,
			Version: 0,
		}, {
			Type:    resourceAzureSubscriptionV1().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureSubscriptionStateUpgradeV1,
			Version: 1,
		}},
	}
}

// azureCreateSubscription run the Create operation for the Azure subscription
// resource. This adds the Azure subscription to the Polaris platform.
func azureCreateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateSubscription")

	client := m.(*polaris.Client)

	subscriptionID, err := uuid.Parse(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	tenantDomain := d.Get("tenant_domain").(string)

	var opts []azure.OptionFunc
	if name, ok := d.GetOk("subscription_name"); ok {
		opts = append(opts, azure.Name(name.(string)))
	}

	// Check if the subscription already exist in Polaris.
	account, err := client.Azure().Subscription(ctx, azure.SubscriptionID(subscriptionID), core.FeatureAll)
	if err == nil {
		return diag.Errorf("subscription %q already added to polaris", account.NativeID)
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return diag.FromErr(err)
	}

	// Polaris Cloud Account id. Returned when the account is added for the
	// cloud native protection feature.
	var id uuid.UUID

	cnpBlock, ok := d.GetOk("cloud_native_protection")
	if ok {
		block := cnpBlock.([]interface{})[0].(map[string]interface{})

		var cnpOpts []azure.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
			cnpOpts = append(cnpOpts, azure.Region(region.(string)))
		}

		cnpOpts = append(cnpOpts, opts...)
		id, err = client.Azure().AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain),
			core.FeatureCloudNativeProtection, cnpOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	exoBlock, ok := d.GetOk("exocompute")
	if ok {
		block := exoBlock.([]interface{})[0].(map[string]interface{})

		var exoOpts []azure.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
			exoOpts = append(exoOpts, azure.Region(region.(string)))
		}

		exoOpts = append(exoOpts, opts...)
		_, err := client.Azure().AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain),
			core.FeatureExocompute, exoOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(id.String())

	azureReadSubscription(ctx, d, m)
	return nil
}

// azureReadSubscription run the Read operation for the Azure subscription
// resource. This reads the state of the Azure subscription in Polaris.
func azureReadSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadSubscription")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the Polaris cloud account using the cloud account id.
	account, err := client.Azure().Subscription(ctx, azure.CloudAccountID(id), core.FeatureAll)
	if err != nil {
		return diag.FromErr(err)
	}

	cnpFeature, ok := account.Feature(core.FeatureCloudNativeProtection)
	if ok {
		regions := schema.Set{F: schema.HashString}
		for _, region := range cnpFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(cnpFeature.Status)
		err := d.Set("cloud_native_protection", []interface{}{
			map[string]interface{}{
				"regions": &regions,
				"status":  &status,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("cloud_native_protection", nil); err != nil {
			return diag.FromErr(err)
		}
	}

	exoFeature, ok := account.Feature(core.FeatureExocompute)
	if ok {
		regions := schema.Set{F: schema.HashString}
		for _, region := range exoFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(exoFeature.Status)
		err := d.Set("exocompute", []interface{}{
			map[string]interface{}{
				"regions": &regions,
				"status":  &status,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("exocompute", nil); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("subscription_name", account.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("tenant_domain", account.TenantDomain); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// azureUpdateSubscription run the Update operation for the Azure subscription
// resource. This updates the Azure subscription in Polaris.
func azureUpdateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateSubscription")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("cloud_native_protection") {
		cnpBlock, ok := d.GetOk("cloud_native_protection")
		if ok {
			block := cnpBlock.([]interface{})[0].(map[string]interface{})

			var opts []azure.OptionFunc
			for _, region := range block["regions"].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}

			if err := client.Azure().UpdateSubscription(ctx, azure.CloudAccountID(id), core.FeatureCloudNativeProtection, opts...); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if _, ok := d.GetOk("exocompute"); ok {
				return diag.Errorf("cloud native protection is required by exocompute")
			}

			snapshots := d.Get("delete_snapshots_on_destroy").(bool)
			if err := client.Azure().RemoveSubscription(ctx, azure.CloudAccountID(id), core.FeatureCloudNativeProtection, snapshots); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("exocompute") {
		oldExoBlock, newExoBlock := d.GetChange("exocompute")
		oldExoList := oldExoBlock.([]interface{})
		newExoList := newExoBlock.([]interface{})

		// Determine whether we are adding, removing or updating the Exocompute
		// feature.
		switch {
		case len(oldExoList) == 0:
			var opts []azure.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}

			subscriptionID, err := uuid.Parse(d.Get("subscription_id").(string))
			if err != nil {
				return diag.FromErr(err)
			}

			tenantDomain := d.Get("tenant_domain").(string)
			_, err = client.Azure().AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain),
				core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		case len(newExoList) == 0:
			err := client.Azure().RemoveSubscription(ctx, azure.CloudAccountID(id), core.FeatureExocompute, false)
			if err != nil {
				return diag.FromErr(err)
			}
		default:
			var opts []azure.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}

			err = client.Azure().UpdateSubscription(ctx, azure.CloudAccountID(id), core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("subscription_name") {
		opts := []azure.OptionFunc{azure.Name(d.Get("subscription_name").(string))}
		err = client.Azure().UpdateSubscription(ctx, azure.CloudAccountID(id), core.FeatureCloudNativeProtection, opts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadSubscription(ctx, d, m)
	return nil
}

// azureDeleteSubscription run the Delete operation for the Azure subscription
// resource. This removes the Azure subscription from Polaris.
func azureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteSubscription")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the old resource arguments.
	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	if _, ok := d.GetOk("exocompute"); ok {
		err = client.Azure().RemoveSubscription(ctx, azure.CloudAccountID(id), core.FeatureExocompute, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk("cloud_native_protection"); ok {
		err = client.Azure().RemoveSubscription(ctx, azure.CloudAccountID(id), core.FeatureCloudNativeProtection, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")

	return nil
}
