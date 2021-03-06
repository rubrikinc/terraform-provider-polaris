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
	credentials   = "{{ .Resource.Credentials }}"
	tenant_domain = "{{ .Resource.TenantDomain }}"
}

resource "polaris_azure_subscription" "default" {
	subscription_id   = "{{ .Resource.SubscriptionID }}"
	subscription_name = "{{ .Resource.SubscriptionName }}"
	tenant_domain     = "{{ .Resource.TenantDomain }}"

	cloud_native_protection {
		regions = [
			"eastus2",
		]
	}

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
			Config: exocompute,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),

				// Exocompute feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "exocompute.0.regions.*", "eastus2"),

				// Exocompute resource
				resource.TestCheckResourceAttrPair("polaris_azure_exocompute.default", "subscription_id", "polaris_azure_subscription.default", "id"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "region", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "polaris_managed", "true"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "subnet", subscription.Exocompute.SubnetID),
			),
		}},
	})
}
