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
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserDataSource(t *testing.T) {
	roleID := createTestRoleWithUniqueName(t)
	userID := createTestUser(t, testUserEmail(t), roleID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// Verify that the data source can look up the user by ID and email.
			Config: `
				variable "user_email" {
					type = string
				}

				data "polaris_user" "by_email" {
					email  = var.user_email
					domain = "LOCAL"
				}

				data "polaris_user" "by_id" {
					user_id = data.polaris_user.by_email.id
				}
			`,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				// By Email.
				statecheck.ExpectKnownValue("data.polaris_user.by_email", tfjsonpath.New(keyID),
					knownvalue.StringExact(userID)),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyID),
					"data.polaris_user.by_email", tfjsonpath.New(keyUserID),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_user.by_email", tfjsonpath.New(keyDomain),
					knownvalue.StringExact("LOCAL")),
				statecheck.ExpectKnownValue("data.polaris_user.by_email", tfjsonpath.New(keyIsAccountOwner),
					knownvalue.Bool(false)),
				statecheck.ExpectKnownValue("data.polaris_user.by_email", tfjsonpath.New(keyStatus),
					knownvalue.StringExact("ACTIVE")),
				statecheck.ExpectKnownValue("data.polaris_user.by_email", tfjsonpath.New(keyRoles),
					knownvalue.SetExact([]knownvalue.Check{
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							keyID: knownvalue.StringExact(roleID.String()),
						}),
					})),
				// By ID.
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyID),
					"data.polaris_user.by_id", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyUserID),
					"data.polaris_user.by_id", tfjsonpath.New(keyUserID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyEmail),
					"data.polaris_user.by_id", tfjsonpath.New(keyEmail),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyDomain),
					"data.polaris_user.by_id", tfjsonpath.New(keyDomain),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyIsAccountOwner),
					"data.polaris_user.by_id", tfjsonpath.New(keyIsAccountOwner),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyRoles),
					"data.polaris_user.by_id", tfjsonpath.New(keyRoles),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.by_email", tfjsonpath.New(keyStatus),
					"data.polaris_user.by_id", tfjsonpath.New(keyStatus),
					compare.ValuesSame()),
			},
		}},
	})
}

// TestAccUserDataSource_FrameworkMigration verifies that the migrated user data
// source is backwards compatible with the SDKv2 provider.
func TestAccUserDataSource_FrameworkMigration(t *testing.T) {
	userID := createTestUser(t, testUserEmail(t), createTestRoleWithUniqueName(t))

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
				variable "user_email" {
					type = string
				}

				data "polaris_user" "old" {
					provider = polaris-sdkv2

					email  = var.user_email
					domain = "LOCAL"
				}

				data "polaris_user" "new" {
					email  = var.user_email
					domain = "LOCAL"
				}
			`,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("data.polaris_user.new", tfjsonpath.New(keyID),
					knownvalue.StringExact(userID)),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyID),
					"data.polaris_user.new", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyUserID),
					"data.polaris_user.new", tfjsonpath.New(keyUserID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyEmail),
					"data.polaris_user.new", tfjsonpath.New(keyEmail),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyDomain),
					"data.polaris_user.new", tfjsonpath.New(keyDomain),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyIsAccountOwner),
					"data.polaris_user.new", tfjsonpath.New(keyIsAccountOwner),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyRoles),
					"data.polaris_user.new", tfjsonpath.New(keyRoles),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_user.old", tfjsonpath.New(keyStatus),
					"data.polaris_user.new", tfjsonpath.New(keyStatus),
					compare.ValuesSame()),
			},
		}},
	})
}
