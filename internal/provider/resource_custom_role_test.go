// Copyright 2023 Rubrik, Inc.
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

const customRoleViewClusterTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_custom_role" "default" {
	name        = "View Cluster Role"
    description = "View Cluster Role Description"

	permission {
		operation = "VIEW_CLUSTER"
		hierarchy {
			snappable_type = "AllSubHierarchyType"
			object_ids     = ["CLUSTER_ROOT"]
		}
	}
}
`

const customRoleExportDataTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_custom_role" "default" {
	name        = "Export Data Role"
    description = "Export Data Role Description"

	permission {
		operation = "EXPORT_DATA_CLASS_GLOBAL"
		hierarchy {
			snappable_type = "AllSubHierarchyType"
			object_ids     = ["GlobalResource"]
		}
  	}
}
`

func TestAccPolarisCustomRole_basic(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	customViewClusterRole, err := makeTerraformConfig(config, customRoleViewClusterTmpl)
	if err != nil {
		t.Fatal(err)
	}
	customExportDataRole, err := makeTerraformConfig(config, customRoleExportDataTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: customViewClusterRole,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_custom_role.default", "name", "View Cluster Role"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "description", "View Cluster Role Description"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "permission.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_custom_role.default", "permission.*", map[string]string{"operation": "VIEW_CLUSTER"}),
			),
		}, {
			Config: customExportDataRole,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_custom_role.default", "name", "Export Data Role"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "description", "Export Data Role Description"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "permission.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_custom_role.default", "permission.*", map[string]string{"operation": "EXPORT_DATA_CLASS_GLOBAL"}),
			),
		}},
	})
}

const customRoleFromTemplateTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

data "polaris_role_template" "compliance_auditor" {
  name = "Compliance Auditor"
}

resource "polaris_custom_role" "default" {
  name        = "Compliance Auditor Role"
  description = "Based on the ${data.polaris_role_template.compliance_auditor.name} template"

  dynamic "permission" {
    for_each = data.polaris_role_template.compliance_auditor.permission
    content {
      operation = permission.value["operation"]

      dynamic "hierarchy" {
        for_each = permission.value["hierarchy"]
        content {
          snappable_type = hierarchy.value["snappable_type"]
          object_ids     = hierarchy.value["object_ids"]
        }
      }
    }
  }
}
`

func TestAccPolarisCustomRoleFromTemplate_basic(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	customRoleFromTemplate, err := makeTerraformConfig(config, customRoleFromTemplateTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: customRoleFromTemplate,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_custom_role.default", "name", "Compliance Auditor Role"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "description", "Based on the Compliance Auditor template"),
				resource.TestCheckResourceAttr("polaris_custom_role.default", "permission.#", "2"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_custom_role.default", "permission.*", map[string]string{"operation": "EXPORT_DATA_CLASS_GLOBAL"}),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_custom_role.default", "permission.*", map[string]string{"operation": "VIEW_DATA_CLASS_GLOBAL"}),
			),
		}},
	})
}
