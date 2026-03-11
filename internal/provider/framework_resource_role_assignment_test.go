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

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPolarisRoleAssignment(t *testing.T) {
	createTestUser(t, testUserEmail(t), createTestRoleWithUniqueName(t))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             roleAssignmentCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the resource can be created.
			Config: `
				variable "user_email" {
					type = string
				}

				data "polaris_user" "user" {
					email = var.user_email
				}

				resource "polaris_custom_role" "auditor" {
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
						operation = "VIEW_DATA_CLASS_GLOBAL"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["GlobalResource"]
						}
					}
				}

				resource "polaris_role_assignment" "auditor" {
					user_id  = data.polaris_user.user.id

					role_ids = [
						polaris_custom_role.auditor.id
					]
				}
			`,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.CompareValuePairs(
					"polaris_role_assignment.auditor", tfjsonpath.New(keyID),
					"data.polaris_user.user", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_role_assignment.auditor", tfjsonpath.New(keyUserID),
					"data.polaris_user.user", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("polaris_role_assignment.auditor", tfjsonpath.New(keyRoleIDs),
					knownvalue.SetSizeExact(1)),
				statecheck.CompareValueCollection(
					"polaris_role_assignment.auditor", []tfjsonpath.Path{tfjsonpath.New(keyRoleIDs)},
					"polaris_custom_role.auditor", tfjsonpath.New(keyID),
					compare.ValuesSame()),
			},
		}, {
			// Verify that the resource can be updated.
			Config: `
				variable "user_email" {
					type = string
				}

				data "polaris_user" "user" {
					email = var.user_email
				}

				resource "polaris_custom_role" "auditor" {
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
						operation = "VIEW_DATA_CLASS_GLOBAL"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["GlobalResource"]
						}
					}
				}

				resource "polaris_custom_role" "cluster_viewer" {
					name        = "Test Cluster Viewer"
					description = "Test Role: Delete Me!"

					permission {
						operation = "VIEW_CLUSTER"
						hierarchy {
							snappable_type = "AllSubHierarchyType"
							object_ids     = ["CLUSTER_ROOT"]
						}
					}
				}

				resource "polaris_role_assignment" "auditor" {
					user_id  = data.polaris_user.user.id

					role_ids = [
						polaris_custom_role.auditor.id,
						polaris_custom_role.cluster_viewer.id,
					]
				}
			`,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.CompareValuePairs(
					"polaris_role_assignment.auditor", tfjsonpath.New(keyID),
					"data.polaris_user.user", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_role_assignment.auditor", tfjsonpath.New(keyUserID),
					"data.polaris_user.user", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("polaris_role_assignment.auditor", tfjsonpath.New(keyRoleIDs),
					knownvalue.SetSizeExact(2)),
				statecheck.CompareValueCollection(
					"polaris_role_assignment.auditor", []tfjsonpath.Path{tfjsonpath.New(keyRoleIDs)},
					"polaris_custom_role.auditor", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValueCollection(
					"polaris_role_assignment.auditor", []tfjsonpath.Path{tfjsonpath.New(keyRoleIDs)},
					"polaris_custom_role.cluster_viewer", tfjsonpath.New(keyID),
					compare.ValuesSame()),
			},
		}, {
			// Verify that the resource can be imported. Note, the import will
			// import all roles assigned to the user.
			ResourceName:            "polaris_role_assignment.auditor",
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{keyUserEmail, keyRoleIDs},
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
		}},
	})
}

// TestAccPolarisRoleAssignment_FrameworkMigration verifies that existing state
// created by the SDKv2 provider (v1.5.0) can be read by the Framework
// provider without drift.
func TestAccPolarisRoleAssignment_FrameworkMigration(t *testing.T) {
	createTestUser(t, testUserEmail(t), createTestRoleWithUniqueName(t))

	// Test 1: Modern fields (user_id + role_ids).
	tfConfig := `
		variable "user_email" {
			type = string
		}

		data "polaris_user" "user" {
			email = var.user_email
		}

		resource "polaris_custom_role" "auditor" {
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
				operation = "VIEW_DATA_CLASS_GLOBAL"
				hierarchy {
					snappable_type = "AllSubHierarchyType"
					object_ids     = ["GlobalResource"]
				}
			}
		}

		resource "polaris_role_assignment" "auditor" {
			user_id  = data.polaris_user.user.id

			role_ids = [
				polaris_custom_role.auditor.id,
			]
		}
	`

	resource.Test(t, resource.TestCase{
		CheckDestroy: roleAssignmentCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			ExternalProviders: map[string]resource.ExternalProvider{
				"polaris": {
					Source:            "rubrikinc/polaris",
					VersionConstraint: "1.5.0",
				},
			},
			Config: tfConfig,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
		}, {
			ProtoV6ProviderFactories: protoV6ProviderFactories,
			Config:                   tfConfig,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			PlanOnly: true,
		}},
	})

	// Test 2: Deprecated fields (user_email + role_id).
	tfConfig = `
		variable "user_email" {
			type = string
		}

		resource "polaris_custom_role" "auditor" {
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
				operation = "VIEW_DATA_CLASS_GLOBAL"
				hierarchy {
					snappable_type = "AllSubHierarchyType"
					object_ids     = ["GlobalResource"]
				}
			}
		}

		resource "polaris_role_assignment" "auditor" {
			user_email = var.user_email
			role_id    = polaris_custom_role.auditor.id
		}
	`

	resource.Test(t, resource.TestCase{
		CheckDestroy: roleAssignmentCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			ExternalProviders: map[string]resource.ExternalProvider{
				"polaris": {
					Source:            "rubrikinc/polaris",
					VersionConstraint: "1.5.0",
				},
			},
			Config: tfConfig,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
		}, {
			ProtoV6ProviderFactories: protoV6ProviderFactories,
			Config:                   tfConfig,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			PlanOnly: true,
		}},
	})
}
