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
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const objectAWSAccountTmpl = `
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
}

data "polaris_object" "aws_account" {
	name        = "{{ .Resource.AccountName }}"
	object_type = "AwsNativeAccount"

	depends_on = [polaris_aws_account.default]
}
`

func TestAccPolarisObject_awsAccount(t *testing.T) {
	config, account, err := loadAWSTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	objectAWSAccount, err := makeTerraformConfig(config, objectAWSAccountTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: objectAWSAccount,
			Check: resource.ComposeTestCheckFunc(
				// Verify the AWS account resource was created
				resource.TestCheckResourceAttr("polaris_aws_account.default", "name", account.AccountName),
				resource.TestCheckResourceAttr("polaris_aws_account.default", "cloud_native_protection.0.status", "connected"),

				// Verify the object data source returns the correct values
				resource.TestCheckResourceAttrSet("data.polaris_object.aws_account", "id"),
				resource.TestCheckResourceAttr("data.polaris_object.aws_account", "name", account.AccountName),
				resource.TestCheckResourceAttr("data.polaris_object.aws_account", "object_type", "AwsNativeAccount"),
			),
		}},
	})
}

const objectAzureSubscriptionTmpl = `
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

	depends_on = [polaris_azure_service_principal.default]
}

data "polaris_object" "azure_subscription" {
	name        = "{{ .Resource.SubscriptionName }}"
	object_type = "AzureNativeSubscription"

	depends_on = [polaris_azure_subscription.default]
}
`

func TestAccPolarisObject_azureSubscription(t *testing.T) {
	config, subscription, err := loadAzureTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	objectAzureSubscription, err := makeTerraformConfig(config, objectAzureSubscriptionTmpl)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: objectAzureSubscription,
			Check: resource.ComposeTestCheckFunc(
				// Verify the Azure subscription resource was created
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "subscription_name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("polaris_azure_subscription.default", "cloud_native_protection.0.status", "CONNECTED"),

				// Verify the object data source returns the correct values
				resource.TestCheckResourceAttrSet("data.polaris_object.azure_subscription", "id"),
				resource.TestCheckResourceAttr("data.polaris_object.azure_subscription", "name", subscription.SubscriptionName),
				resource.TestCheckResourceAttr("data.polaris_object.azure_subscription", "object_type", "AzureNativeSubscription"),
			),
		}},
	})
}

func TestValidateObjectConfig(t *testing.T) {
	tests := []struct {
		name   string
		config objectModel
		// wantErrPaths lists the attribute paths that should produce a
		// validation error, in any order. Empty means the config is valid.
		wantErrPaths []string
	}{
		{
			name: "NoParentIDs",
			config: objectModel{
				ObjectType: types.StringValue("AwsNativeEc2Instance"),
			},
		},
		{
			name: "UnknownObjectTypeSkipsValidation",
			config: objectModel{
				ObjectType:     types.StringUnknown(),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
				OrgID:          types.StringValue("550e8400-e29b-41d4-a716-446655440001"),
				ProjectID:      types.StringValue("550e8400-e29b-41d4-a716-446655440002"),
			},
		},
		{
			name: "NullObjectTypeSkipsValidation",
			config: objectModel{
				ObjectType:     types.StringNull(),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
		},
		{
			name: "SubscriptionIDWithResourceGroup",
			config: objectModel{
				ObjectType:     types.StringValue("AzureNativeResourceGroup"),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
		},
		{
			name: "SubscriptionIDWithWrongType",
			config: objectModel{
				ObjectType:     types.StringValue("AwsNativeEc2Instance"),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
			wantErrPaths: []string{keySubscriptionID},
		},
		{
			name: "OrgIDWithAzureDevOpsProject",
			config: objectModel{
				ObjectType: types.StringValue("AzureDevOpsProject"),
				OrgID:      types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
		},
		{
			name: "OrgIDWithGitHubRepository",
			config: objectModel{
				ObjectType: types.StringValue("GitHubRepository"),
				OrgID:      types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
		},
		{
			name: "OrgIDWithWrongType",
			config: objectModel{
				ObjectType: types.StringValue("GitHubOrganization"),
				OrgID:      types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
			wantErrPaths: []string{keyOrgID},
		},
		{
			name: "ProjectIDWithAzureDevOpsRepository",
			config: objectModel{
				ObjectType: types.StringValue("AzureDevOpsRepository"),
				ProjectID:  types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
		},
		{
			name: "ProjectIDWithWrongType",
			config: objectModel{
				ObjectType: types.StringValue("AzureDevOpsProject"),
				ProjectID:  types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			},
			wantErrPaths: []string{keyProjectID},
		},
		{
			// A resource group accepts subscription_id but not org_id, so
			// setting both flags only org_id.
			name: "ResourceGroupWithSubscriptionAndOrgID",
			config: objectModel{
				ObjectType:     types.StringValue("AzureNativeResourceGroup"),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
				OrgID:          types.StringValue("550e8400-e29b-41d4-a716-446655440001"),
			},
			wantErrPaths: []string{keyOrgID},
		},
		{
			name: "UnknownParentIDSkipped",
			config: objectModel{
				ObjectType:     types.StringValue("AwsNativeEc2Instance"),
				SubscriptionID: types.StringUnknown(),
			},
		},
		{
			name: "MultipleViolations",
			config: objectModel{
				ObjectType:     types.StringValue("AwsNativeEc2Instance"),
				SubscriptionID: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
				OrgID:          types.StringValue("550e8400-e29b-41d4-a716-446655440001"),
				ProjectID:      types.StringValue("550e8400-e29b-41d4-a716-446655440002"),
			},
			wantErrPaths: []string{keySubscriptionID, keyOrgID, keyProjectID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateObjectConfig(tc.config)

			var gotErrPaths []string
			for _, d := range diags.Errors() {
				dp, ok := d.(diag.DiagnosticWithPath)
				if !ok {
					t.Errorf("error diagnostic has no path: %s: %s", d.Summary(), d.Detail())
					continue
				}
				gotErrPaths = append(gotErrPaths, dp.Path().String())
			}

			var wantErrPaths []string
			for _, key := range tc.wantErrPaths {
				wantErrPaths = append(wantErrPaths, path.Root(key).String())
			}

			slices.Sort(gotErrPaths)
			slices.Sort(wantErrPaths)
			if !slices.Equal(gotErrPaths, wantErrPaths) {
				t.Errorf("error paths mismatch:\n got: %v\nwant: %v", gotErrPaths, wantErrPaths)
			}
		})
	}
}
