// Copyright 2025 Rubrik, Inc.
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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPolarisPermissionGroupsAWS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `data "polaris_permission_groups" "aws" { cloud_provider = "AWS" }`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.aws", "id"),
				resource.TestCheckResourceAttr("data.polaris_permission_groups.aws", "cloud_provider", "AWS"),
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.aws", "features.#"),
			),
		}},
	})
}

func TestAccPolarisPermissionGroupsAzure(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `data "polaris_permission_groups" "azure" { cloud_provider = "AZURE" }`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.azure", "id"),
				resource.TestCheckResourceAttr("data.polaris_permission_groups.azure", "cloud_provider", "AZURE"),
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.azure", "features.#"),
			),
		}},
	})
}

func TestAccPolarisPermissionGroupsGCP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `data "polaris_permission_groups" "gcp" { cloud_provider = "GCP" }`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.gcp", "id"),
				resource.TestCheckResourceAttr("data.polaris_permission_groups.gcp", "cloud_provider", "GCP"),
				resource.TestCheckResourceAttrSet("data.polaris_permission_groups.gcp", "features.#"),
			),
		}},
	})
}

func TestAccPolarisPermissionGroupsInvalidProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config:      `data "polaris_permission_groups" "invalid" { cloud_provider = "INVALID" }`,
			ExpectError: regexp.MustCompile(`expected cloud_provider to be one of`),
		}},
	})
}
