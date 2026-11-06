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

	azureRegion "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
)

func TestAzureCloudClusterRegion(t *testing.T) {
	tests := []struct {
		name        string
		rscRegion   string
		stateRegion string
		expected    azureRegion.Region
	}{
		{"NativeName", "germanywestcentral", "eastus", azureRegion.RegionGermanyWestCentral},
		{"PascalCase", "GermanyWestCentral", "eastus", azureRegion.RegionGermanyWestCentral},
		{"UpperCase", "GERMANYWESTCENTRAL", "eastus", azureRegion.RegionGermanyWestCentral},
		{"UnknownFallsBackToState", "Germany West Central", "germanywestcentral", azureRegion.RegionGermanyWestCentral},
		{"EmptyFallsBackToState", "", "germanywestcentral", azureRegion.RegionGermanyWestCentral},
		{"BothUnknown", "", "", azureRegion.RegionUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if region := azureCloudClusterRegion(test.rscRegion, test.stateRegion); region != test.expected {
				t.Errorf("got %q want %q", region, test.expected)
			}
		})
	}
}
