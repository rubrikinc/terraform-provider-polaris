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
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var provider *schema.Provider = Provider()

var providerFactories = map[string]func() (*schema.Provider, error){
	"polaris": func() (*schema.Provider, error) {
		return provider, nil
	},
}

const credentialsFromEnv = `
provider "polaris" {
}

data "polaris_role" "admin" {
  name = "Administrator"
}
`

func TestAccProviderCredentialsInEnv_basic(t *testing.T) {
	credentials, err := loadTestCredentials("RUBRIK_POLARIS_SERVICEACCOUNT_FILE")
	if err != nil {
		t.Fatal(err)
	}

	// Valid service account in RUBRIK_POLARIS_SERVICEACCOUNT_FILE.
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: credentialsFromEnv,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("data.polaris_role.admin", "id", "00000000-0000-0000-0000-000000000000"),
				resource.TestCheckResourceAttr("data.polaris_role.admin", "name", "Administrator"),
			),
		}},
	})

	// Non-existing service account in RUBRIK_POLARIS_SERVICEACCOUNT_FILE.
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "03147711-359c-40fd-b635-69619fcf374d")
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config:      credentialsFromEnv,
			ExpectError: regexp.MustCompile("RSC functionality has not been configured: account not found, searched: default service account file and env"),
		}},
	})

	// Valid service account in RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS.
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "")
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS", credentials)
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: credentialsFromEnv,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("data.polaris_role.admin", "id", "00000000-0000-0000-0000-000000000000"),
				resource.TestCheckResourceAttr("data.polaris_role.admin", "name", "Administrator"),
			),
		}},
	})

	// Invalid service account in RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS.
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS", "invalid")
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config:      credentialsFromEnv,
			ExpectError: regexp.MustCompile("RSC functionality has not been configured: account not found, searched: default service account file and env"),
		}},
	})

	// Partial service account in env. This could happen if the service account
	// is given in parts and one of the parts is missing.
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_CREDENTIALS", "")
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_NAME", "name")
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config:      credentialsFromEnv,
			ExpectError: regexp.MustCompile("invalid service account client id"),
		}},
	})

	// No service account in env. This could happen if the provider is used to
	// bootstrap a CDM cluster without RSC credentials, but an RSC resource is
	// used.
	t.Setenv("RUBRIK_POLARIS_SERVICEACCOUNT_NAME", "")
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config:      credentialsFromEnv,
			ExpectError: regexp.MustCompile("RSC functionality has not been configured: account not found, searched: default service account file and env"),
		}},
	})
}

// loadTestCredentials returns the content of the file pointed to by the
// credentialsEnv parameter.
func loadTestCredentials(credentialsEnv string) (string, error) {
	credentials := os.Getenv(credentialsEnv)
	if credentials == "" {
		return "", fmt.Errorf("%s is empty", credentialsEnv)
	}

	buf, err := os.ReadFile(credentials)
	if err != nil {
		return "", fmt.Errorf("failed to read file pointed to by %s: %v", credentialsEnv, err)
	}

	return string(buf), nil
}

// testConfig holds the configuration for a test, i.e. the actual values to
// give to a Terraform template.
type testConfig struct {
	Provider struct {
		Credentials string
	}
	Resource interface{}
}

// loadTestConfig returns a new testConfig initialized from the file pointed
// to by the environmental variable in resourceFileEnv. Note that it must be
// possible to unmarshal the file to the resource type and that resource must
// be of pointer type.
func loadTestConfig(credentialsEnv, resourceFileEnv string, resource interface{}) (testConfig, error) {
	credentials := os.Getenv(credentialsEnv)
	if credentials == "" {
		return testConfig{}, fmt.Errorf("%s is empty", credentialsEnv)
	}

	buf, err := os.ReadFile(os.Getenv(resourceFileEnv))
	if err != nil {
		return testConfig{}, fmt.Errorf("failed to read file pointed to by %s: %v", resourceFileEnv, err)
	}

	if err := json.Unmarshal(buf, resource); err != nil {
		return testConfig{}, err
	}

	config := testConfig{
		Provider: struct{ Credentials string }{
			Credentials: credentials,
		},
		Resource: resource,
	}

	return config, nil
}

