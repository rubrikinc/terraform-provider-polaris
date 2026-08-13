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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

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
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

		regions = [
			"eastus2",
			"westus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
`

const azureSubscriptionEntraID1Tmpl = `
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
	entra_group_id    = "00000000-0000-0000-0000-000000000001"

	cloud_native_protection {
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

		regions = [
			"eastus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
`

const azureSubscriptionEntraID2Tmpl = `
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
	entra_group_id    = "00000000-0000-0000-0000-000000000002"

	cloud_native_protection {
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

		regions = [
			"eastus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
`

const azureSubscriptionPostgresTmpl = `
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

	postgres_flexible_server_protection {
		resource_group_name   = "{{ .Resource.PostgresFlexibleServer.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.PostgresFlexibleServer.ResourceGroupRegion }}"

		user_assigned_managed_identity_name                = "{{ .Resource.PostgresFlexibleServer.ManagedIdentityName }}"
		user_assigned_managed_identity_principal_id        = "{{ .Resource.PostgresFlexibleServer.ManagedIdentityPrincipalID }}"
		user_assigned_managed_identity_region              = "{{ .Resource.PostgresFlexibleServer.ManagedIdentityRegion }}"
		user_assigned_managed_identity_resource_group_name = "{{ .Resource.PostgresFlexibleServer.ResourceGroupName }}"

		regions = [
{{- range .Resource.PostgresFlexibleServer.Regions }}
			"{{ . }}",
{{- end }}
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
`

// TestAccPolarisAzureSubscription_postgresFlexibleServer onboards and offboards
// the Postgres Flexible Server Protection feature.
//
// The feature is gated in RSC behind the
// REL_ENABLE_AZURE_POSTGRES_FLEXIBLE_SERVER feature flag, and unlike the other
// features it requires both a resource group and a user-assigned managed
// identity that already exist in Azure, with the identity in the feature's
// resource group. The test therefore skips unless the postgresFlexibleServer
// section is present in TEST_AZURESUBSCRIPTION_FILE.
func TestAccPolarisAzureSubscription_postgresFlexibleServer(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	if subscription.PostgresFlexibleServer.ResourceGroupName == "" ||
		subscription.PostgresFlexibleServer.ManagedIdentityName == "" {
		t.Skip("skipping, postgresFlexibleServer is not configured in TEST_AZURESUBSCRIPTION_FILE")
	}

	subscriptionPostgres, err := makeTerraformConfig(config, azureSubscriptionPostgresTmpl)
	if err != nil {
		t.Fatal(err)
	}

	checks := []resource.TestCheckFunc{
		// Subscription resource
		resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
		resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),

		// Postgres Flexible Server Protection feature
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.status", "CONNECTED"),
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.regions.#",
			strconv.Itoa(len(subscription.PostgresFlexibleServer.Regions))),
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.resource_group_name",
			subscription.PostgresFlexibleServer.ResourceGroupName),
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.resource_group_region",
			subscription.PostgresFlexibleServer.ResourceGroupRegion),

		// RSC requires the managed identity to be in the feature's
		// resource group, so the two must agree in state.
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.user_assigned_managed_identity_name",
			subscription.PostgresFlexibleServer.ManagedIdentityName),
		resource.TestCheckResourceAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.user_assigned_managed_identity_resource_group_name",
			subscription.PostgresFlexibleServer.ResourceGroupName),
	}
	for _, region := range subscription.PostgresFlexibleServer.Regions {
		checks = append(checks, resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default",
			"postgres_flexible_server_protection.0.regions.*", region))
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: subscriptionPostgres,
			Check:  resource.ComposeTestCheckFunc(checks...),
		}, {
			// Verify that the subscription can be imported. RSC does not expose
			// the feature's user-assigned managed identity, see
			// updateAzureFeatureState, so those fields are empty in the imported
			// state and cannot be compared against the pre-import state. The
			// first apply after an import reconciles them from the configuration
			// without re-onboarding the feature.
			ResourceName:      "polaris_azure_subscription.default",
			ImportState:       true,
			ImportStateVerify: true,
			ImportStateVerifyIgnore: []string{
				"postgres_flexible_server_protection.0.user_assigned_managed_identity_name",
				"postgres_flexible_server_protection.0.user_assigned_managed_identity_principal_id",
				"postgres_flexible_server_protection.0.user_assigned_managed_identity_region",
				"postgres_flexible_server_protection.0.user_assigned_managed_identity_resource_group_name",
			},
		}},
	})
}

func TestAccPolarisAzureSubscription_entraGroupID(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	entraID1Config, err := makeTerraformConfig(config, azureSubscriptionEntraID1Tmpl)
	if err != nil {
		t.Fatal(err)
	}

	entraID2Config, err := makeTerraformConfig(config, azureSubscriptionEntraID2Tmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: entraID1Config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "entra_group_id", "00000000-0000-0000-0000-000000000001"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
			),
		}, {
			Config: entraID2Config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "entra_group_id", "00000000-0000-0000-0000-000000000002"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
			),
		}},
	})
}

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
			Config: subscriptionOneRegion,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_name",
					subscription.CloudNativeProtection.ResourceGroupName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_region",
					subscription.CloudNativeProtection.ResourceGroupRegion),
			),
		}, {
			Config: subscriptionTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "westus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_name",
					subscription.CloudNativeProtection.ResourceGroupName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_region",
					subscription.CloudNativeProtection.ResourceGroupRegion),
			),
		}, {
			Config: subscriptionOneRegion,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_name",
					subscription.CloudNativeProtection.ResourceGroupName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_region",
					subscription.CloudNativeProtection.ResourceGroupRegion),
			),
		}},
	})
}
