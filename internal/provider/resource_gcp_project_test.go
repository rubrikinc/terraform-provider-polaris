// Copyright 2021 Rubrik, Inc.
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
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const gcpProjectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	cloud_native_protection {
	}
}
`

const gcpProjectFromValuesTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_service_account" "default" {
	credentials = "{{ .Resource.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	organization_name = "{{ .Resource.OrganizationName }}"
	project           = "{{ .Resource.ProjectID }}"
	project_name      = "{{ .Resource.ProjectName }}"
	project_number    = {{ .Resource.ProjectNumber }}

	cloud_native_protection {
	}

	depends_on = [polaris_gcp_service_account.default]
}
`

func TestAccPolarisGCPProject_basic(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	projectCredentials, err := makeTerraformConfig(config, gcpProjectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectCredentials,
			Check: resource.ComposeTestCheckFunc(
				// Project resource
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "cloud_native_protection.0.status", "connected"),
			),
		}},
	})

	projectValues, err := makeTerraformConfig(config, gcpProjectFromValuesTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectValues,
			Check: resource.ComposeTestCheckFunc(
				// Project resource
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "cloud_native_protection.0.status", "connected"),
			),
		}},
	})
}

const gcpProjectUsingFeatureTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_NATIVE_PROTECTION"
		permission_groups = [
			"BASIC",
			"EXPORT_AND_RESTORE",
			"FILE_LEVEL_RECOVERY",
		]
	}

	feature {
		name = "GCP_SHARED_VPC_HOST"
		permission_groups = [
			"BASIC",
		]
	}
}
`

func TestAccPolarisGCPProject_feature(t *testing.T) {
	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	projectUsingFeature, err := makeTerraformConfig(config, gcpProjectUsingFeatureTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectUsingFeature,
			Check: resource.ComposeTestCheckFunc(
				// Project resource.
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "credentials", project.Credentials),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "organization_name", project.OrganizationName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "BASIC"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "EXPORT_AND_RESTORE"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "FILE_LEVEL_RECOVERY"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_NATIVE_PROTECTION",
					"permissions":         "",
					"permission_groups.#": "3",
					"status":              "CONNECTED",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "GCP_SHARED_VPC_HOST",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "CONNECTED",
				}),
			),
		}},
	})
}

const gcpProjectCloudSQLTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_SQL_PROTECTION"
		permission_groups = [
			"BASIC",
			"EXPORT_AND_RESTORE",
		]
	}

	feature {
		name = "EXOCOMPUTE"
		permission_groups = [
			"BASIC",
			"CLOUDSQL",
		]
	}

	feature {
		name = "GCP_SHARED_VPC_HOST"
		permission_groups = [
			"BASIC",
			"CLOUDSQL",
		]
	}
}
`

// requireCloudSQLFeatureFlag skips the test if the Cloud SQL protection feature
// is not enabled for the account.
func requireCloudSQLFeatureFlag(t *testing.T) {
	t.Helper()

	credentials := os.Getenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE")
	if credentials == "" {
		t.Skip("RUBRIK_POLARIS_SERVICEACCOUNT_FILE not set")
	}

	ctx := context.Background()
	c, err := newClient(ctx, credentials, polaris.CacheParams{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	const cloudSQL = core.FeatureFlagName("CNP_GCP_SQL_ENABLED")
	if !c.flag(ctx, cloudSQL) {
		t.Skipf("feature flag %s is not enabled", cloudSQL)
	}
}

// TestAccPolarisGCPProject_cloudSQL verifies that the Cloud SQL protection
// feature can be onboarded on a GCP project, together with the CLOUDSQL
// permission group of the EXOCOMPUTE and GCP_SHARED_VPC_HOST features, which
// RSC requires for Cloud SQL archival and archived recovery.
//
// The test skips unless the CNP_GCP_SQL_ENABLED feature flag is enabled for the
// RSC account, since RSC rejects the feature without it.
func TestAccPolarisGCPProject_cloudSQL(t *testing.T) {
	requireCloudSQLFeatureFlag(t)

	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	projectCloudSQL, err := makeTerraformConfig(config, gcpProjectCloudSQLTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: projectCloudSQL,
			Check: resource.ComposeTestCheckFunc(
				// Project resource.
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_name", project.ProjectName),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project_number", strconv.FormatInt(project.ProjectNumber, 10)),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "3"),

				// Cloud SQL Protection feature.
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_SQL_PROTECTION",
					"permissions":         "",
					"permission_groups.#": "2",
					"status":              "CONNECTED",
				}),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "BASIC"),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "EXPORT_AND_RESTORE"),

				// Exocompute feature, with the Cloud SQL permission group.
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "EXOCOMPUTE",
					"permissions":         "",
					"permission_groups.#": "2",
					"status":              "CONNECTED",
				}),

				// Shared VPC host feature, with the Cloud SQL permission group.
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "GCP_SHARED_VPC_HOST",
					"permissions":         "",
					"permission_groups.#": "2",
					"status":              "CONNECTED",
				}),
				resource.TestCheckTypeSetElemAttr("polaris_gcp_project.default", "feature.*.permission_groups.*", "CLOUDSQL"),
			),
		}},
	})
}

const gcpProjectCloudDiscoveryTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_DISCOVERY"
		permission_groups = [
			"BASIC",
		]
	}

	feature {
		name = "CLOUD_NATIVE_PROTECTION"
		permission_groups = [
			"BASIC",
		]
	}
}
`

const gcpProjectCloudDiscoveryOnlyTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_DISCOVERY"
		permission_groups = [
			"BASIC",
		]
	}
}
`

const gcpProjectProtectionOnlyTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_gcp_project" "default" {
	credentials    = "{{ .Resource.Credentials }}"
	project        = "{{ .Resource.ProjectID }}"
	project_name   = "{{ .Resource.ProjectName }}"
	project_number = {{ .Resource.ProjectNumber }}

	feature {
		name = "CLOUD_NATIVE_PROTECTION"
		permission_groups = [
			"BASIC",
		]
	}
}
`

// requireCloudDiscoveryFeatureFlag skips the test if the GCP Cloud Discovery
// feature is not enabled for the account.
func requireCloudDiscoveryFeatureFlag(t *testing.T) {
	t.Helper()

	credentials := os.Getenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE")
	if credentials == "" {
		t.Skip("RUBRIK_POLARIS_SERVICEACCOUNT_FILE not set")
	}

	ctx := context.Background()
	c, err := newClient(ctx, credentials, polaris.CacheParams{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if !c.flag(ctx, core.FeatureFlagGCPCloudDiscovery) {
		t.Skipf("feature flag %s is not enabled", core.FeatureFlagGCPCloudDiscovery)
	}
}

// TestAccPolarisGCPProject_cloudDiscovery verifies that the Cloud Discovery
// feature can be onboarded together with a protection feature, that the plan
// refuses to remove Cloud Discovery while the protection feature is enabled,
// and that removing the protection feature first, then Cloud Discovery, works.
//
// The test skips unless the REL_ENABLE_CNP_GCP_CLOUD_DISCOVERY_MODE feature
// flag is enabled for the RSC account, since RSC rejects the feature without it.
func TestAccPolarisGCPProject_cloudDiscovery(t *testing.T) {
	requireCloudDiscoveryFeatureFlag(t)

	config, project, err := loadGCPTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	withBoth, err := makeTerraformConfig(config, gcpProjectCloudDiscoveryTmpl)
	if err != nil {
		t.Fatal(err)
	}
	cloudDiscoveryOnly, err := makeTerraformConfig(config, gcpProjectCloudDiscoveryOnlyTmpl)
	if err != nil {
		t.Fatal(err)
	}
	protectionOnly, err := makeTerraformConfig(config, gcpProjectProtectionOnlyTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Onboard Cloud Discovery together with a protection feature.
			Config: withBoth,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "project", project.ProjectID),
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "2"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_DISCOVERY",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "connected",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_NATIVE_PROTECTION",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "connected",
				}),
			),
		}, {
			// Removing Cloud Discovery while the protection feature remains is
			// rejected at plan time, before anything is sent to RSC.
			Config:      protectionOnly,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`CLOUD_DISCOVERY cannot be removed while the protection feature CLOUD_NATIVE_PROTECTION is enabled`),
		}, {
			// Remove the protection feature first, keeping Cloud Discovery.
			Config: cloudDiscoveryOnly,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_gcp_project.default", "feature.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("polaris_gcp_project.default", "feature.*", map[string]string{
					"%":                   "4",
					"name":                "CLOUD_DISCOVERY",
					"permissions":         "",
					"permission_groups.#": "1",
					"status":              "connected",
				}),
			),
		}},
		// Destroy removes the remaining Cloud Discovery feature.
	})
}

// TestGcpCheckCloudDiscoveryRemoval verifies the plan-time rule that Cloud
// Discovery can only be removed from a project once no protection feature is
// left on it.
func TestGcpCheckCloudDiscoveryRemoval(t *testing.T) {
	tests := []struct {
		name        string
		oldFeatures []string
		newFeatures []string
		wantErr     string
	}{{
		name:        "RemoveCloudDiscoveryWithProtectionLeft",
		oldFeatures: []string{"CLOUD_DISCOVERY", "CLOUD_NATIVE_PROTECTION"},
		newFeatures: []string{"CLOUD_NATIVE_PROTECTION"},
		wantErr:     "CLOUD_DISCOVERY cannot be removed while the protection feature CLOUD_NATIVE_PROTECTION is enabled",
	}, {
		name:        "RemoveCloudDiscoveryWithCloudSQLLeft",
		oldFeatures: []string{"CLOUD_DISCOVERY", "CLOUD_SQL_PROTECTION", "EXOCOMPUTE"},
		newFeatures: []string{"CLOUD_SQL_PROTECTION", "EXOCOMPUTE"},
		wantErr:     "protection feature CLOUD_SQL_PROTECTION is enabled",
	}, {
		name:        "RemoveCloudDiscoveryAndProtectionTogether",
		oldFeatures: []string{"CLOUD_DISCOVERY", "CLOUD_NATIVE_PROTECTION"},
		newFeatures: []string{"EXOCOMPUTE"},
	}, {
		name:        "RemoveCloudDiscoveryWithOnlyNonProtectionLeft",
		oldFeatures: []string{"CLOUD_DISCOVERY", "EXOCOMPUTE", "GCP_SHARED_VPC_HOST"},
		newFeatures: []string{"EXOCOMPUTE", "GCP_SHARED_VPC_HOST"},
	}, {
		name:        "KeepCloudDiscovery",
		oldFeatures: []string{"CLOUD_DISCOVERY", "CLOUD_NATIVE_PROTECTION"},
		newFeatures: []string{"CLOUD_DISCOVERY"},
	}, {
		name:        "CloudDiscoveryNeverPresent",
		oldFeatures: []string{"CLOUD_NATIVE_PROTECTION"},
		newFeatures: []string{"CLOUD_NATIVE_PROTECTION", "CLOUD_SQL_PROTECTION"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gcpCheckCloudDiscoveryRemoval(tt.oldFeatures, tt.newFeatures)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Errorf("expected error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
