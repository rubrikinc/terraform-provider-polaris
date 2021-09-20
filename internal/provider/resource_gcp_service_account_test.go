package provider

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const gcpServiceAccountTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_service_account" "default" {
	credentials = "{{ .Resource.Credentials }}"
}
`

func TestAccPolarisGCPServiceAccount_basic(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// When name isn't specified the file name, without extension, is used.
	id := strings.TrimSuffix(filepath.Base(project.Credentials), filepath.Ext(project.Credentials))

	serviceAccount, err := makeTerraformConfig(config, gcpServiceAccountTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			PreConfig: testStepDelay,
			Config:    serviceAccount,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_gcp_service_account.default", "id", id),
				resource.TestCheckResourceAttr("polaris_gcp_service_account.default", "credentials", project.Credentials),
			),
		}},
	})
}
