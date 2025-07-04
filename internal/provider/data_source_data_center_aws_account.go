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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
)

const dataSourceDataCenterAWSAccountDescription = `
The ´polaris_data_center_aws_account´ data source is used to access information
about an AWS data center account added to RSC. A data center account is looked
up using the name.

-> **Note:** Data center accounts and cloud native accounts are different and
   cannot be used interchangeably.

-> **Note:** The name is the name of the data center account as it appears in
   RSC.
`

func dataSourceDataCenterAWSAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataCenterAWSAccountRead,

		Description: description(dataSourceDataCenterAWSAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC data center cloud account ID (UUID).",
			},
			keyAccessKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS access key.",
			},
			keyConnectionStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Connection status.",
			},
			keyDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Data center account description.",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Data center account name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func dataCenterAWSAccountRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterAWSAccountRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the AWS cloud account using the name. We don't allow prefix searches
	// since it would be impossible to uniquely identify a cloud account with a
	// name being the prefix of another cloud account.
	cloudAccount, err := archival.Wrap(client).AWSCloudAccountByName(ctx, d.Get(keyName).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyAccessKey, cloudAccount.AccessKey); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionStatus, cloudAccount.ConnectionStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, cloudAccount.Description); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cloudAccount.ID.String())
	return nil
}
