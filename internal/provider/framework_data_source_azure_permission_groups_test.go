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

func TestAccAzurePermissionGroupsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// Single-feature lookup, plus a parallel count-based lookup over
			// two features so we can verify that the data source returns the
			// same data when invoked indirectly. count is used here instead of
			// for_each because for_each on data sources interacts badly with
			// the testing framework.
			Config: `
					locals {
						features = ["CLOUD_NATIVE_PROTECTION", "EXOCOMPUTE"]
					}

					data "polaris_azure_permission_groups" "cnp" {
						feature = "CLOUD_NATIVE_PROTECTION"
					}

					data "polaris_azure_permission_groups" "all" {
						count   = length(local.features)
						feature = local.features[count.index]
					}
				`,
			ConfigStateChecks: []statecheck.StateCheck{
				// id is a SHA-256 hex digest.
				statecheck.ExpectKnownValue("data.polaris_azure_permission_groups.cnp", tfjsonpath.New(keyID),
					knownvalue.StringRegexp(sha256Hex)),

				// Input feature is echoed back on the state.
				statecheck.ExpectKnownValue("data.polaris_azure_permission_groups.cnp", tfjsonpath.New(keyFeature),
					knownvalue.StringExact("CLOUD_NATIVE_PROTECTION")),

				// CLOUD_NATIVE_PROTECTION always exposes the BASIC permission
				// group with at least one statement at the subscription scope.
				// Asserting just the shape (group name, non-null version, and
				// one (scope=subscription, kind=action) statement with a
				// non-null permission) without pinning specific permission
				// strings or use cases, which the RSC catalog evolves over
				// time.
				statecheck.ExpectKnownValue("data.polaris_azure_permission_groups.cnp", tfjsonpath.New(keyPermissionGroups),
					knownvalue.SetPartial([]knownvalue.Check{
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							keyName:    knownvalue.StringExact("BASIC"),
							keyVersion: knownvalue.NotNull(),
							keyStatements: knownvalue.SetPartial([]knownvalue.Check{
								knownvalue.ObjectPartial(map[string]knownvalue.Check{
									keyScope:      knownvalue.StringExact("subscription"),
									keyKind:       knownvalue.StringExact("action"),
									keyPermission: knownvalue.NotNull(),
									keyUseCase:    knownvalue.NotNull(),
								}),
							}),
						}),
					})),

				// Counted lookup for CLOUD_NATIVE_PROTECTION (index 0) returns
				// identical data to the directly-targeted lookup.
				statecheck.CompareValuePairs(
					"data.polaris_azure_permission_groups.cnp", tfjsonpath.New(keyID),
					"data.polaris_azure_permission_groups.all[0]", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"data.polaris_azure_permission_groups.cnp", tfjsonpath.New(keyPermissionGroups),
					"data.polaris_azure_permission_groups.all[0]", tfjsonpath.New(keyPermissionGroups),
					compare.ValuesSame()),

				// EXOCOMPUTE (index 1) produces a distinct id, confirming the
				// data source is keyed on the input feature.
				statecheck.CompareValuePairs(
					"data.polaris_azure_permission_groups.all[0]", tfjsonpath.New(keyID),
					"data.polaris_azure_permission_groups.all[1]", tfjsonpath.New(keyID),
					compare.ValuesDiffer()),
			},
		}},
	})
}
