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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// resourceAzureExocomputeV0 defines the schema for version 0 of the Azure
// service principal resource and how to migrate to version 1.
func resourceAzureExocomputeV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Polaris subscription id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"polaris_managed": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "If true the security groups are managed by Polaris.",
			},
			"region": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Azure region to run the exocompute instance in.",
				ValidateDiagFunc: validateAzureRegion,
			},
			"subnet": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Azure subnet id.",
			},
		},
	}
}

// resourceAzureExocomputeStateUpgradeV0 removes the polaris_managed parameter.
// Exocompute on Azure only supports RSC managed configurations.
func resourceAzureExocomputeStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	tflog.Trace(ctx, "azureExocomputeStateUpgradeV0")

	delete(state, "polaris_managed")

	return state, nil
}
