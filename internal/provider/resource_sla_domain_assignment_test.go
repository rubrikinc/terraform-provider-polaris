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
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Common template fragments for SLA domain assignment tests.
const (
	tmplProvider = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}
`
	// tmplTagRuleTest creates a tag rule for SLA assignment tests.
	tmplTagRuleTest = `
resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for SLA Assignment"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-sla-assignment"
	tag_all_values = true
}
`
	// tmplTagRuleDoNotProtect creates a tag rule for do-not-protect tests.
	tmplTagRuleDoNotProtect = `
resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for Do Not Protect"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-do-not-protect"
	tag_all_values = true
}
`
	// tmplSLADomainTest creates an SLA domain for assignment tests.
	tmplSLADomainTest = `
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
`
)

// slaDomainAssignmentProtectWithSlaTmpl creates a tag rule and an SLA domain, then assigns
// the SLA domain to the tag rule.
const slaDomainAssignmentProtectWithSlaTmpl = tmplProvider + tmplTagRuleTest + tmplSLADomainTest + `
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots   = true
	apply_changes_to_non_policy_snapshots = false
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

	// Update the SLA domain assignment with different settings.
	updateConfig, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleTest+tmplSLADomainTest+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots   = true
	apply_changes_to_non_policy_snapshots = true
}
`)
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

// Templates for TestAccPolarisSLADomainAssignment_doNotProtect.
const (
	// slaDomainAssignmentDoNotProtectTmpl creates a tag rule and a "do not protect" assignment.
	slaDomainAssignmentDoNotProtectTmpl = tmplProvider + tmplTagRuleDoNotProtect + `
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`
	// slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl updates a "do not protect" assignment
	// with expire immediately retention.
	slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl = tmplProvider + tmplTagRuleDoNotProtect + `
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "EXPIRE_IMMEDIATELY"
}
`
)

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

	// Switch from protectWithSla to doNotProtect while keeping the SLA domain
	// resource (to prevent deletion during update).
	switchToDoNotProtect, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleTest+tmplSLADomainTest+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`)
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

// Template fragments for TestAccPolarisSLADomainAssignment_addRemoveObjects
// and TestAccPolarisSLADomainAssignment_multipleObjects.
const (
	// tmplTagRulesMulti creates two tag rules named "first" and "second".
	tmplTagRulesMulti = `
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
`
	// tmplSLADomainMulti creates an SLA domain for multiple objects testing.
	tmplSLADomainMulti = `
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
`
	// tmplTagRulesAddRemove creates two tag rules for add/remove testing.
	tmplTagRulesAddRemove = `
resource "polaris_tag_rule" "first" {
	name           = "Test Tag Rule First Add-Remove"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-add-remove-first"
	tag_all_values = true
}

resource "polaris_tag_rule" "second" {
	name           = "Test Tag Rule Second Add-Remove"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-add-remove-second"
	tag_all_values = true
}
`
	// tmplSLADomainAddRemove creates an SLA domain for add/remove testing.
	tmplSLADomainAddRemove = `
resource "polaris_sla_domain" "test" {
	name         = "Test SLA for Add-Remove"
	description  = "SLA Domain for add/remove object testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}
`
)

