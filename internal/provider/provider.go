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
	"errors"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/cdm"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/log"
)

const (
	appCloudAccountPrefix = "app-"
)

// Provider defines the schema and resource map for the RSC provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			keyCredentials: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The service account file or local user account name to use when accessing RSC.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			keyPolarisAWSAccount:                     resourceAwsAccount(),
			keyPolarisAWSArchivalLocation:            resourceAwsArchivalLocation(),
			keyPolarisAWSCNPAccount:                  resourceAwsCnpAccount(),
			keyPolarisAWSCNPAccountAttachments:       resourceAwsCnpAccountAttachments(),
			keyPolarisAWSCNPAccountTrustPolicy:       resourceAwsCnpAccountTrustPolicy(),
			keyPolarisAWSExocompute:                  resourceAwsExocompute(),
			keyPolarisAWSExocomputeClusterAttachment: resourceAwsExocomputeClusterAttachment(),
			keyPolarisAWSPrivateContainerRegistry:    resourceAwsPrivateContainerRegistry(),
			keyPolarisAzureArchivalLocation:          resourceAzureArchivalLocation(),
			keyPolarisAzureExocompute:                resourceAzureExocompute(),
			keyPolarisAzureServicePrincipal:          resourceAzureServicePrincipal(),
			keyPolarisAzureSubscription:              resourceAzureSubscription(),
			"polaris_cdm_bootstrap":                  resourceCDMBootstrap(),
			"polaris_cdm_bootstrap_cces_aws":         resourceCDMBootstrapCCESAWS(),
			"polaris_cdm_bootstrap_cces_azure":       resourceCDMBootstrapCCESAzure(),
			keyPolarisCustomRole:                     resourceCustomRole(),
			"polaris_gcp_project":                    resourceGcpProject(),
			"polaris_gcp_service_account":            resourceGcpServiceAccount(),
			keyPolarisRoleAssignment:                 resourceRoleAssignment(),
			keyPolarisUser:                           resourceUser(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			keyPolarisAccount:               dataSourceAccount(),
			keyPolarisAWSArchivalLocation:   dataSourceAwsArchivalLocation(),
			keyPolarisAWSCNPArtifacts:       dataSourceAwsArtifacts(),
			keyPolarisAWSCNPPermissions:     dataSourceAwsPermissions(),
			keyPolarisAzureArchivalLocation: dataSourceAzureArchivalLocation(),
			keyPolarisAzurePermissions:      dataSourceAzurePermissions(),
			keyPolarisDeployment:            dataSourceDeployment(),
			keyPolarisFeatures:              dataSourceFeatures(),
			"polaris_gcp_permissions":       dataSourceGcpPermissions(),
			keyPolarisRole:                  dataSourceRole(),
			keyPolarisRoleTemplate:          dataSourceRoleTemplate(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

type client struct {
	cdmClient     *cdm.BootstrapClient
	polarisClient *polaris.Client
}

func (c *client) cdm() (*cdm.BootstrapClient, error) {
	if c.cdmClient == nil {
		return nil, errors.New("cdm functionality has not been configured in the provider block")
	}

	return c.cdmClient, nil
}

func (c *client) polaris() (*polaris.Client, error) {
	if c.polarisClient == nil {
		return nil, errors.New("polaris functionality has not been configured in the provider block")
	}

	return c.polarisClient, nil
}

// providerConfigure configures the RSC provider.
func providerConfigure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	logger := log.NewStandardLogger()
	if err := polaris.SetLogLevelFromEnv(logger); err != nil {
		return nil, diag.FromErr(err)
	}

	client := &client{
		cdmClient: cdm.NewBootstrapClientWithLogger(true, logger),
	}

	var account polaris.Account
	if c, ok := d.GetOk(keyCredentials); ok {
		credentials := c.(string)

		// When credentials refer to an existing file, we load the file as a
		// service account, otherwise we assume that it's a user account name.
		if _, err := os.Stat(credentials); err == nil {
			if account, err = polaris.ServiceAccountFromFile(credentials, true); err != nil {
				return nil, diag.FromErr(err)
			}
		} else {
			if account, err = polaris.DefaultUserAccount(credentials, true); err != nil {
				return nil, diag.FromErr(err)
			}
		}
	} else {
		var err error
		if account, err = polaris.ServiceAccountFromEnv(); err != nil {
			if !errors.Is(err, graphql.ErrNotFound) {
				return nil, diag.FromErr(err)
			}

			// Make sure the interface value is an untyped nil, see SA4023 for
			// details.
			account = nil
		}
	}
	if account != nil {
		polarisClient, err := polaris.NewClientWithLogger(account, logger)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		client.polarisClient = polarisClient
	}

	return client, nil
}

// description returns the description string with all acute accents replaced
// with grave accents (backticks).
func description(description string) string {
	return strings.ReplaceAll(description, "´", "`")
}
