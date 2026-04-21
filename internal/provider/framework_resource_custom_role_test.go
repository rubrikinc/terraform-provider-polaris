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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             customRoleCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the resource can be created.
			Config: `
				resource "polaris_custom_role" "role" {
					name        = "Test Auditor"
					description = "Test Role: Delete Me!"

					permission {
						operation = "EXPORT_DATA_CLASS_GLOBAL"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["GlobalResource"]
						}
					}
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyName),
					knownvalue.StringExact("Test Auditor")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Test Role: Delete Me!")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyPermission),
					knownvalue.SetExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOperation: knownvalue.StringExact("EXPORT_DATA_CLASS_GLOBAL"),
							keyHierarchy: knownvalue.SetExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
								keySnappableType: knownvalue.StringExact("AllSubHierarchyType"),
								keyObjectIDs: knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("GlobalResource")}),
							})}),
						}),
					})),
				statecheck.ExpectIdentity("polaris_custom_role.role", map[string]knownvalue.Check{
					keyID: NonNullUUID(),
				}),
				statecheck.ExpectIdentityValueMatchesState("polaris_custom_role.role", tfjsonpath.New(keyID)),
			},
		}, {
			// Verify that the resource can be updated.
			Config: `
				resource "polaris_custom_role" "role" {
					name        = "Test Auditor Update"
					description = "Test Role: Delete Me! Update"

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
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyName),
					knownvalue.StringExact("Test Auditor Update")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Test Role: Delete Me! Update")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyPermission),
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
							keyOperation: knownvalue.StringExact("VIEW_CLUSTER"),
							keyHierarchy: knownvalue.SetExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
								keySnappableType: knownvalue.StringExact("AllSubHierarchyType"),
								keyObjectIDs: knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("CLUSTER_ROOT"),
								}),
							})}),
						}),
					})),
				statecheck.ExpectIdentity("polaris_custom_role.role", map[string]knownvalue.Check{
					keyID: NonNullUUID(),
				}),
				statecheck.ExpectIdentityValueMatchesState("polaris_custom_role.role", tfjsonpath.New(keyID)),
			},
		}, {
			// Verify that the resource can be imported.
			ResourceName:      "polaris_custom_role.role",
			ImportState:       true,
			ImportStateVerify: true,
		}},
	})
}

func TestAccCustomRoleResource_FromTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             customRoleCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the resource can be created from a role template.
			Config: `
				data "polaris_role_template" "auditor" {
				  	name = "Compliance Auditor"
				}
				
				resource "polaris_custom_role" "role" {
					name        = "Test Auditor"
					description = "Based on the ${data.polaris_role_template.auditor.name} template: Delete Me!"
					
					dynamic "permission" {
						for_each = data.polaris_role_template.auditor.permission
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
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyName),
					knownvalue.StringExact("Test Auditor")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Based on the Compliance Auditor template: Delete Me!")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyPermission),
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
			},
		}},
	})
}

// TestAccPolarisCustomRole_FrameworkMigration verifies that existing state
// created by the SDKv2 provider (v1.5.0) can be read by the Framework
// provider without drift. Step 1 creates the resource using the published
// SDKv2 provider; step 2 refreshes state using the local Framework provider
// and asserts the plan is empty.
func TestAccCustomRoleResource_FrameworkMigration(t *testing.T) {
	config := `
		resource "polaris_custom_role" "role" {
			name        = "Test Auditor"
			description = "Test Role: Delete Me!"

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
	`

	resource.Test(t, resource.TestCase{
		CheckDestroy: customRoleCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			ExternalProviders: map[string]resource.ExternalProvider{
				"polaris": {
					Source:            "rubrikinc/polaris",
					VersionConstraint: "1.5.0",
				},
			},
			Config: config,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyName),
					knownvalue.StringExact("Test Auditor")),
				statecheck.ExpectKnownValue("polaris_custom_role.role", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Test Role: Delete Me!")),
			},
		}, {
			ProtoV6ProviderFactories: protoV6ProviderFactories,
			Config:                   config,
			PlanOnly:                 true,
		}},
	})
}
