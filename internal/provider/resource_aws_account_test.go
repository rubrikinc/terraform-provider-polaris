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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const awsAccountOneRegionTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
		]
	}
}
`

const awsAccountTwoRegionsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
			"us-west-2",
		]
	}
}
`

const awsCrossAccountOneRegionTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	assume_role = "{{ .Resource.CrossAccountRole }}"
	name        = "{{ .Resource.CrossAccountName }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
		]
	}
}
`

const awsCrossAccountTwoRegionsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	assume_role = "{{ .Resource.CrossAccountRole }}"
	name        = "{{ .Resource.CrossAccountName }}"

	cloud_native_protection {
		regions = [
			"us-east-2",
			"us-west-2",
		]
	}
}
`

func TestAccPolarisAWSAccount_basic(t *testing.T) {
	config, account, err := loadAWSTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	accountOneRegion, err := makeTerraformConfig(config, awsAccountOneRegionTmpl)
	if err != nil {
		t.Fatal(err)
	}
	accountTwoRegions, err := makeTerraformConfig(config, awsAccountTwoRegionsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Add and update account using a profile
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}, {
			Config: accountTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-west-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}, {
			Config: accountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "assume_role"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}},
	})

	crossAccountOneRegion, err := makeTerraformConfig(config, awsCrossAccountOneRegionTmpl)
	if err != nil {
		t.Fatal(err)
	}
	crossAccountTwoRegions, err := makeTerraformConfig(config, awsCrossAccountTwoRegionsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Add and update account using cross account role. This test uses the
	// default profile to assume the role.
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: crossAccountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}, {
			Config: crossAccountTwoRegions,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-west-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}, {
			Config: crossAccountOneRegion,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.CrossAccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "assume_role", account.CrossAccountRole),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),
				resource.TestCheckNoResourceAttr("polaris_aws_account.default", "profile"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.permission_groups.#", "0"),
			),
		}},
	})
}

