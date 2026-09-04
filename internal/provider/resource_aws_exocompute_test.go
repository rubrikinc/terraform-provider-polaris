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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var awsExocomputeTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_aws_account" "default" {
	name    = "{{ .Resource.AccountName }}"
	profile = "{{ .Resource.Profile }}"

	cloud_native_protection {
		permission_groups = [
			"BASIC",
		]
		regions = [
			"us-east-2",
		]
	}
  
	exocompute {
		permission_groups = [
			"BASIC",
			"RSC_MANAGED_CLUSTER",
		]

		regions = [
			"us-east-2",
		]
	}
}

resource "polaris_aws_exocompute" "default" {
	account_id = polaris_aws_account.default.id
	region     = "us-east-2"
	vpc_id     = "{{ .Resource.Exocompute.VPCID }}"

	subnets = [
		{{ range slice .Resource.Exocompute.Subnets 0 2 }}
		"{{ .ID }}",
		{{ end }}
	]
}
`

func TestAccPolarisAWSExocompute_basic(t *testing.T) {
	config, account, err := loadAWSTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	exocompute, err := makeTerraformConfig(config, awsExocomputeTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: exocompute,
			Check: resource.ComposeTestCheckFunc(
				// Account resource
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "profile", account.Profile),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "cloud_native_protection.0.regions.*", "us-east-2"),

				// Exocompute feature
				resource.TestCheckResourceAttr("polaris_aws_account.default", "exocompute.0.status", "connected"),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_account.default", "exocompute.0.regions.*", "us-east-2"),

				// Exocompute resource
				resource.TestCheckResourceAttrPair("polaris_aws_exocompute.default", "account_id", "polaris_aws_account.default", "id"),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "region", "us-east-2"),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "vpc_id", account.Exocompute.VPCID),
				resource.TestCheckResourceAttr("polaris_aws_exocompute.default", "polaris_managed", "true"),
				resource.TestCheckTypeSetElemAttr("polaris_aws_exocompute.default", "subnets.*", account.Exocompute.Subnets[0].ID),
				resource.TestCheckTypeSetElemAttr("polaris_aws_exocompute.default", "subnets.*", account.Exocompute.Subnets[1].ID),
			),
		}},
	})
}

// awsExocomputeRawConfig returns a minimal RSC managed AWS exocompute
// configuration. When withSecurityGroups is true, the deprecated
// cluster_security_group_id and node_security_group_id fields are included.
func awsExocomputeRawConfig(withSecurityGroups bool) map[string]any {
	raw := map[string]any{
		"account_id": "6f1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8",
		"region":     "us-east-2",
		"vpc_id":     "vpc-4859acb9",
		"subnets":    []any{"subnet-ea67b67b", "subnet-ea43ec78"},
	}
	if withSecurityGroups {
		raw["cluster_security_group_id"] = "sg-005656347687b8170"
		raw["node_security_group_id"] = "sg-00e147656785d7e2f"
	}
	return raw
}

// TestAWSExocomputeSecurityGroupDeprecationWarning verifies that the deprecated
// security group fields only warn when they are set in the configuration. Both
// fields are Optional and Computed, so a spurious warning would reach every RSC
// managed user, including those who never set them.
func TestAWSExocomputeSecurityGroupDeprecationWarning(t *testing.T) {
	res := resourceAwsExocompute()

	for _, diag := range res.Validate(terraform.NewResourceConfigRaw(awsExocomputeRawConfig(false))) {
		if strings.Contains(strings.ToLower(diag.Summary), "deprecat") {
			t.Errorf("unexpected deprecation diagnostic when the security group fields are absent: %s", diag.Summary)
		}
	}

	var warnings int
	for _, diag := range res.Validate(terraform.NewResourceConfigRaw(awsExocomputeRawConfig(true))) {
		if strings.Contains(strings.ToLower(diag.Summary), "deprecat") {
			warnings++
		}
	}
	if warnings != 2 {
		t.Errorf("expected 2 deprecation diagnostics when both security group fields are set, got %d", warnings)
	}
}

// awsExocomputeReadState builds the instance state the way awsReadExocompute
// writes it, including the computed attributes a raw configuration leaves out.
// Read populates both subnets and subnet, so a state built from configuration
// alone is not representative of a deployed configuration.
func awsExocomputeReadState(t *testing.T, clusterSecurityGroupID, nodeSecurityGroupID string, rscManaged bool) *terraform.InstanceState {
	t.Helper()

	res := resourceAwsExocompute()
	data := schema.TestResourceDataRaw(t, res.Schema, map[string]any{})
	data.SetId("0a1b2c3d-4e5f-6071-8293-a4b5c6d7e8f9")

	set := func(key string, value any) {
		t.Helper()
		if err := data.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	set(keyAccountID, "6f1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8")
	set(keyRegion, "us-east-2")
	set(keyVPCID, "vpc-4859acb9")
	set(keyClusterSecurityGroupID, clusterSecurityGroupID)
	set(keyNodeSecurityGroupID, nodeSecurityGroupID)
	set(keyPolarisManaged, rscManaged)
	set(keyClusterAccess, "EKS_CLUSTER_ACCESS_TYPE_PUBLIC")

	subnetIDs := schema.Set{F: schema.HashString}
	subnetIDs.Add("subnet-ea67b67b")
	subnetIDs.Add("subnet-ea43ec78")
	set(keySubnets, &subnetIDs)

	subnets := schema.Set{F: schema.HashResource(awsSubnetResource())}
	subnets.Add(map[string]any{keySubnetID: "subnet-ea67b67b", keyPodSubnetID: ""})
	subnets.Add(map[string]any{keySubnetID: "subnet-ea43ec78", keyPodSubnetID: ""})
	set(keySubnet, &subnets)

	return data.State()
}

// TestAWSExocomputeRemoveSecurityGroupsNoDiff verifies that dropping the
// deprecated security group fields from a configuration which already has them
// applied does not plan a change, and in particular does not force the
// exocompute configuration to be replaced.
//
// The check goes through SimpleDiff, which is the entry point
// PlanResourceChange uses. It does not cover the protocol layer above that, so
// it is not a substitute for running terraform plan against a deployment.
func TestAWSExocomputeRemoveSecurityGroupsNoDiff(t *testing.T) {
	res := resourceAwsExocompute()
	state := awsExocomputeReadState(t, "sg-005656347687b8170", "sg-00e147656785d7e2f", false)

	diff, err := res.SimpleDiff(context.Background(), state, terraform.NewResourceConfigRaw(awsExocomputeRawConfig(false)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		return
	}
	if diff.RequiresNew() {
		t.Errorf("dropping the security group fields forces a new resource: %v", diff.Attributes)
	}
	if !diff.Empty() {
		t.Errorf("dropping the security group fields plans a change: %v", diff.Attributes)
	}
}

// TestAWSExocomputeKeepSecurityGroupsForcesNew covers the opposite case. A
// configuration which still supplies security group IDs for an exocompute
// configuration RSC reports as RSC managed differs from the remote state, and
// because both fields force a new resource that is planned as a replacement of
// the exocompute configuration.
func TestAWSExocomputeKeepSecurityGroupsForcesNew(t *testing.T) {
	res := resourceAwsExocompute()
	state := awsExocomputeReadState(t, "", "", true)

	diff, err := res.SimpleDiff(context.Background(), state, terraform.NewResourceConfigRaw(awsExocomputeRawConfig(true)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil || !diff.RequiresNew() {
		t.Error("expected supplying security group IDs for an RSC managed configuration to force a new resource")
	}
}
