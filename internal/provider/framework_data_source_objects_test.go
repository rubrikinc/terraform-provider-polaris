// Copyright 2026 Rubrik, Inc.
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
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// subscriptionOnlyTmpl onboards the Azure subscription alone, with no data
// sources - used as step 1 so we can insert a real Go-level sleep (PreConfig)
// before step 2 reads the objects data sources, decoupled from any
// in-config wait mechanism.
const subscriptionOnlyTmpl = `
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

const objectsAzureResourceGroupTmpl = subscriptionOnlyTmpl + `
data "polaris_objects" "resource_groups" {
	object_type     = "AzureNativeResourceGroup"
	subscription_id = polaris_azure_subscription.default.id

	depends_on = [polaris_azure_subscription.default]
}

data "polaris_objects" "all_subscriptions" {
	object_type = "AzureNativeResourceGroup"

	depends_on = [polaris_azure_subscription.default]
}
`

func TestAccPolarisObjectsDataSource_azureResourceGroup(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	subscriptionOnlyConfig, err := makeTerraformConfig(config, subscriptionOnlyTmpl)
	if err != nil {
		t.Fatal(err)
	}
	objectsConfig, err := makeTerraformConfig(config, objectsAzureResourceGroupTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resourceGroupCheck := knownvalue.SetPartial([]knownvalue.Check{
		knownvalue.ObjectPartial(map[string]knownvalue.Check{
			keyName: knownvalue.StringExact(subscription.CloudNativeProtection.ResourceGroupName),
		}),
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Onboard the subscription alone first.
				Config: subscriptionOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
					resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				),
			},
			{
				// Sleep for real wall-clock time before adding the objects
				// data sources. After a subscription connects, RSC needs to
				// run native discovery before its resource groups become
				// visible to azureNativeResourceGroups (~1 minute observed);
				// wait with margin so the read is not racing discovery.
				PreConfig: func() { time.Sleep(2 * time.Minute) },
				Config:    objectsConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Scoped to the fixture's subscription.
					statecheck.ExpectKnownValue("data.polaris_objects.resource_groups", tfjsonpath.New(keyID),
						knownvalue.StringRegexp(sha256Hex)),
					statecheck.ExpectKnownValue("data.polaris_objects.resource_groups", tfjsonpath.New(keyObjects),
						resourceGroupCheck),

					// Searching across all subscriptions still finds it.
					statecheck.ExpectKnownValue("data.polaris_objects.all_subscriptions", tfjsonpath.New(keyObjects),
						resourceGroupCheck),
				},
			},
		},
	})
}
