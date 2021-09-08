package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var providerFactories = map[string]func() (*schema.Provider, error){
	"polaris": func() (*schema.Provider, error) {
		return Provider(), nil
	},
}

// testConfig holds the configuration for a test, i.e. the actaul values to
// give to a terraform template.
type testConfig struct {
	Provider struct {
		Credentials string
	}
	Resource interface{}
}

// loadTestConfig returns a new testConfig initialized from the file pointed
// to by the environmental variable in resourceFileEnv. Note that it must be
// possible to unmarshal the file to resource and that resource must be of
// pointer type.
func loadTestConfig(credentialsEnv, resourceFileEnv string, resource interface{}) (testConfig, error) {
	credentials := os.Getenv(credentialsEnv)
	if credentials == "" {
		return testConfig{}, fmt.Errorf("%s is empty", credentialsEnv)
	}

	buf, err := os.ReadFile(os.Getenv(resourceFileEnv))
	if err != nil {
		return testConfig{}, fmt.Errorf("failed to read file pointed to %s: %v", resourceFileEnv, err)
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
// configuration and a terraform template.
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
	Profile     string `json:"profile"`
	AccountID   string `json:"accountId"`
	AccountName string `json:"name"`
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
	SubscriptionName string `json:"name"`
	TenantID         string `json:"tenantId"`
	TenantDomain     string `json:"tenantDomain"`
	PrincipalID      string `json:"principalId"`
	PrincipalName    string `json:"principalName"`
	PrincipalSecret  string `json:"principalSecret"`
}

// loadAzureTestConfig loads an Azure test configuration using the default
// environment variables.
func loadAzureTestConfig() (testConfig, testAzureSubscription, error) {
	subscription := testAzureSubscription{}
	config, err := loadTestConfig("RUBRIK_POLARIS_SERVICEACCOUNT_FILE", "TEST_AZURESUBSCRIPTION_FILE", &subscription)

	if subscription.Credentials == "" {
		subscription.Credentials = os.Getenv("AZURE_SERVICEPRINCIPAL_LOCATION")
	}

	os.Getenv("AZURE_SUBSCRIPTION_LOCATION")

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
