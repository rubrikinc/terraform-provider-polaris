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

const azureServicePrincipalTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_azure_service_principal" "default" {
	credentials   = "{{ .Resource.Credentials }}"
	tenant_domain = "{{ .Resource.TenantDomain }}"
}
`

const azureServicePrincipalFromValues = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_azure_service_principal" "default" {
	app_id        = "{{ .Resource.PrincipalID }}"
	app_name      = "{{ .Resource.PrincipalName }}"
	app_secret    = "{{ .Resource.PrincipalSecret }}"
	tenant_id     = "{{ .Resource.TenantID }}"
	tenant_domain = "{{ .Resource.TenantDomain }}"
}
`

func TestAccPolarisAzureServicePrincipal_basic(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	servicePrincipal, err := makeTerraformConfig(config, azureServicePrincipalTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: servicePrincipal,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "id", subscription.PrincipalID),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "credentials", subscription.Credentials),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckNoResourceAttr("polaris_azure_service_principal.default", "app_id"),
				resource.TestCheckNoResourceAttr("polaris_azure_service_principal.default", "app_name"),
				resource.TestCheckNoResourceAttr("polaris_azure_service_principal.default", "app_secret"),
				resource.TestCheckNoResourceAttr("polaris_azure_service_principal.default", "tenant_id"),
			),
		}},
	})

	servicePrincipal, err = makeTerraformConfig(config, azureServicePrincipalFromValues)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: servicePrincipal,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "id", subscription.PrincipalID),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "app_id", subscription.PrincipalID),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "app_name", subscription.PrincipalName),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "app_secret", subscription.PrincipalSecret),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "tenant_id", subscription.TenantID),
				resource.TestCheckResourceAttr("polaris_azure_service_principal.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckNoResourceAttr("polaris_azure_service_principal.default", "credentials"),
			),
		}},
	})
}
