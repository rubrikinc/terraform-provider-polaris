// Copyright 2024 Rubrik, Inc.
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAzureSubscriptionDescription = `
The ´polaris_azure_subscription´ data source is used to access information about an
Azure subscription added to RSC. An Azure subscription is looked up using either the
Azure subscription ID or the name. When looking up an Azure subscription using the
subscription name, the tenant domain can be used to specify in which tenant to look
for the name.

-> **Note:** The subscription name is the name of the Azure subscription as it appears
   in RSC.
`

func dataSourceAzureSubscription() *schema.Resource {
	return &schema.Resource{
		ReadContext: azureSubscriptionRead,

		Description: description(dataSourceAzureSubscriptionDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keySubscriptionID, keyName},
				Description:  "Azure subscription ID.",
				ValidateFunc: validation.IsUUID,
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keySubscriptionID, keyName},
				Description:  "Azure subscription name.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			keyTenantDomain: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Azure tenant primary domain.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func azureSubscriptionRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureSubscriptionRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the Azure subscription using either the ID or the name. We don't
	// allow prefix searches since it would be impossible to uniquely identify
	// a subscription with a name being the prefix of another subscription.
	var subscription azure.CloudAccount
	if subscriptionID := d.Get(keySubscriptionID).(string); subscriptionID != "" {
		id, err := uuid.Parse(subscriptionID)
		if err != nil {
			return diag.FromErr(err)
		}
		subscription, err = azure.Wrap(client).SubscriptionByNativeID(ctx, core.FeatureAll, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		subscription, err = azure.Wrap(client).SubscriptionByName(ctx, core.FeatureAll, d.Get(keyName).(string),
			d.Get(keyTenantDomain).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keySubscriptionID, subscription.NativeID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, subscription.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTenantDomain, subscription.TenantDomain); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subscription.ID.String())
	return nil
}
