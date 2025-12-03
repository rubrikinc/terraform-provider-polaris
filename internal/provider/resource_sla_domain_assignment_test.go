// Copyright 2025 Rubrik, Inc.
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

// slaDomainAssignmentProtectWithSlaTmpl creates a tag rule and an SLA domain, then assigns
// the SLA domain to the tag rule.
const slaDomainAssignmentProtectWithSlaTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for SLA Assignment"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-sla-assignment"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA Domain for Assignment"
	description  = "SLA Domain for assignment testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots   = true
	apply_changes_to_non_policy_snapshots = false
}
`

// slaDomainAssignmentDoNotProtectTmpl creates a tag rule and a "do not protect" assignment.
const slaDomainAssignmentDoNotProtectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for Do Not Protect"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-do-not-protect"
	tag_all_values = true
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`

// slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl updates a "do not protect" assignment
// with expire immediately retention.
const slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for Do Not Protect"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-do-not-protect"
	tag_all_values = true
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "EXPIRE_IMMEDIATELY"
}
`

// slaDomainAssignmentUpdateTmpl updates the SLA domain assignment with different settings.
const slaDomainAssignmentUpdateTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for SLA Assignment"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-sla-assignment"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA Domain for Assignment"
	description  = "SLA Domain for assignment testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots   = true
	apply_changes_to_non_policy_snapshots = true
}
`

// slaDomainAssignmentSwitchToDoNotProtectTmpl switches from protectWithSla to doNotProtect
// while keeping the SLA domain resource (to prevent deletion during update).
const slaDomainAssignmentSwitchToDoNotProtectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for SLA Assignment"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-sla-assignment"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA Domain for Assignment"
	description  = "SLA Domain for assignment testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`

func TestAccPolarisSLADomainAssignment_protectWithSla(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	protectWithSla, err := makeTerraformConfig(config, slaDomainAssignmentProtectWithSlaTmpl)
	if err != nil {
		t.Fatal(err)
	}

	updateConfig, err := makeTerraformConfig(config, slaDomainAssignmentUpdateTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: protectWithSla,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_existing_snapshots", "true"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_non_policy_snapshots", "false"),
			),
		}, {
			Config: updateConfig,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_existing_snapshots", "true"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_non_policy_snapshots", "true"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_doNotProtect(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	doNotProtect, err := makeTerraformConfig(config, slaDomainAssignmentDoNotProtectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	doNotProtectExpire, err := makeTerraformConfig(config, slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: doNotProtect,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "id", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "RETAIN_SNAPSHOTS"),
			),
		}, {
			Config: doNotProtectExpire,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "EXPIRE_IMMEDIATELY"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_switchAssignmentType(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	protectWithSla, err := makeTerraformConfig(config, slaDomainAssignmentProtectWithSlaTmpl)
	if err != nil {
		t.Fatal(err)
	}

	switchToDoNotProtect, err := makeTerraformConfig(config, slaDomainAssignmentSwitchToDoNotProtectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: protectWithSla,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}, {
			Config: switchToDoNotProtect,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "id", "doNotProtect"),
			),
		}, {
			Config: protectWithSla,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}},
	})
}

// slaDomainAssignmentMultipleObjectsTmpl creates multiple tag rules assigned to the same SLA.
const slaDomainAssignmentMultipleObjectsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "first" {
	name           = "Test Tag Rule First"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-first"
	tag_all_values = true
}

resource "polaris_tag_rule" "second" {
	name           = "Test Tag Rule Second"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-second"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA for Multiple Objects"
	description  = "SLA Domain for multiple objects testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.first.id, polaris_tag_rule.second.id]

	apply_changes_to_existing_snapshots = true
}
`

// slaDomainAssignmentMultipleObjectsDoNotProtectTmpl switches multiple objects to doNotProtect.
const slaDomainAssignmentMultipleObjectsDoNotProtectTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "first" {
	name           = "Test Tag Rule First"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-first"
	tag_all_values = true
}

resource "polaris_tag_rule" "second" {
	name           = "Test Tag Rule Second"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-second"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA for Multiple Objects"
	description  = "SLA Domain for multiple objects testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.first.id, polaris_tag_rule.second.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"

	# With RETAIN_SNAPSHOTS, objects keep the previous SLA as their configured
	# SLA (for retention). This dependency ensures proper deletion ordering so
	# the assignment is removed before the SLA domain.
	depends_on = [polaris_sla_domain.test]
}
`

// slaDomainAssignmentMultipleAssignmentsTmpl creates two separate assignments to the same SLA.
const slaDomainAssignmentMultipleAssignmentsTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "first" {
	name           = "Test Tag Rule First"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-first"
	tag_all_values = true
}

resource "polaris_tag_rule" "second" {
	name           = "Test Tag Rule Second"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-multi-second"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA for Multiple Objects"
	description  = "SLA Domain for multiple objects testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain_assignment" "first" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.first.id]

	apply_changes_to_existing_snapshots = true
}

resource "polaris_sla_domain_assignment" "second" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.second.id]

	apply_changes_to_existing_snapshots = true
}
`

func TestAccPolarisSLADomainAssignment_multipleObjects(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	multipleObjects, err := makeTerraformConfig(config, slaDomainAssignmentMultipleObjectsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	multipleObjectsDoNotProtect, err := makeTerraformConfig(config, slaDomainAssignmentMultipleObjectsDoNotProtectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	multipleAssignments, err := makeTerraformConfig(config, slaDomainAssignmentMultipleAssignmentsTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Assign multiple objects to SLA with single assignment resource
			Config: multipleObjects,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "2"),
			),
		}, {
			// Step 2: Switch multiple objects to doNotProtect
			Config: multipleObjectsDoNotProtect,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "2"),
			),
		}, {
			// Step 3: Use two separate assignments to the same SLA
			Config: multipleAssignments,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.first", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.first", "object_ids.#", "1"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.second", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.second", "object_ids.#", "1"),
			),
		}},
	})
}
