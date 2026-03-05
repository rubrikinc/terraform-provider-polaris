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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
)

const (
	// The AWS credentials defaults to the default profile.
	rscConfigFileEnv         = "TEST_RSCCONFIG_FILE"
	awsAccountFileEnv        = "TEST_AWSACCOUNT_FILE"
	azureSubscriptionFileEnv = "TEST_AZURESUBSCRIPTION_FILE"
	azureCredentialsEnv      = "AZURE_SERVICEPRINCIPAL_LOCATION"
	gcpProjectFileEnv        = "TEST_GCPPROJECT_FILE"
	gcpCredentialsEnv        = "GOOGLE_APPLICATION_CREDENTIALS"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"polaris": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()

		sdkProviderV6, err := tf5to6server.UpgradeServer(ctx, sdkProvider.GRPCProvider)
		if err != nil {
			return nil, err
		}

		providers := []func() tfprotov6.ProviderServer{
			providerserver.NewProtocol6(&FrameworkProvider{Version: "test"}),
			func() tfprotov6.ProviderServer {
				return sdkProviderV6
			},
		}
		muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
		if err != nil {
			return nil, err
		}

		return muxServer.ProviderServer(), nil
	},
}

// terraformConfig returns a Terraform configuration given a test configuration
// and a Terraform template. Fails fatally if the template cannot be parsed or
// executed.
func terraformConfig(t *testing.T, config any, terraformTemplate string) string {
	tmpl, err := template.New("config").Parse(terraformTemplate)
	if err != nil {
		t.Fatalf("failed to parse terraform template: %s", err)
	}

	out := &strings.Builder{}
	if err := tmpl.Execute(out, config); err != nil {
		t.Fatalf("failed to execute terraform template: %s", err)
	}

	return out.String()
}

// testClient returns a Polaris client for testing outside the Terraform
// provider. E.g. checking if resources has been destroyed in a check destroy
// function.
func testClient() (*polaris.Client, error) {
	ctx := context.Background()

	// Looks for RSC credentials in standard environment variables. CacheParams
	// have sane default values.
	client, err := newClient(ctx, "", polaris.CacheParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to create test client: %s", err)
	}

	return client.polaris()
}

// testConfig2 holds the configuration for a test, i.e. the actual values to
// give to a Terraform template.
type testFullConfig struct {
	RSC   testRSCConfig
	AWS   testAWSAccount
	Azure testAzureSubscription
	GCP   testGCPProject
}

// testConf returns the full test configuration loaded by loadTestConf. Fails
// fatally if it can't be loaded.
func testConf(t *testing.T) testFullConfig {
	conf, err := loadTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return conf
}

// loadTestConf loads the full test configuration using the filenames pointed
// to by the individual test configuration environment variables.
func loadTestConf() (testFullConfig, error) {
	rsc, err := loadRSCTestConf()
	if err != nil {
		return testFullConfig{}, err
	}

	aws, err := loadAWSTestConf()
	if err != nil {
		return testFullConfig{}, err
	}

	azure, err := loadAzureTestConf()
	if err != nil {
		return testFullConfig{}, err
	}

	gcp, err := loadGCPTestConf()
	if err != nil {
		return testFullConfig{}, err
	}

	return testFullConfig{RSC: rsc, AWS: aws, Azure: azure, GCP: gcp}, nil
}

// rscTestConf returns the test configuration loaded by loadRSCTestConf. Fails
// fatally if it can't be loaded.
func rscTestConf(t *testing.T) testRSCConfig {
	rsc, err := loadRSCTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return rsc
}

// loadRSCTestConf loads the RSC test configuration using the filename pointed
// to by the TEST_RSCCONFIG_FILE environment variable.
func loadRSCTestConf() (testRSCConfig, error) {
	buf, err := os.ReadFile(os.Getenv(rscConfigFileEnv))
	if err != nil {
		return testRSCConfig{}, fmt.Errorf("failed to read file pointed to by %s: %s", rscConfigFileEnv, err)
	}

	rsc := testRSCConfig{}
	if err := json.Unmarshal(buf, &rsc); err != nil {
		return testRSCConfig{}, fmt.Errorf("failed to unmarshal RSC config file: %s", err)
	}

	return rsc, nil
}

// awsTestConf returns the test configuration loaded by loadAWSTestConf. Fails
// fatally if it can't be loaded.
func awsTestConf(t *testing.T) testAWSAccount {
	account, err := loadAWSTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return account
}

// loadAWSTestConf loads the AWS test configuration using the filename pointed
// to by the TEST_AWSACCOUNT_FILE environment variables.
func loadAWSTestConf() (testAWSAccount, error) {
	buf, err := os.ReadFile(os.Getenv(awsAccountFileEnv))
	if err != nil {
		return testAWSAccount{}, fmt.Errorf("failed to read file pointed to by %s: %s", awsAccountFileEnv, err)
	}

	account := testAWSAccount{}
	if err := json.Unmarshal(buf, &account); err != nil {
		return testAWSAccount{}, fmt.Errorf("failed to unmarshal AWS config file: %s", err)
	}
	if account.Profile == "" {
		account.Profile = "default"
	}

	return account, nil
}

// azureTestConf returns the test configuration loaded by loadAzureTestConf.
// Fails fatally if it can't be loaded.
func azureTestConf(t *testing.T) testAzureSubscription {
	subscription, err := loadAzureTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return subscription
}

// loadAzureTestConf loads the Azure test configuration using the filename
// pointed to by the AzureSubscriptionFileEnv environment variables.
func loadAzureTestConf() (testAzureSubscription, error) {
	buf, err := os.ReadFile(os.Getenv(azureSubscriptionFileEnv))
	if err != nil {
		return testAzureSubscription{}, fmt.Errorf("failed to read file pointed to by %s: %s", azureSubscriptionFileEnv, err)
	}

	subscription := testAzureSubscription{}
	if err := json.Unmarshal(buf, &subscription); err != nil {
		return testAzureSubscription{}, fmt.Errorf("failed to unmarshal Azure config file: %s", err)
	}
	if subscription.Credentials == "" {
		subscription.Credentials = os.Getenv(azureCredentialsEnv)
	}

	return subscription, nil
}

// gcpTestConf returns the test configuration loaded by loadGCPTestConf. Fails
// fatally if it can't be loaded.
func gcpTestConf(t *testing.T) testGCPProject {
	project, err := loadGCPTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return project
}

// loadGCPTestConf loads the GCP test configuration using the filename pointed
// to by the GCPProjectFileEnv environment variables.
func loadGCPTestConf() (testGCPProject, error) {
	buf, err := os.ReadFile(os.Getenv(gcpProjectFileEnv))
	if err != nil {
		return testGCPProject{}, fmt.Errorf("failed to read file pointed to by %s: %s", gcpProjectFileEnv, err)
	}

	project := testGCPProject{}
	if err := json.Unmarshal(buf, &project); err != nil {
		return testGCPProject{}, fmt.Errorf("failed to unmarshal GCP config file: %s", err)
	}
	if project.Credentials == "" {
		project.Credentials = os.Getenv(gcpCredentialsEnv)
	}

	return project, nil
}
