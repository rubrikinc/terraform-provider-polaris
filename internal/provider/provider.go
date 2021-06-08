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
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/log"
)

// Provider defines the schema and resource map for the Polaris provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The local user account name or service account file to use when accessing Rubrik Polaris.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"polaris_aws_account":             resourceAwsAccount(),
			"polaris_azure_service_principal": resourceAzureServicePrincipal(),
			"polaris_azure_subscription":      resourceAzureSubcription(),
			"polaris_gcp_project":             resourceGcpProject(),
			"polaris_gcp_service_account":     resourceGcpServiceAccount(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

// providerConfigure configures the Polaris provider.
func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	credentials := d.Get("credentials").(string)

	// When credentials doesn't refer to an existing file we assume that
	// it's an account name.
	if _, err := os.Stat(credentials); err != nil {
		account, err := polaris.DefaultAccount(credentials)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		client, err := polaris.NewClient(account, &log.StandardLogger{})
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return client, nil
	}

	// Otherwise we load the file as a service account.
	account, err := polaris.ServiceAccountFromFile(credentials)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	client, err := polaris.NewClientFromServiceAccount(account, &log.StandardLogger{})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return client, nil
}