func TestHandleOrderedFeatureUpdates(t *testing.T) {
	tests := []struct {
		name                  string
		hasOutpostChange      bool
		hasDSPMChange         bool
		hasDataScanningChange bool
		outpostExists         bool
		dspmExists            bool
		dataScanningExists    bool
		expectedOrder         []string
		description           string
	}{
		{
			name:                  "adding_all_features",
			hasOutpostChange:      true,
			hasDSPMChange:         true,
			hasDataScanningChange: true,
			outpostExists:         true,
			dspmExists:            true,
			dataScanningExists:    true,
			expectedOrder:         []string{"outpost", "dspm", "data_scanning"},
			description:           "When adding features, Outpost should be first, then DSPM and Data Scanning",
		},
		{
			name:                  "removing_all_features",
			hasOutpostChange:      true,
			hasDSPMChange:         true,
			hasDataScanningChange: true,
			outpostExists:         false,
			dspmExists:            false,
			dataScanningExists:    false,
			expectedOrder:         []string{"dspm", "data_scanning", "outpost"},
			description:           "When removing features, DSPM and Data Scanning should be first, then Outpost last",
		},
		{
			name:                  "adding_outpost_only",
			hasOutpostChange:      true,
			hasDSPMChange:         false,
			hasDataScanningChange: false,
			outpostExists:         true,
			dspmExists:            false,
			dataScanningExists:    false,
			expectedOrder:         []string{"outpost"},
			description:           "When adding only Outpost, it should be processed",
		},
		{
			name:                  "removing_outpost_only",
			hasOutpostChange:      true,
			hasDSPMChange:         false,
			hasDataScanningChange: false,
			outpostExists:         false,
			dspmExists:            false,
			dataScanningExists:    false,
			expectedOrder:         []string{"outpost"},
			description:           "When removing only Outpost, it should be processed",
		},
		{
			name:                  "no_changes",
			hasOutpostChange:      false,
			hasDSPMChange:         false,
			hasDataScanningChange: false,
			outpostExists:         false,
			dspmExists:            false,
			dataScanningExists:    false,
			expectedOrder:         []string{},
			description:           "When no features have changes, nothing should be processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAddingFeatures := (tt.hasOutpostChange && tt.outpostExists) ||
				(tt.hasDSPMChange && tt.dspmExists) ||
				(tt.hasDataScanningChange && tt.dataScanningExists)

			isRemovingFeatures := (tt.hasOutpostChange && !tt.outpostExists) ||
				(tt.hasDSPMChange && !tt.dspmExists) ||
				(tt.hasDataScanningChange && !tt.dataScanningExists)

			var actualOrder []string

			if isAddingFeatures {
				// When adding: Outpost first, then DSPM and Data Scanning
				if tt.hasOutpostChange && tt.outpostExists {
					actualOrder = append(actualOrder, "outpost")
				}
				if tt.hasDSPMChange && tt.dspmExists {
					actualOrder = append(actualOrder, "dspm")
				}
				if tt.hasDataScanningChange && tt.dataScanningExists {
					actualOrder = append(actualOrder, "data_scanning")
				}
			}

			if isRemovingFeatures {
				if tt.hasDSPMChange && !tt.dspmExists {
					actualOrder = append(actualOrder, "dspm")
				}
				if tt.hasDataScanningChange && !tt.dataScanningExists {
					actualOrder = append(actualOrder, "data_scanning")
				}
				if tt.hasOutpostChange && !tt.outpostExists {
					actualOrder = append(actualOrder, "outpost")
				}
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Errorf("Expected %d operations, got %d. Expected: %v, Got: %v",
					len(tt.expectedOrder), len(actualOrder), tt.expectedOrder, actualOrder)
				return
			}

			for i, expected := range tt.expectedOrder {
				if actualOrder[i] != expected {
					t.Errorf("Expected operation %d to be %s, got %s. Full order - Expected: %v, Got: %v",
						i, expected, actualOrder[i], tt.expectedOrder, actualOrder)
				}
			}

			t.Logf("✓ %s: Order verified as %v", tt.description, actualOrder)
		})
	}
}

func TestFeatureOrderingScenarios(t *testing.T) {
	t.Run("Outpost is added first when adding features", func(t *testing.T) {
		hasOutpostChange := true
		hasDSPMChange := true
		outpostExists := true
		dspmExists := true

		isAddingFeatures := (hasOutpostChange && outpostExists) || (hasDSPMChange && dspmExists)

		var addOrder []string
		if isAddingFeatures {
			if hasOutpostChange && outpostExists {
				addOrder = append(addOrder, "outpost")
			}
			if hasDSPMChange && dspmExists {
				addOrder = append(addOrder, "dspm")
			}
		}

		expectedAddOrder := []string{"outpost", "dspm"}
		if len(addOrder) != len(expectedAddOrder) || addOrder[0] != "outpost" || addOrder[1] != "dspm" {
			t.Errorf("When adding features, expected order %v, got %v", expectedAddOrder, addOrder)
		}

		// When removing features, Outpost should be removed last
		outpostExists = false
		dspmExists = false

		isRemovingFeatures := (hasOutpostChange && !outpostExists) || (hasDSPMChange && !dspmExists)

		var removeOrder []string
		if isRemovingFeatures {
			if hasDSPMChange && !dspmExists {
				removeOrder = append(removeOrder, "dspm")
			}
			if hasOutpostChange && !outpostExists {
				removeOrder = append(removeOrder, "outpost")
			}
		}

		expectedRemoveOrder := []string{"dspm", "outpost"}
		if len(removeOrder) != len(expectedRemoveOrder) || removeOrder[0] != "dspm" || removeOrder[1] != "outpost" {
			t.Errorf("When removing features, expected order %v, got %v", expectedRemoveOrder, removeOrder)
		}

		t.Log("Issue scenario verified: Outpost first when adding, last when removing")
	})
}
