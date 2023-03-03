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

const roleAssignmentTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_custom_role" "view_cluster" {
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

resource "polaris_role_assignment" "user_view_cluster" {
  role_id    = polaris_custom_role.view_cluster.id
  user_email = "{{ .Resource.ExistingUserEmail }}"
}
`

func TestAccPolarisRoleAssignment_basic(t *testing.T) {
	config, rscConfig, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	roleAssignment, err := makeTerraformConfig(config, roleAssignmentTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: roleAssignment,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair("polaris_role_assignment.user_view_cluster", "role_id", "polaris_custom_role.view_cluster", "id"),
				resource.TestCheckResourceAttr("polaris_role_assignment.user_view_cluster", "user_email", rscConfig.ExistingUserEmail),
			),
		}},
	})
}
