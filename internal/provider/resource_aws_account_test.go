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

const awsAccountOneRegionTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
		]
	}
}
`

const awsAccountTwoRegionsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
			"us-west-2",
		]
	}
}
`

const awsCrossAccountOneRegionTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	assume_role = "{{ .Resource.CrossAccountRole }}"
	name        = "{{ .Resource.CrossAccountName }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
		]
	}
}
`

const awsCrossAccountTwoRegionsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	assume_role = "{{ .Resource.CrossAccountRole }}"
	name        = "{{ .Resource.CrossAccountName }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
			"us-west-2",
		]
	}
}
`

func TestAccPolarisAWSAccount_basic(t *testing.T) {
	config, account, err := loadAWSTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	accountOneRegion, err := makeTerraformConfig(config, awsAccountOneRegionTmpl)
	if err != nil {
		t.Fatal(err)
	}
	accountTwoRegions, err := makeTerraformConfig(config, awsAccountTwoRegionsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Add and update account using a profile
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
			),
		}, {
			Config: accountTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-west-2"),
			),
		}, {
			Config: accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
			),
		}},
	})

	crossAccountOneRegion, err := makeTerraformConfig(config, awsCrossAccountOneRegionTmpl)
	if err != nil {
		t.Fatal(err)
	}
	crossAccountTwoRegions, err := makeTerraformConfig(config, awsCrossAccountTwoRegionsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Add and update account using cross account role. This test uses the
	// default profile to assume the role.
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: crossAccountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
			),
		}, {
			Config: crossAccountTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-west-2"),
			),
		}, {
			Config: crossAccountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
			),
		}},
	})
}
