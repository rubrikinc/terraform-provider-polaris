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

//go:build cdm

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const dataCenterArchivalLocationTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

data "polaris_data_center_archival_location" "test" {
	cluster_id = "{{ .Resource.ClusterUUID }}"
	name       = "{{ .Resource.LocationName }}"
}
`

func TestAccCDMDataCenterArchivalLocation(t *testing.T) {
	config, dc, err := loadDataCenterTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(dc.ArchivalLocations) == 0 {
		t.Fatal("TEST_DATACENTER_FILE has no archival locations")
	}

	for _, loc := range dc.ArchivalLocations {
		t.Run(loc.LocationName, func(t *testing.T) {
			config.Resource = testDataCenterArchivalLocation{
				ClusterUUID:          dc.ClusterUUID,
				ClusterIP:            dc.ClusterIP,
				testArchivalLocation: loc,
			}

			dataCenterArchivalLocation, err := makeTerraformConfig(config, dataCenterArchivalLocationTmpl)
			if err != nil {
				t.Fatal(err)
			}

			const ds = "data.polaris_data_center_archival_location.test"
			resource.Test(t, resource.TestCase{
				ProviderFactories: providerFactories,
				Steps: []resource.TestStep{{
					Config: dataCenterArchivalLocation,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(ds, "name", loc.LocationName),
						resource.TestCheckResourceAttr(ds, "id", loc.LocationID),
						resource.TestCheckResourceAttr(ds, "target_type", loc.LocationType),
						resource.TestCheckResourceAttrSet(ds, "status"),
						resource.TestCheckResourceAttrSet(ds, "cluster_name"),
						resource.TestCheckResourceAttrSet(ds, "cluster_status"),
						resource.TestCheckResourceAttrSet(ds, "cluster_version"),
					),
				}},
			})
		})
	}
}
