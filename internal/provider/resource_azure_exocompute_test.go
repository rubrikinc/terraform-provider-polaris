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
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

		regions = [
			"eastus2",
		]
	}

	exocompute {
		resource_group_name   = "{{ .Resource.Exocompute.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.Exocompute.ResourceGroupRegion }}"

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
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_name",
					subscription.CloudNativeProtection.ResourceGroupName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_region",
					subscription.CloudNativeProtection.ResourceGroupRegion),

				// Exocompute feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "exocompute.0.regions.*", "eastus2"),

				// Exocompute resource
				resource.TestCheckResourceAttrPair("polaris_azure_exocompute.default", "subscription_id", "polaris_azure_subscription.default", "id"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "region", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "subnet", subscription.Exocompute.SubnetID),
			),
		}},
	})
}
