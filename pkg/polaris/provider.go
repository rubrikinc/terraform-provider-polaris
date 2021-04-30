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

package polaris

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris/log"
)

// fromResourceID converts a resource id to a Polaris cloud account id and
// a native id. Native id in this context means CSP specific id.
func fromResourceID(resourceID string) (cloudAccountID string, nativeID string, err error) {
	parts := strings.Split(resourceID, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("polaris: invalid resource id: %s", resourceID)
	}

	return parts[0], parts[1], nil
}

// toResourceID converts a Polaris cloud account id and a native id to a
// resource id. Native id in this context means CSP specific id.
func toResourceID(cloudAccountID, nativeID string) string {
	return fmt.Sprintf("%s:%s", cloudAccountID, nativeID)
}

// Provider defines the schema and resource map for the Polaris provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account": {
				Type:             schema.TypeString,
				Required:         true,
				DefaultFunc:      schema.EnvDefaultFunc("RUBRIK_POLARIS_ACCOUNT", "default"),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "The account to use when accessing Rubrik Polaris.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"polaris_aws_account": resourceAwsAccount(),
			"polaris_gcp_project": resourceGcpProject(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

// providerConfigure configures the Polaris provider.
func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	account := d.Get("account").(string)

	// Load default configuration from the users home folder.
	polConfig, err := polaris.DefaultConfig(account)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// Create the Polaris client.
	polClient, err := polaris.NewClient(polConfig, &log.StandardLogger{})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return polClient, nil
}
