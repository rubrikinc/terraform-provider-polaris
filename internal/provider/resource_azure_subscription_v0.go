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

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

func validateAzureRegion(m interface{}, p cty.Path) diag.Diagnostics {
	return nil
}

// resourceAzureSubscriptionV0 defines the schema for version 0 of the Azure
// subscription resource and how to migrate to version 1.
func resourceAzureSubscriptionV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"delete_snapshots_on_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateAzureRegion,
				},
				Required: true,
			},
			"subscription_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"subscription_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_domain": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

// resourceAzureSubscriptionStateUpgradeV0 migrates the resource id from the
// Azure subscription id to the Polaris cloud account id.
func resourceAzureSubscriptionStateUpgradeV0(ctx context.Context, state map[string]interface{}, m interface{}) (map[string]interface{}, error) {
	tflog.Trace(ctx, "azureSubscriptionStateUpgradeV0")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(state["id"].(string))
	if err != nil {
		return nil, err
	}

	account, err := azure.Wrap(client).Subscription(ctx, azure.SubscriptionID(id), core.FeatureCloudNativeProtection)
	if err != nil {
		return nil, err
	}

	// Migrate the id to the Polaris cloud account id.
	state["id"] = account.ID.String()
	return state, nil
}
