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

func TestAccSSOGroupDataSource(t *testing.T) {
	// Check if the test SSO group is available, if not, the test is skipped.
	checkTestSSOGroup(t, testSSOGroupName(t))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// Verify that the data source can look up the SSO group
			// by ID and name.
			Config: `
				variable "sso_group_name" {
					type = string
				}

				data "polaris_sso_group" "by_name" {
					name = var.sso_group_name
				}

				data "polaris_sso_group" "by_id" {
					sso_group_id = data.polaris_sso_group.by_name.id
				}
			`,
			ConfigVariables: config.Variables{
				"sso_group_name": config.StringVariable(testSSOGroupName(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				// By Name.
				statecheck.ExpectKnownValue("data.polaris_sso_group.by_name", tfjsonpath.New(keyID),
					knownvalue.NotNull()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyID),
					"data.polaris_sso_group.by_name", tfjsonpath.New(keySSOGroupID),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_sso_group.by_name", tfjsonpath.New(keyDomainName),
					knownvalue.NotNull()),
				// By ID.
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyID),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keySSOGroupID),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keySSOGroupID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyName),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyDomainName),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keyDomainName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyRoles),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keyRoles),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.by_name", tfjsonpath.New(keyUsers),
					"data.polaris_sso_group.by_id", tfjsonpath.New(keyUsers),
					compare.ValuesSame()),
			},
		}},
	})
}

// TestAccSSOGroupDataSource_FrameworkMigration verifies that the migrated SSO
// group data source is backwards compatible with the SDKv2 provider.
func TestAccSSOGroupDataSource_FrameworkMigration(t *testing.T) {
	// Check if the test SSO group is available, if not, the test is skipped.
	checkTestSSOGroup(t, testSSOGroupName(t))

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
				variable "sso_group_name" {
					type = string
				}

				data "polaris_sso_group" "old" {
					provider = polaris-sdkv2

					name = var.sso_group_name
				}

				data "polaris_sso_group" "new" {
					name = var.sso_group_name
				}
			`,
			ConfigVariables: config.Variables{
				"sso_group_name": config.StringVariable(testSSOGroupName(t)),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("data.polaris_sso_group.new", tfjsonpath.New(keyID),
					knownvalue.NotNull()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keyID),
					"data.polaris_sso_group.new", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keySSOGroupID),
					"data.polaris_sso_group.new", tfjsonpath.New(keySSOGroupID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keyName),
					"data.polaris_sso_group.new", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keyDomainName),
					"data.polaris_sso_group.new", tfjsonpath.New(keyDomainName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keyRoles),
					"data.polaris_sso_group.new", tfjsonpath.New(keyRoles),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_sso_group.old", tfjsonpath.New(keyUsers),
					"data.polaris_sso_group.new", tfjsonpath.New(keyUsers),
					compare.ValuesSame()),
			},
		}},
	})
}
