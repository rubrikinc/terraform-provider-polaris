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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSecurityPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             dataSecurityPolicyCheckDestroy(t.Context()),
		Steps: []resource.TestStep{{
			// Verify that the resource can be created with a simple filter.
			Config: `
				resource "polaris_data_security_policy" "test" {
					name        = "Terraform Test Policy"
					description = "Acceptance test: Delete Me!"
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
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyName),
					knownvalue.StringExact("Terraform Test Policy")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Acceptance test: Delete Me!")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyCategory),
					knownvalue.StringExact("OVEREXPOSED")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keySeverity),
					knownvalue.StringExact("MEDIUM")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyEnabled),
					knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyPredefined),
					knownvalue.Bool(false)),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyFilter),
					knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOp: knownvalue.StringExact("AND"),
							keyCondition: knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									keyFilterType:   knownvalue.StringExact("SECURITY_DOCUMENT_SENSITIVITY"),
									keyRelationship: knownvalue.StringExact("IS"),
									keyValues: knownvalue.ListExact([]knownvalue.Check{
										knownvalue.StringExact("HIGH"),
									}),
								}),
							}),
							keyGroup: knownvalue.ListExact([]knownvalue.Check{}),
						}),
					})),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyThresholdFilter),
					knownvalue.ListExact([]knownvalue.Check{})),
			},
		}, {
			// Verify that the resource can be updated with a nested group and
			// threshold filter.
			Config: `
				resource "polaris_data_security_policy" "test" {
					name        = "Terraform Test Policy Updated"
					description = "Acceptance test updated: Delete Me!"
					category    = "OVEREXPOSED"
					severity    = "HIGH"

					filter {
						op = "AND"
						condition {
							filter_type  = "SECURITY_DOCUMENT_SENSITIVITY"
							relationship = "IS"
							values       = ["HIGH"]
						}
						group {
							op = "OR"
							condition {
								filter_type  = "SECURITY_DOCUMENT_SENSITIVITY"
								relationship = "IS"
								values       = ["MEDIUM"]
							}
						}
					}

					threshold_filter {
						op = "AND"
						condition {
							filter_type  = "SECURITY_DOCUMENT_HIT_COUNT"
							relationship = "GREATER_THAN"
							values       = ["5"]
						}
					}
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyID),
					NonNullUUID()),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyName),
					knownvalue.StringExact("Terraform Test Policy Updated")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyDescription),
					knownvalue.StringExact("Acceptance test updated: Delete Me!")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keySeverity),
					knownvalue.StringExact("HIGH")),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyFilter),
					knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOp: knownvalue.StringExact("AND"),
							keyCondition: knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									keyFilterType:   knownvalue.StringExact("SECURITY_DOCUMENT_SENSITIVITY"),
									keyRelationship: knownvalue.StringExact("IS"),
									keyValues: knownvalue.ListExact([]knownvalue.Check{
										knownvalue.StringExact("HIGH"),
									}),
								}),
							}),
							keyGroup: knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									keyOp: knownvalue.StringExact("OR"),
									keyCondition: knownvalue.ListExact([]knownvalue.Check{
										knownvalue.ObjectExact(map[string]knownvalue.Check{
											keyFilterType:   knownvalue.StringExact("SECURITY_DOCUMENT_SENSITIVITY"),
											keyRelationship: knownvalue.StringExact("IS"),
											keyValues: knownvalue.ListExact([]knownvalue.Check{
												knownvalue.StringExact("MEDIUM"),
											}),
										}),
									}),
								}),
							}),
						}),
					})),
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyThresholdFilter),
					knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							keyOp: knownvalue.StringExact("AND"),
							keyCondition: knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									keyFilterType:   knownvalue.StringExact("SECURITY_DOCUMENT_HIT_COUNT"),
									keyRelationship: knownvalue.StringExact("GREATER_THAN"),
									keyValues: knownvalue.ListExact([]knownvalue.Check{
										knownvalue.StringExact("5"),
									}),
								}),
							}),
							keyGroup: knownvalue.ListExact([]knownvalue.Check{}),
						}),
					})),
			},
		}, {
			// Verify that the threshold filter can be removed.
			Config: `
				resource "polaris_data_security_policy" "test" {
					name        = "Terraform Test Policy Updated"
					description = "Acceptance test updated: Delete Me!"
					category    = "OVEREXPOSED"
					severity    = "HIGH"

					filter {
						op = "AND"
						condition {
							filter_type  = "SECURITY_DOCUMENT_SENSITIVITY"
							relationship = "IS"
							values       = ["HIGH"]
						}
						group {
							op = "OR"
							condition {
								filter_type  = "SECURITY_DOCUMENT_SENSITIVITY"
								relationship = "IS"
								values       = ["MEDIUM"]
							}
						}
					}
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("polaris_data_security_policy.test", tfjsonpath.New(keyThresholdFilter),
					knownvalue.ListExact([]knownvalue.Check{})),
			},
		}, {
			// Verify that the resource can be imported.
			ResourceName:      "polaris_data_security_policy.test",
			ImportState:       true,
			ImportStateVerify: true,
		}},
	})
}