func TestAccPolarisSLADomainAssignment_addRemoveObjects(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Step 1: Start with a single object assigned.
	step1, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesAddRemove+tmplSLADomainAddRemove+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.first.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Step 2: Add a second object to the assignment.
	step2, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesAddRemove+tmplSLADomainAddRemove+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.first.id, polaris_tag_rule.second.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Step 3: Remove the first object, keeping only the second.
	step3, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesAddRemove+tmplSLADomainAddRemove+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.second.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Start with single object assigned
			Config: step1,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
			),
		}, {
			// Step 2: Add a second object to the assignment
			Config: step2,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "2"),
			),
		}, {
			// Step 3: Remove the first object, keeping only the second
			Config: step3,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_multipleObjects(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Create multiple tag rules assigned to the same SLA.
	multipleObjects, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesMulti+tmplSLADomainMulti+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.first.id, polaris_tag_rule.second.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Switch multiple objects to doNotProtect. With RETAIN_SNAPSHOTS, objects
	// keep the previous SLA as their configured SLA (for retention). The
	// dependency ensures proper deletion ordering.
	multipleObjectsDoNotProtect, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesMulti+tmplSLADomainMulti+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.first.id, polaris_tag_rule.second.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"

	depends_on = [polaris_sla_domain.test]
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Create two separate assignments to the same SLA.
	multipleAssignments, err := makeTerraformConfig(config, tmplProvider+tmplTagRulesMulti+tmplSLADomainMulti+`
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
`)
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
			// Step 3: Split into two separate assignments to the same SLA
			// This tests a potential race condition where the old resource is deleted
			// while new resources are created concurrently.
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

// Template fragments for TestAccPolarisSLADomainAssignment_changeSLADomain
// and TestAccPolarisSLADomainAssignment_doNotProtectToDifferentSLA.
const (
	// tmplTwoSLADomains creates two SLA domains for testing domain changes.
	tmplTwoSLADomains = `
resource "polaris_sla_domain" "first" {
	name         = "Test SLA Domain First"
	description  = "First SLA Domain for change testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 4
		retention      = 24
		retention_unit = "HOURS"
	}
}

resource "polaris_sla_domain" "second" {
	name         = "Test SLA Domain Second"
	description  = "Second SLA Domain for change testing"
	object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]

	hourly_schedule {
		frequency      = 6
		retention      = 48
		retention_unit = "HOURS"
	}
}
`
	// tmplTagRuleChangeSLA creates a tag rule for SLA change tests.
	tmplTagRuleChangeSLA = `
resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for SLA Change"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-sla-change"
	tag_all_values = true
}
`
)

func TestAccPolarisSLADomainAssignment_changeSLADomain(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Assign the first SLA domain.
	step1, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleChangeSLA+tmplTwoSLADomains+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.first.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Switch to the second SLA domain.
	step2, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleChangeSLA+tmplTwoSLADomains+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.second.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Assign first SLA domain
			Config: step1,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}, {
			// Step 2: Switch to second SLA domain
			Config: step2,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}, {
			// Step 3: Switch back to first SLA domain
			Config: step1,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_keepForeverRetention(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	retainSnapshots, err := makeTerraformConfig(config, slaDomainAssignmentDoNotProtectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Test KEEP_FOREVER retention.
	keepForever, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleDoNotProtect+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "KEEP_FOREVER"
}
`)
	if err != nil {
		t.Fatal(err)
	}

	expireImmediately, err := makeTerraformConfig(config, slaDomainAssignmentDoNotProtectExpireImmediatelyTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Start with RETAIN_SNAPSHOTS
			Config: retainSnapshots,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "RETAIN_SNAPSHOTS"),
			),
		}, {
			// Step 2: Switch to KEEP_FOREVER
			Config: keepForever,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "KEEP_FOREVER"),
			),
		}, {
			// Step 3: Switch to EXPIRE_IMMEDIATELY
			Config: expireImmediately,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "EXPIRE_IMMEDIATELY"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_doNotProtectNoRetention(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Test doNotProtect without retention policy.
	noRetention, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleDoNotProtect+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "doNotProtect"
	object_ids      = [polaris_tag_rule.test.id]
}
`)
	if err != nil {
		t.Fatal(err)
	}

	withRetention, err := makeTerraformConfig(config, slaDomainAssignmentDoNotProtectTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Create doNotProtect without retention policy
			Config: noRetention,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "id", "doNotProtect"),
			),
		}, {
			// Step 2: Add retention policy
			Config: withRetention,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "existing_snapshot_retention", "RETAIN_SNAPSHOTS"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_applyToExistingFalse(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Test apply_changes_to_existing_snapshots = false.
	applyFalse, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleTest+tmplSLADomainTest+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.test.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots = false
}
`)
	if err != nil {
		t.Fatal(err)
	}

	applyTrue, err := makeTerraformConfig(config, slaDomainAssignmentProtectWithSlaTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Create with apply_changes_to_existing_snapshots = false
			Config: applyFalse,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_existing_snapshots", "false"),
			),
		}, {
			// Step 2: Switch to apply_changes_to_existing_snapshots = true
			Config: applyTrue,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "apply_changes_to_existing_snapshots", "true"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_doNotProtectToDifferentSLA(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Start with doNotProtect.
	doNotProtect, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleChangeSLA+tmplTwoSLADomains+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`)
	if err != nil {
		t.Fatal(err)
	}

	// Switch to protectWithSlaId with second SLA.
	protectWithSLA, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleChangeSLA+tmplTwoSLADomains+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type = "protectWithSlaId"
	sla_domain_id   = polaris_sla_domain.second.id
	object_ids      = [polaris_tag_rule.test.id]

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Start with doNotProtect
			Config: doNotProtect,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "id", "doNotProtect"),
			),
		}, {
			// Step 2: Switch to protectWithSlaId with a specific SLA
			Config: protectWithSLA,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
			),
		}},
	})
}

