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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Acceptance test templates

const tagRuleWithValueTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name        = "Test Tag Rule With Value"
	object_type = "AWS_EC2_INSTANCE"
	tag_key     = "Environment"
	tag_value   = "Production"
}

data "polaris_tag_rule" "default_by_id" {
	id = polaris_tag_rule.default.id
}

data "polaris_tag_rule" "default_by_name" {
	name = polaris_tag_rule.default.name
}
`

const tagRuleWithAllValuesTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name           = "Test Tag Rule All Values"
	object_type    = "AWS_EC2_INSTANCE"
	tag_key        = "Environment"
	tag_all_values = true
}

data "polaris_tag_rule" "default_by_id" {
	id = polaris_tag_rule.default.id
}

data "polaris_tag_rule" "default_by_name" {
	name = polaris_tag_rule.default.name
}
`

const tagRuleUpdatedTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name        = "Test Tag Rule Updated"
	object_type = "AWS_EC2_INSTANCE"
	tag_key     = "Environment"
	tag_value   = "Production"
}

data "polaris_tag_rule" "default_by_id" {
	id = polaris_tag_rule.default.id
}

data "polaris_tag_rule" "default_by_name" {
	name = polaris_tag_rule.default.name
}
`

// Acceptance test functions

func TestAccPolarisTagRule_withValue(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRuleWithValue, err := makeTerraformConfig(config, tagRuleWithValueTmpl)
	if err != nil {
		t.Fatal(err)
	}

	tagRuleUpdated, err := makeTerraformConfig(config, tagRuleUpdatedTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRuleWithValue,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule With Value"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_value", "Production"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_all_values", "false"),

				// Data source checks (by ID)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule With Value"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_value", "Production"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_all_values", "false"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule With Value"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag_value", "Production"),
			),
		}, {
			Config: tagRuleUpdated,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks - name should be updated
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule Updated"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_value", "Production"),

				// Data source checks (by ID)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule Updated"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule Updated"),
			),
		}},
	})
}

func TestAccPolarisTagRule_withAllValues(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRuleWithAllValues, err := makeTerraformConfig(config, tagRuleWithAllValuesTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRuleWithAllValues,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule All Values"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_value", ""),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_all_values", "true"),

				// Data source checks (by ID)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule All Values"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_all_values", "true"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule All Values"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag_all_values", "true"),
			),
		}},
	})
}
