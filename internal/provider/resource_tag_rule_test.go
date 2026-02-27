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
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// Test using the new tag block style with a single tag condition.
const tagRuleSingleTagTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name        = "Test Tag Rule Single"
	object_type = "AWS_EC2_INSTANCE"

	tag {
		key    = "Environment"
		values = ["Production"]
	}
}

data "polaris_tag_rule" "default_by_id" {
	id = polaris_tag_rule.default.id
}

data "polaris_tag_rule" "default_by_name" {
	name = polaris_tag_rule.default.name
}
`

// Test using the new tag block style with multiple tag conditions.
const tagRuleMultiTagTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name        = "Test Tag Rule Multi"
	object_type = "AWS_EC2_INSTANCE"

	tag {
		key    = "Environment"
		values = ["Production", "Staging"]
	}

	tag {
		key       = "Owner"
		match_all = true
	}
}

data "polaris_tag_rule" "default_by_id" {
	id = polaris_tag_rule.default.id
}

data "polaris_tag_rule" "default_by_name" {
	name = polaris_tag_rule.default.name
}
`

// Test using the deprecated tag_key/tag_value style.
const tagRuleDeprecatedStyleTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_tag_rule" "default" {
	name        = "Test Tag Rule Deprecated"
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

// Test using the deprecated tag_all_values style.
const tagRuleDeprecatedAllValuesTmpl = `
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

func TestAccPolarisTagRule_singleTag(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRule, err := makeTerraformConfig(config, tagRuleSingleTagTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRule,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule Single"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.#", "1"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.values.#", "1"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.values.0", "Production"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.match_all", "false"),

				// Data source checks (by ID)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule Single"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag.#", "1"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag.0.key", "Environment"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule Single"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag.#", "1"),
			),
		}},
	})
}

// requireMultiTagFeatureFlag skips the test if the feature flag for multiple
// key-value pairs in tag rules is not enabled.
func requireMultiTagFeatureFlag(t *testing.T) {
	t.Helper()

	credentials := os.Getenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE")
	if credentials == "" {
		t.Skip("RUBRIK_POLARIS_SERVICEACCOUNT_FILE not set")
	}

	ctx := context.Background()
	c, err := newClient(ctx, credentials, polaris.CacheParams{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if !c.flag(ctx, core.FeatureFlagMultipleKeyValuePairsInTagRules) {
		t.Skipf("feature flag %s is not enabled", core.FeatureFlagMultipleKeyValuePairsInTagRules)
	}
}

func TestAccPolarisTagRule_multiTag(t *testing.T) {
	requireMultiTagFeatureFlag(t)

	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRule, err := makeTerraformConfig(config, tagRuleMultiTagTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRule,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule Multi"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.#", "2"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.values.#", "2"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.values.0", "Production"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.values.1", "Staging"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.0.match_all", "false"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.1.key", "Owner"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag.1.match_all", "true"),

				// Data source checks (by ID)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule Multi"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag.#", "2"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule Multi"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "tag.#", "2"),
			),
		}},
	})
}

func TestAccPolarisTagRule_deprecatedStyle(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRule, err := makeTerraformConfig(config, tagRuleDeprecatedStyleTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRule,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks - when using deprecated fields, only deprecated
				// fields are populated (not the tag block) to avoid plan diffs.
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule Deprecated"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_value", "Production"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_all_values", "false"),

				// Data source checks (by ID) - data sources always populate both styles
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule Deprecated"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag.#", "1"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_value", "Production"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule Deprecated"),
			),
		}},
	})
}

func TestAccPolarisTagRule_deprecatedAllValues(t *testing.T) {
	config, _, err := loadRSCTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	tagRule, err := makeTerraformConfig(config, tagRuleDeprecatedAllValuesTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: tagRule,
			Check: resource.ComposeTestCheckFunc(
				// Resource checks - when using deprecated fields, only deprecated
				// fields are populated (not the tag block) to avoid plan diffs.
				checkResourceAttrIsUUID("polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "name", "Test Tag Rule All Values"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "object_type", "AWS_EC2_INSTANCE"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("polaris_tag_rule.default", "tag_all_values", "true"),

				// Data source checks (by ID) - data sources always populate both styles
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_id", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "name", "Test Tag Rule All Values"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag.#", "1"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_key", "Environment"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_id", "tag_all_values", "true"),

				// Data source checks (by name)
				resource.TestCheckResourceAttrPair("data.polaris_tag_rule.default_by_name", "id", "polaris_tag_rule.default", "id"),
				resource.TestCheckResourceAttr("data.polaris_tag_rule.default_by_name", "name", "Test Tag Rule All Values"),
			),
		}},
	})
}
