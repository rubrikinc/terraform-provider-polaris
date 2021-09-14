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

func validateAzureRegion(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := graphql_azure.ParseRegion(m.(string))
	return diag.FromErr(err)
}

// resourceAzureSubcription defines the schema for the Azure subscription
// resource.
func resourceAzureSubcription() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateSubscription,
		ReadContext:   azureReadSubscription,
		UpdateContext: azureUpdateSubscription,
		DeleteContext: azureDeleteSubscription,

		Schema: map[string]*schema.Schema{
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
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the exocompute feature for the account.",
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateAzureRegion,
				},
				Required:    true,
				Description: "Regions that Polaris will monitor for instances to automatically protect.",
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

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureSubcriptionV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureProjectStateUpgradeV0,
			Version: 0,
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

	var opts []azure.OptionFunc
	if name, ok := d.GetOk("subscription_name"); ok {
		opts = append(opts, azure.Name(name.(string)))
	}

	regions := d.Get("regions").(*schema.Set)
	for _, region := range regions.List() {
		opts = append(opts, azure.Region(region.(string)))
	}

	// Exocompute parameter, optional. Verify the regions specified and
	// guarantee that it's nil if it's not specified.
	exocompute, ok := d.GetOk("exocompute")
	if ok {
		block := exocompute.([]interface{})[0].(map[string]interface{})
		for _, region := range block["regions"].(*schema.Set).List() {
			if !regions.Contains(region) {
				return diag.Errorf("exocompute can only have a subset of the subscription regions")
			}
		}
	} else {
		exocompute = nil
	}

	tenantDomain := d.Get("tenant_domain").(string)

	// Check if the subscription already exist in Polaris.
	account, err := client.Azure().Subscription(ctx, azure.SubscriptionID(subscriptionID), core.CloudNativeProtection)
	switch {
	case errors.Is(err, graphql.ErrNotFound):
	case err == nil:
		return diag.Errorf("subscription %q already added to polaris", account.NativeID)
	case err != nil:
		return diag.FromErr(err)
	}

	id, err := client.Azure().AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain), opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	// Enable the Exocompute feature if specified.
	if exocompute != nil {
		block := exocompute.([]interface{})[0].(map[string]interface{})

		var regions []string
		for _, region := range block["regions"].(*schema.Set).List() {
			regions = append(regions, region.(string))
		}

		err := client.Azure().EnableExocompute(ctx, azure.SubscriptionID(subscriptionID), regions...)
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

	account, err := client.Azure().Subscription(ctx, azure.CloudAccountID(id), core.CloudNativeProtection)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(account.Features) != 1 {
		return diag.Errorf("expected a single feature got multiple")
	}

	if err := d.Set("subscription_name", account.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("tenant_domain", account.TenantDomain); err != nil {
		return diag.FromErr(err)
	}

	cnpRegions := schema.Set{F: schema.HashString}
	for _, region := range account.Features[0].Regions {
		cnpRegions.Add(region)
	}
	if err := d.Set("regions", &cnpRegions); err != nil {
		return diag.FromErr(err)
	}

	// Read the exocompute feature.
	account, err = client.Azure().Subscription(ctx, azure.CloudAccountID(id), core.Exocompute)
	if err != nil && !errors.Is(err, graphql.ErrNotFound) {
		return diag.FromErr(err)
	}
	if err == nil {
		if len(account.Features[0].Regions) > 0 {
			exoRegions := schema.Set{F: schema.HashString}
			for _, region := range account.Features[0].Regions {
				exoRegions.Add(region)
			}
			block := []interface{}{
				map[string]interface{}{
					"regions": &exoRegions,
				},
			}
			if err := d.Set("exocompute", block); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := d.Set("exocompute", nil); err != nil {
				return diag.FromErr(err)
			}
		}
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

	subscriptionID, err := uuid.Parse(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Regions needed due to the existing exocompute resources.
	exoConfigs, err := client.Azure().ExocomputeConfigs(ctx, azure.CloudAccountID(id))
	if err != nil {
		return diag.FromErr(err)
	}
	exoConfigRegions := make(map[string]struct{})
	for _, exoConfig := range exoConfigs {
		exoConfigRegions[exoConfig.Region] = struct{}{}
	}

	// Regions specified by the exocompute feature.
	exoRegions := make(map[string]struct{})
	if exocompute, ok := d.GetOk("exocompute"); ok {
		block := exocompute.([]interface{})[0].(map[string]interface{})
		for _, r := range block["regions"].(*schema.Set).List() {
			exoRegions[r.(string)] = struct{}{}
		}
	}
	for region := range exoConfigRegions {
		if _, ok := exoRegions[region]; !ok {
			return diag.Errorf("exocompute feature regions must be a superset of exocompute resource regions")
		}
	}

	// Regions specified by the cloud native protection feature.
	cnpRegions := make(map[string]struct{})
	for _, region := range d.Get("regions").(*schema.Set).List() {
		cnpRegions[region.(string)] = struct{}{}
	}
	for region := range exoRegions {
		if _, ok := cnpRegions[region]; !ok {
			return diag.Errorf("subscription regions must be a superset of exocompute feature regions")
		}
	}

	if d.HasChange("subscription_name") || d.HasChange("regions") {
		var opts []azure.OptionFunc
		if d.HasChange("subscription_name") {
			opts = append(opts, azure.Name(d.Get("subscription_name").(string)))

		}

		if d.HasChange("regions") {
			for _, region := range d.Get("regions").(*schema.Set).List() {
				opts = append(opts, azure.Region(region.(string)))
			}
		}

		err = client.Azure().UpdateSubscription(ctx, azure.CloudAccountID(id), core.CloudNativeProtection, opts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("exocompute") {
		if _, ok := d.GetOk("exocompute"); ok {
			var regions []string
			for region := range exoRegions {
				regions = append(regions, region)
			}

			err := client.Azure().EnableExocompute(ctx, azure.SubscriptionID(subscriptionID), regions...)
			if errors.Is(err, graphql.ErrAlreadyEnabled) {
				err = client.Azure().UpdateSubscription(ctx, azure.CloudAccountID(id), core.Exocompute,
					azure.Regions(regions...))
			}
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			err := client.Azure().DisableExocompute(ctx, azure.SubscriptionID(subscriptionID))
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

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

	err = client.Azure().RemoveSubscription(ctx, azure.CloudAccountID(id), deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
