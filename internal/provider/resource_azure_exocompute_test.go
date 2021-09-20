package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var azureExocomputeTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_azure_service_principal" "default" {
	credentials = "{{ .Resource.Credentials }}"
}

resource "polaris_azure_subscription" "default" {
	subscription_id   = "{{ .Resource.SubscriptionID }}"
	subscription_name = "{{ .Resource.SubscriptionName }}"
	tenant_domain     = "{{ .Resource.TenantDomain }}"
	regions           = [
		"eastus2",
	]

	exocompute {
		regions = [
			"eastus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
  
resource "polaris_azure_exocompute" "default" {
	subscription_id = polaris_azure_subscription.default.id
	region          = "eastus2"
	subnet          = "{{ .Resource.Exocompute.SubnetID }}"
}  
`

func TestAccPolarisAzureExocompute_basic(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	exocompute, err := makeTerraformConfig(config, azureExocomputeTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			PreConfig: testStepDelay,
			Config:    exocompute,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "regions.*", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "exocompute.0.regions.*", "eastus2"),

				resource.TestCheckResourceAttrPair("polaris_azure_exocompute.default", "subscription_id", "polaris_azure_subscription.default", "id"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "region", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "polaris_managed", "true"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "subnet", subscription.Exocompute.SubnetID),
			),
		}},
	})
}
