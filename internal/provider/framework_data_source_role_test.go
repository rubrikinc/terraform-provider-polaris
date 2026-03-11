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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRoleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             customRoleCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the data source can look up the role by ID and name.
			Config: `
				resource "polaris_custom_role" "role" {
					name        = "Terraform Test Role"
					description = "Terraform Integration Test Role"

					permission {
						operation = "EXPORT_DATA_CLASS_GLOBAL"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["GlobalResource"]
						}
					}
					permission {
						operation = "VIEW_CLUSTER"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["CLUSTER_ROOT"]
						}
					}
				}

				data "polaris_role" "by_id" {
					role_id = polaris_custom_role.role.id
				}

				data "polaris_role" "by_name" {
					name = polaris_custom_role.role.name
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				// Role.
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyID),
					NonNullUUID()),
				// By ID.
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyID),
					"data.polaris_role.by_id", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyID),
					"data.polaris_role.by_id", tfjsonpath.New(keyRoleID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyName),
					"data.polaris_role.by_id", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyDescription),
					"data.polaris_role.by_id", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyPermission),
					"data.polaris_role.by_id", tfjsonpath.New(keyPermission),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_role.by_id", tfjsonpath.New(keyIsOrgAdmin),
					knownvalue.Bool(false)),
				// By Name.
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyID),
					"data.polaris_role.by_name", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyID),
					"data.polaris_role.by_name", tfjsonpath.New(keyRoleID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyName),
					"data.polaris_role.by_name", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyDescription),
					"data.polaris_role.by_name", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_custom_role.role", tfjsonpath.New(keyPermission),
					"data.polaris_role.by_name", tfjsonpath.New(keyPermission),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_role.by_name", tfjsonpath.New(keyIsOrgAdmin),
					knownvalue.Bool(false)),
			},
		}},
	})
}

// TestAccRoleDataSource_FrameworkMigration verifies that the migrated role data
// source is backwards compatible with the SDKv2 provider.
func TestAccRoleDataSource_FrameworkMigration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"polaris-sdkv2": {
				Source:            "rubrikinc/polaris",
				VersionConstraint: "1.5.0",
			},
		},
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             customRoleCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the two data sources are equal.
			Config: `
				resource "polaris_custom_role" "role" {
					name        = "Terraform Migration Role"
					description = "Terraform Integration Migration Role"

					permission {
						operation = "VIEW_CLUSTER"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["CLUSTER_ROOT"]
						}
					}
					permission {
						operation = "EXPORT_DATA_CLASS_GLOBAL"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["GlobalResource"]
						}
					}
				}

				data "polaris_role" "old" {
					provider = polaris-sdkv2

					role_id = polaris_custom_role.role.id
				}

				data "polaris_role" "new" {
					role_id = polaris_custom_role.role.id
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyID),
					"data.polaris_role.new", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyRoleID),
					"data.polaris_role.new", tfjsonpath.New(keyRoleID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyName),
					"data.polaris_role.new", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyDescription),
					"data.polaris_role.new", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyIsOrgAdmin),
					"data.polaris_role.new", tfjsonpath.New(keyIsOrgAdmin),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role.old", tfjsonpath.New(keyPermission),
					"data.polaris_role.new", tfjsonpath.New(keyPermission),
					compare.ValuesSame()),
			},
		}},
	})
}
