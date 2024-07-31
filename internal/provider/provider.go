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
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
				Type:     schema.TypeString,
				Optional: true,
				Description: "The service account credentials, service account credentials file name or local user " +
					"account name to use when accessing RSC.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTokenCache: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable or disable the token cache. The token cache is enabled by default.",
			},
			keyTokenCacheDir: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The directory where cached authentication tokens are stored. The OS directory for " +
					"temporary files is used by default.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTokenCacheSecret: {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				Description: "The secret used as input when generating an encryption key for the authentication " +
					"token. The encryption key is derived from the RSC account information by default.",
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
			keyPolarisAWSAccount:            dataSourceAwsAccount(),
			keyPolarisAWSArchivalLocation:   dataSourceAwsArchivalLocation(),
			keyPolarisAWSCNPArtifacts:       dataSourceAwsArtifacts(),
			keyPolarisAWSCNPPermissions:     dataSourceAwsPermissions(),
			keyPolarisAzureArchivalLocation: dataSourceAzureArchivalLocation(),
			keyPolarisAzurePermissions:      dataSourceAzurePermissions(),
			keyPolarisAzureSubscription:     dataSourceAzureSubscription(),
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

func newClient(account polaris.Account, params polaris.CacheParams, logger log.Logger) (*client, diag.Diagnostics) {
	polarisClient, err := polaris.NewClientWithLoggerAndCacheParams(account, params, logger)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return &client{
		cdmClient:     cdm.NewBootstrapClientWithLogger(true, logger),
		polarisClient: polarisClient,
	}, nil
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

	account, err := polaris.FindAccount(d.Get("credentials").(string), true)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	cacheParams := polaris.CacheParams{
		Enable: d.Get("token_cache").(bool),
		Dir:    d.Get("token_cache_dir").(string),
		Secret: d.Get("token_cache_secret").(string),
	}
	return newClient(account, cacheParams, logger)
}

// description returns the description string with all acute accents replaced
// with grave accents (backticks).
func description(description string) string {
	return strings.ReplaceAll(description, "´", "`")
}
