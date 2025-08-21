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
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
)

const resourceDataCenterAzureSubscriptionDescription = `
The ´polaris_data_center_azure_subscription´ resource adds a data center Azure
subscription to RSC. A data center subscription can only be used with data
center archival.

~> **Note:** Due to technical issue in RSC, names of removed data center Azure
   subscriptions cannot be reused.

-> **Note:** Data center subscriptions and cloud native subscriptions are
   different and cannot be used interchangeably.
`

func resourceDataCenterAzureSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: dataCenterAzureCreateSubscription,
		ReadContext:   dataCenterAzureReadSubscription,
		UpdateContext: dataCenterAzureUpdateSubscription,
		DeleteContext: dataCenterAzureDeleteSubscription,

		Description: description(resourceDataCenterAzureSubscriptionDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC data center cloud account ID (UUID).",
			},
			keyDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Data center subscription description.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Data center subscription name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keySubscriptionID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Azure subscription ID (UUID).",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTenantID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Azure tenant ID (UUID).",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func dataCenterAzureCreateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterAzureCreateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := archival.Wrap(client).CreateAzureCloudAccount(ctx, gqlarchival.CreateAzureCloudAccountParams{
		Name:           d.Get(keyName).(string),
		Description:    d.Get(keyDescription).(string),
		SubscriptionID: d.Get(keySubscriptionID).(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	dataCenterAzureReadSubscription(ctx, d, m)
	return nil
}

func dataCenterAzureReadSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterAzureReadSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccount, err := archival.Wrap(client).AzureCloudAccountByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, cloudAccount.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, cloudAccount.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySubscriptionID, cloudAccount.SubscriptionID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTenantID, cloudAccount.TenantID.String()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterAzureUpdateSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterAzureUpdateSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = archival.Wrap(client).UpdateAzureCloudAccount(ctx, id, gqlarchival.UpdateAzureCloudAccountParams{
		Name:           d.Get(keyName).(string),
		Description:    d.Get(keyDescription).(string),
		SubscriptionID: d.Get(keySubscriptionID).(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterAzureDeleteSubscription(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterAzureDeleteSubscription")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := archival.Wrap(client).DeleteAzureCloudAccount(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
