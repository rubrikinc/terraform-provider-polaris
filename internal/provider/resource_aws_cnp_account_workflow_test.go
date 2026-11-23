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
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// TestAccPolarisAwsCnpAccountWorkflowNativeID onboards a new AWS account in
// one apply, with the role keys and permission policies available at plan time.
func TestAccPolarisAwsCnpAccountWorkflowNativeID(t *testing.T) {
	skipIfNotAcceptance(t)

	account, err := loadAWSTestConf()
	if err != nil {
		t.Fatal(err)
	}

	client, err := testClient(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	flag, err := core.Wrap(client.GQL).FeatureFlag(t.Context(), "REL_ENABLE_AWS_MANUAL_ONBOARDING_MULTI_ROLE")
	if err != nil {
		t.Fatalf("failed to read AWS manual onboarding feature flag: %v", err)
	}
	singleRoleCheck := knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("CROSSACCOUNT")})
	var roleKeyCheck knownvalue.Check = singleRoleCheck
	if flag.Enabled {
		// RSC may add other features, such as CLOUD_COST_REPORT.
		roleKeyCheck = knownvalue.SetPartial([]knownvalue.Check{
			knownvalue.StringExact("CLOUDACCOUNTS"),
			knownvalue.StringExact("CLOUD_DISCOVERY"),
			knownvalue.StringExact("CLOUD_NATIVE_PROTECTION"),
		})
	}

	// The test harness supports count but not for_each on data sources and
	// resources.
	conf := `
		variable "account_name" {
			type = string
		}
		variable "aws_account_id" {
			type = string
		}

		variable "aws_profile" {
			type = string
		}

		provider "aws" {
			profile = var.aws_profile
			region  = "us-east-2"
		}

		locals {
			features = {
				CLOUD_DISCOVERY         = ["BASIC"]
				CLOUD_NATIVE_PROTECTION = ["BASIC"]
			}
		}

		resource "polaris_aws_cnp_account" "account" {
			name      = var.account_name
			native_id = var.aws_account_id
			regions   = ["us-east-2"]

			dynamic "feature" {
				for_each = local.features
				content {
					name              = feature.key
					permission_groups = feature.value
				}
			}
		}

		data "polaris_aws_cnp_artifacts" "artifacts" {
			native_id = var.aws_account_id

			dynamic "feature" {
				for_each = local.features
				content {
					name              = feature.key
					permission_groups = feature.value
				}
			}
		}

		# Omitting native_id must preserve the legacy role layout even
		# on a tenant with multi-role onboarding enabled.
		data "polaris_aws_cnp_artifacts" "legacy" {
			dynamic "feature" {
				for_each = local.features
				content {
					name              = feature.key
					permission_groups = feature.value
				}
			}
		}

		locals {
			# Sorted so count indexes are stable.
			role_keys = sort(data.polaris_aws_cnp_artifacts.artifacts.role_keys)
			trust_policies = {
				for policy in polaris_aws_cnp_account.account.trust_policies : policy.role_key => policy.policy
			}

			# One entry per customer managed policy of each role.
			role_policies = flatten([
				for i, role_key in local.role_keys : [
					for policy in data.polaris_aws_cnp_permissions.permissions[i].customer_managed_policies : {
						role_index = i
						name       = policy.name
						policy     = policy.policy
					}
				]
			])
		}

		data "polaris_aws_cnp_permissions" "permissions" {
			count     = length(local.role_keys)
			native_id = var.aws_account_id
			role_key  = local.role_keys[count.index]

			dynamic "feature" {
				for_each = local.features
				content {
					name              = feature.key
					permission_groups = feature.value
				}
			}
		}

		resource "aws_iam_role" "role" {
			count              = length(local.role_keys)
			name_prefix        = "tfacc-polaris-nid-"
			assume_role_policy = local.trust_policies[local.role_keys[count.index]]

			# Role registration alone doesn't prove that the data source
			# returned policies for every role.
			lifecycle {
				precondition {
					condition     = length(data.polaris_aws_cnp_permissions.permissions[count.index].customer_managed_policies) > 0
					error_message = "no permissions for role ${local.role_keys[count.index]}"
				}
			}
		}

		resource "aws_iam_policy" "policy" {
			count       = length(local.role_policies)
			name_prefix = "tfacc-polaris-nid-${local.role_policies[count.index].name}-"
			policy      = local.role_policies[count.index].policy
		}

		resource "aws_iam_role_policy_attachment" "policy" {
			count      = length(local.role_policies)
			role       = aws_iam_role.role[local.role_policies[count.index].role_index].name
			policy_arn = aws_iam_policy.policy[count.index].arn
		}

		resource "polaris_aws_cnp_account_attachments" "attachments" {
			account_id = polaris_aws_cnp_account.account.id

			dynamic "role" {
				for_each = local.role_keys
				content {
					key         = role.value
					arn         = aws_iam_role.role[role.key].arn
					permissions = data.polaris_aws_cnp_permissions.permissions[role.key].id
				}
			}

			depends_on = [aws_iam_role_policy_attachment.policy]
		}

		# The roles of the artifacts data source and the roles the account
		# resource returns a trust policy for must be the same.
		output "role_keys_match" {
			value = (
				data.polaris_aws_cnp_artifacts.artifacts.role_keys ==
				toset(polaris_aws_cnp_account.account.trust_policies[*].role_key)
			)
		}
	`

	awsAPI := aws.Wrap(client)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: ">=6.0.0",
			},
		},
		PreCheck: func() {
			_, err := awsAPI.AccountByNativeID(t.Context(), account.AccountID)
			if err == nil {
				t.Fatalf("AWS account %q is already onboarded; this test requires a fresh account", account.AccountID)
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				t.Fatalf("failed to check the test AWS account: %v", err)
			}
		},
		CheckDestroy: func(_ *terraform.State) error {
			// Attachments share the lifecycle of the cloud account.
			_, err := awsAPI.AccountByNativeID(t.Context(), account.AccountID)
			if err == nil {
				return fmt.Errorf("AWS account %q still exists in RSC", account.AccountID)
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				return err
			}
			return nil
		},
		Steps: []resource.TestStep{{
			// Look up the role keys and policies before the account exists,
			// then create the account and register the roles in one apply.
			Config: conf,
			ConfigVariables: config.Variables{
				"account_name":   config.StringVariable(account.AccountName),
				"aws_account_id": config.StringVariable(account.AccountID),
				"aws_profile":    config.StringVariable(account.Profile),
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("role_keys_match", knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("data.polaris_aws_cnp_artifacts.artifacts",
					tfjsonpath.New(keyRoleKeys), roleKeyCheck),
				statecheck.ExpectKnownValue("data.polaris_aws_cnp_artifacts.artifacts",
					tfjsonpath.New(keyNativeID), knownvalue.StringExact(account.AccountID)),
				statecheck.ExpectKnownValue("data.polaris_aws_cnp_artifacts.legacy",
					tfjsonpath.New(keyRoleKeys), singleRoleCheck),
			},
		}},
	})
}
