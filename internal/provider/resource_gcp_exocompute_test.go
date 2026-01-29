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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var gcpExocomputeTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "EXOCOMPUTE"
		permission_groups = [
			"BASIC"
		]
	}

	# We ignore changes to the organization name since it's tied to the
	# CLOUD_NATIVE_PROTECTION feature, which isn't onboarded.'
	lifecycle {
		ignore_changes = [
			organization_name,
		]
	}
}

resource "polaris_gcp_exocompute" "default" {
    cloud_account_id = polaris_gcp_project.default.id

    regional_config {
	    region      = "{{ .Resource.Exocompute.Region }}"
		subnet_name = "{{ .Resource.Exocompute.SubnetName }}"
		vpc_name    = "{{ .Resource.Exocompute.VPCName }}"
    }
}
`

func TestAccPolarisGCPExocompute_basic(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	exocompute, err := makeTerraformConfig(config, gcpExocomputeTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: exocompute,
			Check: resource.ComposeTestCheckFunc(
				// Project resource.
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "BASIC"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "EXOCOMPUTE",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "CONNECTED",
				}),

				// Exocompute resource.
				resource.TestCheckResourceAttrPair("polaris_gcp_exocompute.default", "cloud_account_id", "polaris_gcp_project.default", "id"),
				resource.TestCheckResourceAttr("polaris_gcp_exocompute.default", "regional_config.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_exocompute.default", "regional_config.*", map[string]string{
					"%":           "3",
					"region":      project.Exocompute.Region,
					"subnet_name": project.Exocompute.SubnetName,
					"vpc_name":    project.Exocompute.VPCName,
				}),
			),
		}},
	})
}
