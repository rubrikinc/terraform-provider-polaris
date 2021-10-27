package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const azureSubscriptionOneRegionTmpl = `
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
  
	depends_on = [polaris_azure_service_principal.default]
}
`

const azureSubscriptionTwoRegionsTmpl = `
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
			"westus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
`

func TestAccPolarisAzureSubscription_basic(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	subscriptionOneRegion, err := makeTerraformConfig(config, azureSubscriptionOneRegionTmpl)
	if err != nil {
		t.Fatal(err)
	}

	subscriptionTwoRegions, err := makeTerraformConfig(config, azureSubscriptionTwoRegionsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			PreConfig: testStepDelay,
			Config:    subscriptionOneRegion,
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
			),
		}, {
			PreConfig: testStepDelay,
			Config:    subscriptionTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "westus2"),
			),
		}, {
			PreConfig: testStepDelay,
			Config:    subscriptionOneRegion,
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
			),
		}},
	})
}
