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

import "github.com/google/uuid"

// testDataCenter holds the CDM cluster and archival location
// configuration loaded from TEST_DATACENTER_FILE.
type testDataCenter struct {
	ClusterUUID       uuid.UUID              `json:"clusterUuid"`
	ClusterIP         string                 `json:"clusterIp"`
	ArchivalLocations []testArchivalLocation `json:"archivalLocations"`
}

// testArchivalLocation represents a single archival location on the
// CDM cluster.
type testArchivalLocation struct {
	LocationID   string `json:"locationId"`
	LocationName string `json:"locationName"`
	LocationType string `json:"locationType"`
	Bucket       string `json:"bucket"`
}

// testDataCenterArchivalLocation is the per-subtest view passed to a
// Terraform template, pairing the cluster with one archival location.
type testDataCenterArchivalLocation struct {
	ClusterUUID uuid.UUID
	ClusterIP   string
	testArchivalLocation
}

// loadDataCenterTestConfig loads a Data Center (CDM) test configuration
// using the default environment variables.
func loadDataCenterTestConfig() (testConfig, testDataCenter, error) {
	dc := testDataCenter{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_DATACENTER_FILE", &dc)

	return config, dc, err
}
