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
	"io/fs"
	"net/mail"
	"os"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/cdm"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/log"
)

// Provider defines the schema and resource map for the RSC provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The service account file or local user account name to use when accessing RSC.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"polaris_aws_account":                    resourceAwsAccount(),
			"polaris_aws_archival_location":          resourceAwsArchivalLocation(),
			"polaris_aws_cnp_account":                resourceAwsCnpAccount(),
			"polaris_aws_cnp_account_attachments":    resourceAwsCnpAccountAttachments(),
			"polaris_aws_cnp_account_trust_policy":   resourceAwsCnpAccountTrustPolicy(),
			"polaris_aws_exocompute":                 resourceAwsExocompute(),
			"polaris_aws_private_container_registry": resourceAwsPrivateContainerRegistry(),
			"polaris_azure_exocompute":               resourceAzureExocompute(),
			"polaris_azure_service_principal":        resourceAzureServicePrincipal(),
			"polaris_azure_subscription":             resourceAzureSubscription(),
			"polaris_cdm_bootstrap":                  resourceCDMBootstrap(),
			"polaris_cdm_bootstrap_cces_aws":         resourceCDMBootstrapCCESAWS(),
			"polaris_cdm_bootstrap_cces_azure":       resourceCDMBootstrapCCESAzure(),
			"polaris_custom_role":                    resourceCustomRole(),
			"polaris_gcp_project":                    resourceGcpProject(),
			"polaris_gcp_service_account":            resourceGcpServiceAccount(),
			"polaris_role_assignment":                resourceRoleAssignment(),
			"polaris_user":                           resourceUser(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"polaris_aws_archival_location": dataSourceAwsArchivalLocation(),
			"polaris_aws_cnp_artifacts":     dataSourceAwsArtifacts(),
			"polaris_aws_cnp_permissions":   dataSourceAwsPermissions(),
			"polaris_azure_permissions":     dataSourceAzurePermissions(),
			"polaris_features":              dataSourceFeatures(),
			"polaris_gcp_permissions":       dataSourceGcpPermissions(),
			"polaris_role":                  dataSourceRole(),
			"polaris_role_template":         dataSourceRoleTemplate(),
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
	if c, ok := d.GetOk("credentials"); ok {
		credentials := c.(string)

		// When credentials refer to an existing file we load the file as a
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

			// Make sure interface value is an untyped nil, see SA4023 for
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

// validateDuration verifies that i contains a valid duration.
func validateDuration(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if _, err := time.ParseDuration(v); err != nil {
		return nil, []error{fmt.Errorf("%q is not a valid duration", v)}
	}

	return nil, nil
}

// validateEmailAddress verifies that i contains a valid email address.
func validateEmailAddress(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if _, err := mail.ParseAddress(v); err != nil {
		return nil, []error{fmt.Errorf("%q is not a valid email address", v)}
	}

	return nil, nil
}

// fileExists assumes m is a file path and returns nil if the file exists,
// otherwise a diagnostic message is returned.
func fileExists(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := os.Stat(m.(string)); err != nil {
		details := "unknown error"

		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			details = pathErr.Err.Error()
		}

		return diag.Errorf("failed to access file: %s", details)
	}

	return nil
}

// validateHash verifies that m contains a valid SHA-256 hash.
func validateHash(m interface{}, p cty.Path) diag.Diagnostics {
	if hash, ok := m.(string); ok && len(hash) == 64 {
		return nil
	}

	return diag.Errorf("invalid hash value")
}
