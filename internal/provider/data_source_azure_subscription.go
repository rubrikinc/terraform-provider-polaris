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
)

const dataSourceAzureSubscriptionDescription = `
The ´polaris_azure_subscription´ data source is used to access information
about an Azure subscription added to RSC. An Azure subscription is looked up
using either the Azure subscription ID, the RSC cloud account ID, or the name.
When looking up an Azure subscription using the subscription name, the tenant
domain can be used to specify in which tenant to look for the name.

-> **Note:** The subscription name is the name of the Azure subscription as it appears
   in RSC.
`

func dataSourceAzureSubscription() *schema.Resource {
	return &schema.Resource{
		ReadContext: azureSubscriptionRead,

		Description: description(dataSourceAzureSubscriptionDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyName, keySubscriptionID},
				Description:  "RSC cloud account ID (UUID).",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keySubscriptionID},
				Description:  "Azure subscription name.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyName},
				Description:  "Azure subscription ID.",
				ValidateFunc: validation.IsUUID,
			},
			keyTenantDomain: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
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

	// We don't allow prefix searches since it would be impossible to uniquely
	// identify a subscription with a name being the prefix of another
	// subscription.
	var subscription azure.CloudAccount
	switch {
	case d.Get(keySubscriptionID).(string) != "":
		id, err := uuid.Parse(d.Get(keySubscriptionID).(string))
		if err != nil {
			return diag.Errorf("failed to parse subscription id: %s", err)
		}
		subscription, err = azure.Wrap(client).SubscriptionByNativeID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	case d.Get(keyName).(string) != "":
		subscription, err = azure.Wrap(client).SubscriptionByName(ctx, d.Get(keyName).(string),
			d.Get(keyTenantDomain).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	default:
		id, err := uuid.Parse(d.Get(keyID).(string))
		if err != nil {
			return diag.Errorf("failed to parse id: %s", err)
		}
		subscription, err = azure.Wrap(client).SubscriptionByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, subscription.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySubscriptionID, subscription.NativeID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTenantDomain, subscription.TenantDomain); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subscription.ID.String())
	return nil
}
