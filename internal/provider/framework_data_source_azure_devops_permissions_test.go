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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAzureDevOpsPermissionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
					data "polaris_azure_devops_permissions" "basic" {
						feature           = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
						permission_groups = ["BASIC"]
					}
				`,
			ConfigStateChecks: []statecheck.StateCheck{
				// id is a SHA-256 hex digest.
				statecheck.ExpectKnownValue("data.polaris_azure_devops_permissions.basic", tfjsonpath.New(keyID),
					knownvalue.StringRegexp(sha256Hex)),

				// Input feature is echoed back on the state.
				statecheck.ExpectKnownValue("data.polaris_azure_devops_permissions.basic", tfjsonpath.New(keyFeature),
					knownvalue.StringExact("AZURE_DEVOPS_REPOSITORY_PROTECTION")),

				// The requested permission group has a version recorded.
				statecheck.ExpectKnownValue("data.polaris_azure_devops_permissions.basic", tfjsonpath.New(keyPermissionGroupVersions),
					knownvalue.MapPartial(map[string]knownvalue.Check{
						"BASIC": knownvalue.NotNull(),
					})),

				// permissions is the permission document canonicalized by the SDK.
				// Asserting just that it is a JSON array of objects carrying a
				// permission field, without pinning the specific permissions,
				// which the RSC catalog evolves over time. This fails if the RSC
				// permission JSON format drifts away from what sortPermissionJSON
				// in the SDK can order.
				statecheck.ExpectKnownValue("data.polaris_azure_devops_permissions.basic", tfjsonpath.New(keyPermissions),
					knownvalue.StringRegexp(regexp.MustCompile(`^\[\{.*"permission":.*\}\]$`))),
			},
		}},
	})
}
