package provider

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const gcpProjectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials = "{{ .Resource.Credentials }}"
	project     = "{{ .Resource.ProjectID }}"
}
`

const gcpProjectFromValuesTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_service_account" "default" {
	credentials = "{{ .Resource.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	organization_name = "{{ .Resource.OrganizationName }}"
	project           = "{{ .Resource.ProjectID }}"
	project_name      = "{{ .Resource.ProjectName }}"
	project_number    = {{ .Resource.ProjectNumber }}

	depends_on = [polaris_gcp_service_account.default]
}
`

func TestAccPolarisGCPProject_basic(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	projectCredentials, err := makeTerraformConfig(config, gcpProjectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectCredentials,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),
			),
		}},
	})

	projectValues, err := makeTerraformConfig(config, gcpProjectFromValuesTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectValues,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),
			),
		}},
	})
}
