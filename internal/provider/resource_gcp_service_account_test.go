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
	"crypto/sha256"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const gcpServiceAccountWithDefaultNameTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_service_account" "default" {
	credentials = "{{ .Resource.Credentials }}"

	lifecycle {
		ignore_changes = [name]
	}
}
`

const gcpServiceAccountWithNameTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_service_account" "default" {
	credentials = "{{ .Resource.Credentials }}"
	name        = "test-name"

	lifecycle {
		ignore_changes = [name]
	}
}
`

// The RSC GraphQL API appears to be caching the service account name in the
// API-server. So the wrong service account name might be returned depending on
// which API-server instance you get connected to. Therefore, we cannot verify
// that the correct name has been set.
func TestAccPolarisGCPServiceAccount_basic(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	serviceAccountWithDefaultName, err := makeTerraformConfig(config, gcpServiceAccountWithDefaultNameTmpl)
	if err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: serviceAccountWithDefaultName,
			Check: resource.ComposeTestCheckFunc(
				gcpCheckServiceAccountID("polaris_gcp_service_account.default", ""),
				resource.TestCheckResourceAttr("polaris_gcp_service_account.default", "credentials", project.Credentials),
			),
		}},
	})

	serviceAccountWithName, err := makeTerraformConfig(config, gcpServiceAccountWithNameTmpl)
	if err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: serviceAccountWithName,
			Check: resource.ComposeTestCheckFunc(
				gcpCheckServiceAccountID("polaris_gcp_service_account.default", "test-name"),
				resource.TestCheckResourceAttr("polaris_gcp_service_account.default", "credentials", project.Credentials),
			),
		}},
	})
}

// gcpCheckServiceAccountID checks that the resource ID is the SHA-256 sum of
// the service account name. Note, the returned error messages are written to
// follow the format used by the Terraform SDK.
func gcpCheckServiceAccountID(resourceName, serviceAccountName string) func(state *terraform.State) error {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s: Not found in %s", resourceName, state.RootModule().Path)
		}
		inst := res.Primary
		if inst == nil {
			return fmt.Errorf("%s: No primary instance in %s", resourceName, state.RootModule().Path)
		}

		// If an empty service account name is passed in we assume the service
		// account was onboarded using the default generated service account
		// name. Since we cannot reliably read out the generated name, we fall
		// back to verifying that the ID is a valid SHA-256 hash.
		if serviceAccountName == "" {
			ok, err := regexp.Match("^[0-9a-f]{64}$", []byte(inst.ID))
			if err != nil {
				return fmt.Errorf("failed to compile regexp: %s", err)
			}
			if !ok {
				return fmt.Errorf("%s: Attribute 'id' expected to be a SHA-256 hash, got %#v", resourceName, inst.ID)
			}
		} else {
			id := fmt.Sprintf("%x", sha256.Sum256([]byte(serviceAccountName)))
			if inst.ID != id {
				return fmt.Errorf("%s: Attribute 'id' expected %#v, got %#v", resourceName, id, inst.ID)
			}
		}

		return nil
	}
}
