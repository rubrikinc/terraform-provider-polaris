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
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
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
			keyPolarisAWSAccount:                         resourceAwsAccount(),
			keyPolarisAWSArchivalLocation:                resourceAwsArchivalLocation(),
			keyPolarisAWSCloudCluster:                    resourceAwsCloudCluster(),
			keyPolarisAWSCNPAccount:                      resourceAwsCnpAccount(),
			keyPolarisAWSCNPAccountAttachments:           resourceAwsCnpAccountAttachments(),
			keyPolarisAWSCNPAccountTrustPolicy:           resourceAwsCnpAccountTrustPolicy(),
			keyPolarisAWSCustomTags:                      resourceAwsCustomTags(),
			keyPolarisAWSExocompute:                      resourceAwsExocompute(),
			keyPolarisAWSExocomputeClusterAttachment:     resourceAwsExocomputeClusterAttachment(),
			keyPolarisAWSPrivateContainerRegistry:        resourceAwsPrivateContainerRegistry(),
			keyPolarisAzureArchivalLocation:              resourceAzureArchivalLocation(),
			keyPolarisAzureCloudCluster:                  resourceAzureCloudCluster(),
			keyPolarisAzureCustomTags:                    resourceAzureCustomTags(),
			keyPolarisAzureExocompute:                    resourceAzureExocompute(),
			keyPolarisAzureExocomputeClusterAttachment:   resourceAzureExocomputeClusterAttachment(),
			keyPolarisAzurePrivateContainerRegistry:      resourceAzurePrivateContainerRegistry(),
			keyPolarisAzureServicePrincipal:              resourceAzureServicePrincipal(),
			keyPolarisAzureSubscription:                  resourceAzureSubscription(),
			keyPolarisCDMBootstrap:                       resourceCDMBootstrap(),
			keyPolarisCDMBootstrapCCESAWS:                resourceCDMBootstrapCCESAWS(),
			keyPolarisCDMBootstrapCCESAzure:              resourceCDMBootstrapCCESAzure(),
			keyPolarisCDMRegistration:                    resourceCDMRegistration(),
			keyPolarisCustomRole:                         resourceCustomRole(),
			keyPolarisDataCenterAWSAccount:               resourceDataCenterAWSAccount(),
			keyPolarisDataCenterAzureSubscription:        resourceDataCenterAzureSubscription(),
			keyPolarisDataCenterArchivalLocationAmazonS3: resourceDataCenterArchivalLocationAmazonS3(),
			keyPolarisGCPArchivalLocation:                resourceGcpArchivalLocation(),
			keyPolarisGCPCustomLabels:                    resourceGcpCustomLabels(),
			keyPolarisGCPProject:                         resourceGcpProject(),
			keyPolarisGCPServiceAccount:                  resourceGcpServiceAccount(),
			keyPolarisRoleAssignment:                     resourceRoleAssignment(),
			keyPolarisSLADomain:                          resourceSLADomain(),
			keyPolarisSLADomainAssignment:                resourceSLADomainAssignment(),
			keyPolarisTagRule:                            resourceTagRule(),
			keyPolarisUser:                               resourceUser(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			keyPolarisAccount:                     dataSourceAccount(),
			keyPolarisAWSAccount:                  dataSourceAwsAccount(),
			keyPolarisAWSArchivalLocation:         dataSourceAwsArchivalLocation(),
			keyPolarisAWSCNPArtifacts:             dataSourceAwsArtifacts(),
			keyPolarisAWSCNPPermissions:           dataSourceAwsPermissions(),
			keyPolarisAzureArchivalLocation:       dataSourceAzureArchivalLocation(),
			keyPolarisAzurePermissions:            dataSourceAzurePermissions(),
			keyPolarisAzureSubscription:           dataSourceAzureSubscription(),
			keyPolarisDataCenterArchivalLocation:  dataSourceDataCenterArchivalLocation(),
			keyPolarisDataCenterAWSAccount:        dataSourceDataCenterAWSAccount(),
			keyPolarisDataCenterAzureSubscription: dataSourceDataCenterAzureSubscription(),
			keyPolarisDeployment:                  dataSourceDeployment(),
			keyPolarisFeatures:                    dataSourceFeatures(),
			keyPolarisGCPArchivalLocation:         dataSourceGcpArchivalLocation(),
			keyPolarisGCPPermissions:              dataSourceGcpPermissions(),
			keyPolarisGCPProject:                  dataSourceGcpProject(),
			keyPolarisRole:                        dataSourceRole(),
			keyPolarisRoleTemplate:                dataSourceRoleTemplate(),
			keyPolarisSLADomain:                   dataSourceSLADomain(),
			keyPolarisSLASourceCluster:            dataSourceSLASourceCluster(),
			keyPolarisSSOGroup:                    dataSourceSSOGroup(),
			keyPolarisTagRule:                     dataSourceTagRule(),
			keyPolarisUser:                        dataSourceUser(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

// providerConfigure configures the RSC provider.
func providerConfigure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	cacheParams := polaris.CacheParams{
		Enable: d.Get(keyTokenCache).(bool),
		Dir:    d.Get(keyTokenCacheDir).(string),
		Secret: d.Get(keyTokenCacheSecret).(string),
	}

	client, err := newClient(ctx, d.Get("credentials").(string), cacheParams)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return client, nil
}

type client struct {
	logger        log.Logger
	polarisClient *polaris.Client
	polarisErr    error
}

func newClient(ctx context.Context, credentials string, cacheParams polaris.CacheParams) (*client, error) {
	logger := newAPILogger(ctx)
	account, err := polaris.FindAccount(credentials, true)
	if err != nil && !errors.Is(err, polaris.ErrAccountNotFound) {
		return nil, err
	}
	var polarisClient *polaris.Client
	var accountErr error
	if err == nil {
		polarisClient, err = polaris.NewClientWithLoggerAndCacheParams(account, cacheParams, logger)
		if err != nil {
			return nil, err
		}
	} else {
		accountErr = err
	}

	return &client{
		logger:        logger,
		polarisClient: polarisClient,
		polarisErr:    accountErr,
	}, nil
}

func (c *client) flag(ctx context.Context, name core.FeatureFlagName) bool {
	ff, err := core.Wrap(c.polarisClient.GQL).FeatureFlag(ctx, name)
	return err != nil && ff.Enabled
}

func (c *client) polaris() (*polaris.Client, error) {
	if c.polarisClient == nil {
		err := errors.New("RSC functionality has not been configured")
		if c.polarisErr != nil {
			err = fmt.Errorf("%s: %s", err, c.polarisErr)
		}
		return nil, err
	}

	return c.polarisClient, nil
}

// description returns the description string with all acute accents replaced
// with grave accents (backticks).
func description(description string) string {
	return strings.ReplaceAll(description, "´", "`")
}