// makeTerraformConfig returns a Terraform configuration given a test
// configuration and a Terraform template.
func makeTerraformConfig(config testConfig, terraformTemplate string) (string, error) {
	tmpl, err := template.New("resource").Parse(terraformTemplate)
	if err != nil {
		return "", err
	}

	out := &strings.Builder{}
	if err := tmpl.Execute(out, config); err != nil {
		return "", err
	}

	return out.String(), nil
}

// testAWSAccount holds information about an AWS account used in one or more
// acceptance tests.
type testAWSAccount struct {
	Profile          string `json:"profile"`
	AccountID        string `json:"accountId"`
	AccountName      string `json:"accountName"`
	CrossAccountID   string `json:"crossAccountId"`
	CrossAccountName string `json:"crossAccountName"`
	CrossAccountRole string `json:"crossAccountRole"`

	Exocompute struct {
		VPCID   string `json:"vpcId"`
		Subnets []struct {
			ID               string `json:"id"`
			AvailabilityZone string `json:"availabilityZone"`
		} `json:"subnets"`
	} `json:"exocompute"`
}

// loadAWSTestConfig loads an AWS test configuration using the default
// environment variables.
func loadAWSTestConfig() (testConfig, testAWSAccount, error) {
	account := testAWSAccount{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_AWSACCOUNT_FILE", &account)

	// Note that this will update both project and config.
	if account.Profile == "" {
		account.Profile = "default"
	}

	return config, account, err
}

// testAzureSubscription holds information about an Azure subscription used in
// one or more acceptance tests.
type testAzureSubscription struct {
	Credentials      string `json:"credentials"`
	SubscriptionID   string `json:"subscriptionId"`
	SubscriptionName string `json:"subscriptionName"`
	TenantID         string `json:"tenantId"`
	TenantDomain     string `json:"tenantDomain"`
	PrincipalID      string `json:"principalId"`
	PrincipalName    string `json:"principalName"`
	PrincipalSecret  string `json:"principalSecret"`

	CloudNativeProtection struct {
		Regions             []string `json:"regions"`
		ResourceGroupName   string   `json:"resourceGroupName"`
		ResourceGroupRegion string   `json:"resourceGroupRegion"`
	} `json:"cloudNativeProtection"`

	Exocompute struct {
		Regions             []string `json:"regions"`
		ResourceGroupName   string   `json:"resourceGroupName"`
		ResourceGroupRegion string   `json:"resourceGroupRegion"`
		SubnetID            string   `json:"subnetId"`
	} `json:"exocompute"`
}

// loadAzureTestConfig loads an Azure test configuration using the default
// environment variables.
func loadAzureTestConfig() (testConfig, testAzureSubscription, error) {
	subscription := testAzureSubscription{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_AZURESUBSCRIPTION_FILE", &subscription)

	if subscription.Credentials == "" {
		subscription.Credentials = os.Getenv("AZURE_SERVICEPRINCIPAL_LOCATION")
	}

	return config, subscription, err
}

// testGCPProject holds information about a GCP project used in one or more
// acceptance tests.
type testGCPProject struct {
	Credentials      string `json:"credentials"`
	ProjectID        string `json:"projectId"`
	ProjectName      string `json:"projectName"`
	ProjectNumber    int64  `json:"projectNumber"`
	OrganizationName string `json:"organizationName"`
}

// loadGCPTestConfig loads a GCP test configuration using the default
// environment variables.
func loadGCPTestConfig() (testConfig, testGCPProject, error) {
	project := testGCPProject{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_GCPPROJECT_FILE", &project)

	// Note that this will update both project and config.
	if project.Credentials == "" {
		project.Credentials = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	return config, project, err
}

// testRSCConfig holds RSC configuration information used in one or more
// acceptance tests.
type testRSCConfig struct {
	ExistingUserEmail string `json:"existingUserEmail"`
	NewUserEmail      string `json:"newUserEmail"`
}

// loadRSCTestConfig loads an RSC test configuration using the default
// environment variables.
func loadRSCTestConfig() (testConfig, testRSCConfig, error) {
	rsc := testRSCConfig{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_RSCCONFIG_FILE", &rsc)

	return config, rsc, err
}
