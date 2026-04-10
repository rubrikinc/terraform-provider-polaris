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

func TestAccDataSecurityPolicyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             dataSecurityPolicyCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the data source can look up the policy by ID and
			// name.
			Config: `
				resource "polaris_data_security_policy" "test" {
					name        = "Terraform Test DS Policy"
					description = "Data source acceptance test: Delete Me!"
					category    = "OVEREXPOSED"
					severity    = "MEDIUM"

					filter {
						op = "AND"
						condition {
							filter_type  = "SECURITY_DOCUMENT_SENSITIVITY"
							relationship = "IS"
							values       = ["HIGH"]
						}
					}
				}

				data "polaris_data_security_policy" "by_id" {
					policy_id = polaris_data_security_policy.test.id
				}

				data "polaris_data_security_policy" "by_name" {
					name = polaris_data_security_policy.test.name
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				// Resource.
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyID),
					NonNullUUID()),

				// By ID.
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyID),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyID),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyPolicyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyName),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyDescription),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyCategory),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyCategory),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keySeverity),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keySeverity),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_data_security_policy.by_id", tfjsonpath.New(keyEnabled),
					knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("data.polaris_data_security_policy.by_id", tfjsonpath.New(keyPredefined),
					knownvalue.Bool(false)),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyFilter),
					"data.polaris_data_security_policy.by_id", tfjsonpath.New(keyFilter),
					compare.ValuesSame()),

				// By Name.
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyID),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keyID),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyName),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keyName),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyDescription),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keyDescription),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyCategory),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keyCategory),
					compare.ValuesSame()),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keySeverity),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keySeverity),
					compare.ValuesSame()),
				statecheck.ExpectKnownValue("data.polaris_data_security_policy.by_name", tfjsonpath.New(keyEnabled),
					knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("data.polaris_data_security_policy.by_name", tfjsonpath.New(keyPredefined),
					knownvalue.Bool(false)),
				statecheck.CompareValuePairs(
					"polaris_data_security_policy.test", tfjsonpath.New(keyFilter),
					"data.polaris_data_security_policy.by_name", tfjsonpath.New(keyFilter),
					compare.ValuesSame()),
			},
		}},
	})
}
