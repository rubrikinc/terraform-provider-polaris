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

func TestAccRoleTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// Verify that the data source can look up the role template by
			// name.
			Config: `
				data "polaris_role_template" "by_id" {
					role_template_id = data.polaris_role_template.by_name.role_template_id
				}

				data "polaris_role_template" "by_name" {
					name = "Compliance Auditor"
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				// By ID.
				statecheck.ExpectKnownValue("data.polaris_role_template.by_id", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyID),
					"data.polaris_role_template.by_id", tfjsonpath.New(keyRoleTemplateID),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_role_template.by_id", tfjsonpath.New(keyName),
					knownvalue.StringExact("Compliance Auditor")),
				statecheck.ExpectKnownValue("data.polaris_role_template.by_id", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Template for compliance auditor")),
				statecheck.ExpectKnownValue("data.polaris_role_template.by_id", tfjsonpath.New(keyPermission),
					knownvalue.SetExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOperation: knownvalue.StringExact("EXPORT_DATA_CLASS_GLOBAL"),
							keyHierarchy: knownvalue.SetExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
								keySnappableType: knownvalue.StringExact("AllSubHierarchyType"),
								keyObjectIDs: knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("GlobalResource")}),
							})}),
						}),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOperation: knownvalue.StringExact("VIEW_DATA_CLASS_GLOBAL"),
							keyHierarchy: knownvalue.SetExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
								keySnappableType: knownvalue.StringExact("AllSubHierarchyType"),
								keyObjectIDs: knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("GlobalResource"),
								}),
							})}),
						}),
					})),
				// By Name.
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyID),
					"data.polaris_role_template.by_name", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyRoleTemplateID),
					"data.polaris_role_template.by_name", tfjsonpath.New(keyRoleTemplateID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyName),
					"data.polaris_role_template.by_name", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyDescription),
					"data.polaris_role_template.by_name", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.by_id", tfjsonpath.New(keyPermission),
					"data.polaris_role_template.by_name", tfjsonpath.New(keyPermission),
					compare.ValuesSame()),
			},
		}},
	})
}

// TestAccRoleTemplateDataSource_FrameworkMigration verifies that the migrated
// role template data source is backwards compatible with the SDKv2 provider.
func TestAccRoleTemplateDataSource_FrameworkMigration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"polaris-sdkv2": {
				Source:            "rubrikinc/polaris",
				VersionConstraint: "1.5.0",
			},
		},
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// Verify that the two data sources are equal.
			Config: `
				data "polaris_role_template" "old" {
					provider = polaris-sdkv2

					name = "Compliance Auditor"
				}

				data "polaris_role_template" "new" {
					name = "Compliance Auditor"
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("data.polaris_role_template.new", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.old", tfjsonpath.New(keyID),
					"data.polaris_role_template.new", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.old", tfjsonpath.New(keyRoleTemplateID),
					"data.polaris_role_template.new", tfjsonpath.New(keyRoleTemplateID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.old", tfjsonpath.New(keyName),
					"data.polaris_role_template.new", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.old", tfjsonpath.New(keyDescription),
					"data.polaris_role_template.new", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_role_template.old", tfjsonpath.New(keyPermission),
					"data.polaris_role_template.new", tfjsonpath.New(keyPermission),
					compare.ValuesSame()),
			},
		}},
	})
}
