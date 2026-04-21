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

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccUserListResource(t *testing.T) {
	userID := createTestUser(t, testUserEmail(t), createTestRole(t, "Test Search User Role"))

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Query: true,
			Config: `
				provider "polaris" {}

				list "polaris_user" "all" {
					provider = polaris
				}
			`,
			QueryResultChecks: []querycheck.QueryResultCheck{
				querycheck.ExpectIdentity("polaris_user.all", map[string]knownvalue.Check{
					keyID: knownvalue.StringExact(userID),
				}),
			},
		}, {
			Query: true,
			Config: `
				variable "user_email" {
					type = string
				}

				provider "polaris" {}

				list "polaris_user" "filtered" {
					provider = polaris

					config {
						email = var.user_email
					}
				}
			`,
			ConfigVariables: config.Variables{
				"user_email": config.StringVariable(testUserEmail(t)),
			},
			QueryResultChecks: []querycheck.QueryResultCheck{
				querycheck.ExpectIdentity("polaris_user.filtered", map[string]knownvalue.Check{
					keyID: knownvalue.StringExact(userID),
				}),
				querycheck.ExpectLength("polaris_user.filtered", 1),
			},
		}},
	})
}
