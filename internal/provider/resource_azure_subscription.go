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

	// Check if the subscription already exist in Polaris.
	account, err := client.Azure().Subscription(ctx, azure.SubscriptionID(subscriptionID), core.CloudNativeProtection)
	switch {
	case errors.Is(err, graphql.ErrNotFound):
	case err == nil:
		return diag.Errorf("subscription %q already added to polaris", account.NativeID)
	case err != nil:
		return diag.FromErr(err)
	}

	var opts []azure.OptionFunc
	if name, ok := d.GetOk("name"); ok {
		opts = append(opts, azure.Name(name.(string)))
	}
	for _, region := range d.Get("regions").(*schema.Set).List() {
		opts = append(opts, azure.Region(region.(string)))
	}

	tenantDomain := d.Get("tenant_domain").(string)
	id, err := client.Azure().AddSubscription(ctx, azure.Subscription(subscriptionID, tenantDomain), opts...)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id.String())

	// Populate the local Terraform state.
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

	d.Set("subscription_name", account.Name)
	d.Set("tenant_domain", account.TenantDomain)

	regions := schema.Set{F: schema.HashString}
	for _, region := range account.Features[0].Regions {
		regions.Add(region)
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

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

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
