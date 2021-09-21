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
	regions = [
		"us-east-2",
	]
}
`

const awsAccountTwoRegionsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"
	regions = [
		"us-east-2",
		"us-west-2",
	]
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

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			PreConfig: testStepDelay,
			Config:    accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
			),
		}, {
			PreConfig: testStepDelay,
			Config:    accountTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "regions.*", "us-east-2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "regions.*", "us-west-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
			),
		}, {
			PreConfig: testStepDelay,
			Config:    accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
			),
		}},
	})
}
