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
	"io/fs"
	"os"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			"polaris_aws_account":                  resourceAwsAccount(),
			"polaris_aws_cnp_account":              resourceAwsCnpAccount(),
			"polaris_aws_cnp_account_attachments":  resourceAwsCnpAccountAttachments(),
			"polaris_aws_cnp_account_trust_policy": resourceAwsCnpAccountTrustPolicy(),
			"polaris_aws_exocompute":               resourceAwsExocompute(),
			"polaris_azure_exocompute":             resourceAzureExocompute(),
			"polaris_azure_service_principal":      resourceAzureServicePrincipal(),
			"polaris_azure_subscription":           resourceAzureSubscription(),
			"polaris_custom_role":                  resourceCustomRole(),
			"polaris_gcp_project":                  resourceGcpProject(),
			"polaris_gcp_service_account":          resourceGcpServiceAccount(),
			"polaris_role_assignment":              resourceRoleAssignment(),
			"polaris_user":                         resourceUser(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"polaris_aws_cnp_artifacts":   dataSourceAwsArtifacts(),
			"polaris_aws_cnp_permissions": dataSourceAwsPermissions(),
			"polaris_azure_permissions":   dataSourceAzurePermissions(),
			"polaris_features":            dataSourceFeatures(),
			"polaris_gcp_permissions":     dataSourceGcpPermissions(),
			"polaris_role":                dataSourceRole(),
			"polaris_role_template":       dataSourceRoleTemplate(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

// providerConfigure configures the RSC provider.
func providerConfigure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	credentials := d.Get("credentials").(string)

	// If no credentials are given or the credentials refer to an existing file,
	// we load the credentials as a service account, otherwise we assume that
	// it's an account name.
	var account polaris.Account
	if credentials != "" {
		if _, err := os.Stat(credentials); err == nil {
			account, err = polaris.ServiceAccountFromFile(credentials, true)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		} else {
			account, err = polaris.DefaultUserAccount(credentials, true)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		}
	} else {
		var err error
		account, err = polaris.ServiceAccountFromEnv()
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	logger := log.NewStandardLogger()
	polaris.SetLogLevelFromEnv(logger)
	client, err := polaris.NewClientWithLogger(account, logger)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return client, nil
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
