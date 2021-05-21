package polaris

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

// validateAzureRegion -
func validateAzureRegion(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := graphql.AzureParseRegion(m.(string))
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
				Description: "What should happen to snapshots when the project is removed from Polaris.",
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateAzureRegion,
				},
				Required:    true,
				Description: "Polaris will auto-discover instances to be protected from the specified regions.",
			},
			"subscription_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "Subscription id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"subscription_name": {
				Type:             schema.TypeString,
				Required:         true,
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

	regions := make([]graphql.AzureRegion, 0, 10)
	for _, region := range d.Get("regions").(*schema.Set).List() {
		azureRegion, err := graphql.AzureParseRegion(region.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		regions = append(regions, azureRegion)
	}

	err = client.AzureSubscriptionAdd(ctx, polaris.AzureSubscriptionIn{
		Cloud:        graphql.AzurePublic,
		ID:           subscriptionID,
		Name:         d.Get("subscription_name").(string),
		TenantDomain: d.Get("tenant_domain").(string),
		Regions:      regions,
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(subscriptionID.String())

	return azureReadSubscription(ctx, d, m)
}

// azureReadSubscription run the Read operation for the Azure subscription
// resource. This reads the state of the Azure subscription in Polaris.
func azureReadSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadSubscription")

	client := m.(*polaris.Client)

	subscription, err := client.AzureSubscription(ctx, polaris.WithAzureSubscriptionID(d.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("subscription_name", subscription.Name)
	d.Set("tenant_domain", subscription.TenantDomain)

	regions := schema.Set{F: schema.HashString}
	for _, region := range subscription.Feature.Regions {
		regions.Add(graphql.AzureFormatRegion(region))
	}
	if err := d.Set("regions", &regions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// azureUpdateSubscription run the Update operation for the Azure subscription
// resource. This updates the Azure subscription in Polaris.
func azureUpdateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateSubscription")

	client := m.(*polaris.Client)

	// Update subscription name.
	if d.HasChange("subscription_name") {
		name := d.Get("subscription_name").(string)
		err := client.AzureSubscriptionSetName(ctx, polaris.WithAzureSubscriptionID(d.Id()), name)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Update regions.
	if d.HasChange("regions") {
		regions := make([]graphql.AzureRegion, 0, 10)
		for _, region := range d.Get("regions").(*schema.Set).List() {
			azureRegion, err := graphql.AzureParseRegion(region.(string))
			if err != nil {
				return diag.FromErr(err)
			}
			regions = append(regions, azureRegion)
		}

		err := client.AzureSubscriptionSetRegions(ctx, polaris.WithAzureSubscriptionID(d.Id()), regions...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// azureDeleteSubscription run the Delete operation for the Azure subscription
// resource. This removes the Azure subscription from Polaris.
func azureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteSubscription")

	client := m.(*polaris.Client)

	// Get the old resource arguments.
	oldSubscriptionID, _ := d.GetChange("subscription_id")
	subscriptionID := oldSubscriptionID.(string)

	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	err := client.AzureSubscriptionRemove(ctx, polaris.WithAzureSubscriptionID(subscriptionID), deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
