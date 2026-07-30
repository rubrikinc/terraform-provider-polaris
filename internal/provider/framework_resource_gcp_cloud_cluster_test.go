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

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateGcpCloudClusterConfig(t *testing.T) {
	azConfigs := []gcpSubnetAzConfigModel{
		{AvailabilityZone: types.StringValue("us-west1-a"), Subnet: types.StringValue("subnet-a")},
		{AvailabilityZone: types.StringValue("us-west1-b"), Subnet: types.StringValue("subnet-b")},
		{AvailabilityZone: types.StringValue("us-west1-c"), Subnet: types.StringValue("subnet-c")},
	}

	// model builds a config with the fields validateGcpCloudClusterConfig reads.
	model := func(azResilient types.Bool, subnet types.String, numNodes types.Int64, subnetAz []gcpSubnetAzConfigModel) gcpCloudClusterModel {
		return gcpCloudClusterModel{
			AZResilient:   azResilient,
			ClusterConfig: []gcpClusterConfigModel{{NumNodes: numNodes}},
			VMConfig:      []gcpVMConfigModel{{Subnet: subnet, SubnetAzConfig: subnetAz}},
		}
	}

	tests := []struct {
		name      string
		config    gcpCloudClusterModel
		wantError bool
	}{
		{
			name:      "single-AZ with subnet is valid",
			config:    model(types.BoolValue(false), types.StringValue("subnet-a"), types.Int64Value(1), nil),
			wantError: false,
		},
		{
			name:      "single-AZ without subnet is rejected",
			config:    model(types.BoolValue(false), types.StringValue(""), types.Int64Value(1), nil),
			wantError: true,
		},
		{
			name:      "single-AZ with subnet_az_config is rejected",
			config:    model(types.BoolValue(false), types.StringValue("subnet-a"), types.Int64Value(1), azConfigs),
			wantError: true,
		},
		{
			name:      "multi-AZ with three nodes and az configs is valid",
			config:    model(types.BoolValue(true), types.StringNull(), types.Int64Value(3), azConfigs),
			wantError: false,
		},
		{
			name:      "multi-AZ without az configs is rejected",
			config:    model(types.BoolValue(true), types.StringNull(), types.Int64Value(3), nil),
			wantError: true,
		},
		{
			name:      "multi-AZ with subnet is rejected",
			config:    model(types.BoolValue(true), types.StringValue("subnet-a"), types.Int64Value(3), azConfigs),
			wantError: true,
		},
		{
			name:      "multi-AZ with fewer than three nodes is rejected",
			config:    model(types.BoolValue(true), types.StringNull(), types.Int64Value(2), azConfigs),
			wantError: true,
		},
		{
			name:      "unknown az_resilient skips validation",
			config:    model(types.BoolUnknown(), types.StringValue(""), types.Int64Value(1), nil),
			wantError: false,
		},
		{
			name:      "unknown subnet skips the subnet-required check",
			config:    model(types.BoolValue(false), types.StringUnknown(), types.Int64Value(1), nil),
			wantError: false,
		},
		{
			name:      "unknown num_nodes skips the multi-AZ node check",
			config:    model(types.BoolValue(true), types.StringNull(), types.Int64Unknown(), azConfigs),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateGcpCloudClusterConfig(tt.config)
			if got := diags.HasError(); got != tt.wantError {
				t.Errorf("validateGcpCloudClusterConfig() hasError = %v, want %v (diags: %v)", got, tt.wantError, diags)
			}
		})
	}
}