func TestAccPolarisSLADomainAssignment_import(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Create resources for import testing.
	importConfig, err := makeTerraformConfig(config, tmplProvider+`
resource "polaris_tag_rule" "test" {
	name           = "Test Tag Rule for Import"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "test-import"
	tag_all_values = true
}

resource "polaris_sla_domain" "test" {
	name         = "Test SLA Domain for Import"
	description  = "SLA Domain for import testing"
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

	apply_changes_to_existing_snapshots = true
}
`)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Create the resources
			Config: importConfig,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "protectWithSlaId"),
				resource.TestCheckResourceAttrSet("polaris_sla_domain_assignment.default", "sla_domain_id"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
			),
		}, {
			// Step 2: Import the resource using the SLA domain ID
			ResourceName:            "polaris_sla_domain_assignment.default",
			ImportState:             true,
			ImportStateIdFunc:       testAccPolarisSLADomainAssignmentImportStateIdFunc("polaris_sla_domain_assignment.default"),
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"apply_changes_to_existing_snapshots", "apply_changes_to_non_policy_snapshots"},
		}},
	})
}

// testAccPolarisSLADomainAssignmentImportStateIdFunc returns the SLA domain ID for import.
func testAccPolarisSLADomainAssignmentImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		slaID := rs.Primary.Attributes["sla_domain_id"]
		if slaID == "" {
			return "", fmt.Errorf("sla_domain_id not set")
		}
		return slaID, nil
	}
}

func TestAccPolarisSLADomainAssignment_importDoNotProtect(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Create a doNotProtect assignment for import testing.
	doNotProtectConfig, err := makeTerraformConfig(config, tmplProvider+tmplTagRuleDoNotProtect+`
resource "polaris_sla_domain_assignment" "default" {
	assignment_type             = "doNotProtect"
	object_ids                  = [polaris_tag_rule.test.id]
	existing_snapshot_retention = "RETAIN_SNAPSHOTS"
}
`)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// Step 1: Create the doNotProtect assignment
			Config: doNotProtectConfig,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "assignment_type", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "id", "doNotProtect"),
				resource.TestCheckResourceAttr("polaris_sla_domain_assignment.default", "object_ids.#", "1"),
			),
		}, {
			// Step 2: Import the resource using the doNotProtect:<object_id> format
			ResourceName:            "polaris_sla_domain_assignment.default",
			ImportState:             true,
			ImportStateIdFunc:       testAccPolarisSLADomainAssignmentDoNotProtectImportStateIdFunc("polaris_sla_domain_assignment.default"),
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"existing_snapshot_retention", "apply_changes_to_existing_snapshots", "apply_changes_to_non_policy_snapshots"},
		}},
	})
}

// testAccPolarisSLADomainAssignmentDoNotProtectImportStateIdFunc returns the
// import ID in the format "doNotProtect:<object_id1>,<object_id2>,...".
func testAccPolarisSLADomainAssignmentDoNotProtectImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		// Get the object IDs from the resource state
		objectIDCount := rs.Primary.Attributes["object_ids.#"]
		if objectIDCount == "" || objectIDCount == "0" {
			return "", fmt.Errorf("no object_ids found in resource state")
		}

		// Collect all object IDs
		var objectIDs []string
		for key, value := range rs.Primary.Attributes {
			if len(key) > 11 && key[:11] == "object_ids." && key != "object_ids.#" {
				objectIDs = append(objectIDs, value)
			}
		}

		if len(objectIDs) == 0 {
			return "", fmt.Errorf("no object_ids found in resource state")
		}

		return "doNotProtect:" + objectIDs[0], nil
	}
}
