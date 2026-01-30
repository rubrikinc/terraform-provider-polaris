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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const gcpProjectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	cloud_native_protection {
	}
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

	cloud_native_protection {
	}

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
				// Project resource
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "cloud_native_protection.0.status", "connected"),
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
				// Project resource
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "cloud_native_protection.0.status", "connected"),
			),
		}},
	})
}

const gcpProjectUsingFeatureTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_NATIVE_PROTECTION"
		permission_groups = [
			"BASIC",
			"EXPORT_AND_RESTORE",
			"FILE_LEVEL_RECOVERY",
		]
	}

	feature {
		name = "GCP_SHARED_VPC_HOST"
		permission_groups = [
			"BASIC",
		]
	}
}
`

func TestAccPolarisGCPProject_feature(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	projectUsingFeature, err := makeTerraformConfig(config, gcpProjectUsingFeatureTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectUsingFeature,
			Check: resource.ComposeTestCheckFunc(
				// Project resource.
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "BASIC"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "EXPORT_AND_RESTORE"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "FILE_LEVEL_RECOVERY"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_NATIVE_PROTECTION",
					"permissions":         "",
					"permission_groups.#": "3",
					"status":              "CONNECTED",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "GCP_SHARED_VPC_HOST",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "CONNECTED",
				}),
			),
		}},
	})
}
