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
	"maps"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

var azureExocomputeTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_azure_service_principal" "default" {
	credentials   = "{{ .Resource.Credentials }}"
	tenant_domain = "{{ .Resource.TenantDomain }}"
}

resource "polaris_azure_subscription" "default" {
	subscription_id   = "{{ .Resource.SubscriptionID }}"
	subscription_name = "{{ .Resource.SubscriptionName }}"
	tenant_domain     = "{{ .Resource.TenantDomain }}"

	cloud_native_protection {
		resource_group_name   = "{{ .Resource.CloudNativeProtection.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.CloudNativeProtection.ResourceGroupRegion }}"

		regions = [
			"eastus2",
		]
	}

	exocompute {
		resource_group_name   = "{{ .Resource.Exocompute.ResourceGroupName }}"
		resource_group_region = "{{ .Resource.Exocompute.ResourceGroupRegion }}"

		regions = [
			"eastus2",
		]
	}

	depends_on = [polaris_azure_service_principal.default]
}
  
resource "polaris_azure_exocompute" "default" {
	subscription_id = polaris_azure_subscription.default.id
	region          = "eastus2"
	subnet          = "{{ .Resource.Exocompute.SubnetID }}"
}  
`

func TestAccPolarisAzureExocompute_basic(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	exocompute, err := makeTerraformConfig(config, azureExocomputeTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: exocompute,
			Check: resource.ComposeTestCheckFunc(
				// Subscription resource
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_id", subscription.SubscriptionID),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "tenant_domain", subscription.TenantDomain),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "delete_snapshots_on_destroy", "false"),

				// Cloud Native Protection feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "cloud_native_protection.0.regions.*", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_name",
					subscription.CloudNativeProtection.ResourceGroupName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.resource_group_region",
					subscription.CloudNativeProtection.ResourceGroupRegion),

				// Exocompute feature
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.status", "CONNECTED"),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "exocompute.0.regions.#", "1"),
				resource.TestCheckTypeSetElemAttr("polaris_azure_subscription.default", "exocompute.0.regions.*", "eastus2"),

				// Exocompute resource
				resource.TestCheckResourceAttrPair("polaris_azure_exocompute.default", "subscription_id", "polaris_azure_subscription.default", "id"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "region", "eastus2"),
				resource.TestCheckResourceAttr("polaris_azure_exocompute.default", "subnet", subscription.Exocompute.SubnetID),
			),
		}},
	})
}

// TestAzureExocomputeOptionalConfigDiff verifies that the optional_config
// fields added after v1.11.2 keep the value stored in RSC when they are not
// set, and that changing one of them replaces the Exocompute configuration.
func TestAzureExocomputeOptionalConfigDiff(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	const cloudAccountID = "550e8400-e29b-41d4-a716-446655440001"

	// The new fields hold values configured in RSC, e.g. using the RSC UI.
	state := map[string]string{
		"id":                               id,
		"cloud_account_id":                 cloudAccountID,
		"region":                           "eastus2",
		"subnet":                           "subnet",
		"optional_config.#":                "1",
		"optional_config.0.cluster_access": "AKS_CLUSTER_ACCESS_TYPE_PRIVATE",
		"optional_config.0.cluster_tier":   "AKS_CLUSTER_TIER_FREE",
		"optional_config.0.max_node_count": "AKS_NODE_COUNT_BUCKET_MEDIUM",
		"optional_config.0.disk_encryption_set_id":       "des",
		"optional_config.0.health_check_vm_name_prefix":  "prefix",
		"optional_config.0.postgres_private_dns_zone_id": "postgres-zone",
		"optional_config.0.postgres_subnet_id":           "postgres-subnet",
		"optional_config.0.sql_private_dns_zone_id":      "sql-zone",
	}

	testCases := []struct {
		name            string
		optionalConfig  map[string]any
		wantRequiresNew bool
		wantRequiresKey string
	}{{
		name:           "UnsetFieldsKeepTheStoredValues",
		optionalConfig: map[string]any{"cluster_access": "AKS_CLUSTER_ACCESS_TYPE_PRIVATE"},
	}, {
		name: "SameValuesAreNoChange",
		optionalConfig: map[string]any{
			"disk_encryption_set_id":       "des",
			"health_check_vm_name_prefix":  "prefix",
			"postgres_private_dns_zone_id": "postgres-zone",
			"postgres_subnet_id":           "postgres-subnet",
			"sql_private_dns_zone_id":      "sql-zone",
		},
	}, {
		name:            "ChangedPrefixReplaces",
		optionalConfig:  map[string]any{"health_check_vm_name_prefix": "other"},
		wantRequiresNew: true,
		wantRequiresKey: "optional_config.0.health_check_vm_name_prefix",
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := resourceAzureExocompute().Diff(context.Background(),
				&terraform.InstanceState{ID: id, Attributes: state},
				terraform.NewResourceConfigRaw(map[string]any{
					"cloud_account_id": cloudAccountID,
					"region":           "eastus2",
					"subnet":           "subnet",
					"optional_config":  []any{tc.optionalConfig},
				}), nil)
			if err != nil {
				t.Fatal(err)
			}

			if !tc.wantRequiresNew {
				if diff != nil && !diff.Empty() {
					t.Fatalf("expected no diff, got: %v", diff)
				}
				return
			}
			if diff == nil || !diff.RequiresNew() {
				t.Fatalf("expected the resource to be replaced, diff: %v", diff)
			}
			if !diff.Attributes[tc.wantRequiresKey].RequiresNew {
				t.Errorf("expected %s to require a new resource", tc.wantRequiresKey)
			}
		})
	}
}

// TestExocomputeRunHealthCheckCustomerManaged verifies that no health check is
// triggered for a customer managed Exocompute configuration, and that this is
// reported as a warning. The client is nil, so any RSC call would panic.
func TestExocomputeRunHealthCheckCustomerManaged(t *testing.T) {
	diags := exocomputeRunHealthCheck(context.Background(), nil, core.CloudVendorAzure,
		"550e8400-e29b-41d4-a716-446655440000", true)
	if len(diags) != 1 || diags[0].Severity != diag.Warning {
		t.Fatalf("expected a single warning, got: %v", diags)
	}
}

// TestExocomputeTriggerHealthCheckDiff verifies that trigger_health_check on
// the AWS and Azure Exocompute resources is updated in place, so that changing
// it never replaces the Exocompute configuration, and that state created
// before the field existed has no diff when the field is not set.
func TestExocomputeTriggerHealthCheckDiff(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	const cloudAccountID = "550e8400-e29b-41d4-a716-446655440001"

	resources := []struct {
		name     string
		resource *schema.Resource
		state    map[string]string
		config   map[string]any
	}{{
		name:     "AWS",
		resource: resourceAwsExocompute(),
		state: map[string]string{
			"id":                        id,
			"account_id":                cloudAccountID,
			"region":                    "us-east-2",
			"vpc_id":                    "vpc",
			"cluster_access":            "",
			"cluster_security_group_id": "",
			"node_security_group_id":    "",
			"polaris_managed":           "true",
			"subnet.#":                  "0",
			"subnets.#":                 "2",
			"subnets." + strconv.Itoa(schema.HashString("subnet-1")): "subnet-1",
			"subnets." + strconv.Itoa(schema.HashString("subnet-2")): "subnet-2",
		},
		config: map[string]any{
			"account_id": cloudAccountID,
			"region":     "us-east-2",
			"vpc_id":     "vpc",
			"subnets":    []any{"subnet-1", "subnet-2"},
		},
	}, {
		name:     "Azure",
		resource: resourceAzureExocompute(),
		state: map[string]string{
			"id":               id,
			"cloud_account_id": cloudAccountID,
			"region":           "eastus2",
			"subnet":           "subnet",
		},
		config: map[string]any{
			"cloud_account_id": cloudAccountID,
			"region":           "eastus2",
			"subnet":           "subnet",
		},
	}}

	testCases := []struct {
		name        string
		stateValue  string
		configValue any
		wantUpdate  bool
	}{{
		name: "UnsetInStateAndConfigIsNoChange",
	}, {
		name:        "UnsetToTrueUpdatesInPlace",
		configValue: true,
		wantUpdate:  true,
	}, {
		name:        "TrueToFalseUpdatesInPlace",
		stateValue:  "true",
		configValue: false,
		wantUpdate:  true,
	}}
	for _, r := range resources {
		for _, tc := range testCases {
			t.Run(r.name+tc.name, func(t *testing.T) {
				state := maps.Clone(r.state)
				if tc.stateValue != "" {
					state["trigger_health_check"] = tc.stateValue
				}
				config := maps.Clone(r.config)
				if tc.configValue != nil {
					config["trigger_health_check"] = tc.configValue
				}

				diff, err := r.resource.Diff(context.Background(), &terraform.InstanceState{ID: id, Attributes: state},
					terraform.NewResourceConfigRaw(config), nil)
				if err != nil {
					t.Fatal(err)
				}

				if !tc.wantUpdate {
					if diff != nil && !diff.Empty() {
						t.Fatalf("expected no diff, got: %v", diff)
					}
					return
				}
				if diff == nil || diff.Attributes["trigger_health_check"] == nil {
					t.Fatalf("expected trigger_health_check to change, diff: %v", diff)
				}
				if diff.RequiresNew() {
					t.Fatalf("expected an in-place update, diff: %v", diff)
				}
			})
		}
	}
}
