// Copyright 2024 Rubrik, Inc.
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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestFromCloudComputeSettings verifies that fromCloudComputeSettings reads the
// cloud_compute_settings block rather than archival_proxy_settings. Configuring
// both blocks previously caused a nil interface conversion panic because the
// function read the proxy block, which has no vpc_id/subnet_id/security_group_id
// keys.
func TestFromCloudComputeSettings(t *testing.T) {
	res := resourceDataCenterArchivalLocationAmazonS3()

	// Both blocks set: the proxy block must not be mistaken for the compute
	// block.
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]any{
		keyArchivalProxySettings: []any{
			map[string]any{
				keyProxyServer: "10.0.0.1",
				keyProtocol:    "HTTPS",
				keyPortNumber:  8080,
				keyUsername:    "proxyuser",
				keyPassword:    "proxypass",
			},
		},
		keyCloudComputeSettings: []any{
			map[string]any{
				keyVPCID:                 "vpc-12345678",
				keySubnetID:              "subnet-12345678",
				keySecurityGroupID:       "sg-12345678",
				keyArchivalConsolidation: true,
			},
		},
	})

	settings, consolidation := fromCloudComputeSettings(d)
	if settings == nil {
		t.Fatal("expected non-nil cloud compute settings")
	}
	if got, want := settings.VPCID, "vpc-12345678"; got != want {
		t.Errorf("VPCID = %q, want %q", got, want)
	}
	if got, want := settings.SubnetID, "subnet-12345678"; got != want {
		t.Errorf("SubnetID = %q, want %q", got, want)
	}
	if got, want := settings.SecurityGroupID, "sg-12345678"; got != want {
		t.Errorf("SecurityGroupID = %q, want %q", got, want)
	}
	if !consolidation {
		t.Error("archival consolidation = false, want true")
	}
}

// TestFromCloudComputeSettingsUnset verifies that when no cloud_compute_settings
// block is configured the function reports it as unset, even if an
// archival_proxy_settings block is present.
func TestFromCloudComputeSettingsUnset(t *testing.T) {
	res := resourceDataCenterArchivalLocationAmazonS3()

	d := schema.TestResourceDataRaw(t, res.Schema, map[string]any{
		keyArchivalProxySettings: []any{
			map[string]any{
				keyProxyServer: "10.0.0.1",
				keyProtocol:    "HTTPS",
				keyPortNumber:  8080,
				keyUsername:    "proxyuser",
				keyPassword:    "proxypass",
			},
		},
	})

	if settings, _ := fromCloudComputeSettings(d); settings != nil {
		t.Errorf("expected nil cloud compute settings, got %+v", settings)
	}
}
