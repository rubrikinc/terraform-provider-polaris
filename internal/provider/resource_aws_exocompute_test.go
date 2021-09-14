package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var awsExocomputeTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"
	regions = [
		"us-east-2",
	]
  
	exocompute {
		regions = [
			"us-east-2",
		]
	}
}

resource "polaris_aws_exocompute" "default" {
	account_id = polaris_aws_account.default.id
	region     = "us-east-2"
	vpc_id     = "{{ .Resource.Exocompute.VPCID }}"

	subnets = [
		{{ range .Resource.Exocompute.Subnets }}
		"{{ .ID }}",
		{{ end }}
	]
}
`

func TestAccPolarisAWSExocompute_basic(t *testing.T) {
	config, account, err := loadAWSTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	exocompute, err := makeTerraformConfig(config, awsExocomputeTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: exocompute,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "exocompute.0.regions.*", "us-east-2"),

				resource.TestCheckResourceAttrPair("polaris_aws_exocompute.default", "account_id", "polaris_aws_account.default", "id"),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "region", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "vpc_id", account.Exocompute.VPCID),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "polaris_managed", "true"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_exocompute.default", "subnets.*", account.Exocompute.Subnets[0].ID),
				resource.TestCheckTypeSetElemAttr("polaris_aws_exocompute.default", "subnets.*", account.Exocompute.Subnets[0].ID),
			),
		}},
	})
}
