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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
)

const resourceDataCenterAWSAccountDescription = `
The ´polaris_data_center_aws_account´ resource adds a data center AWS account to
RSC. A data center account can only be used with data center archival.

~> **Note:** Due to technical issue in RSC, names of removed data center AWS
   accounts cannot be reused.

-> **Note:** Data center accounts and cloud native accounts are different and
   cannot be used interchangeably.
`

func resourceDataCenterAWSAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: dataCenterAWSCreateAccount,
		ReadContext:   dataCenterAWSReadAccount,
		UpdateContext: dataCenterAWSUpdateAccount,
		DeleteContext: dataCenterAWSDeleteAccount,

		Description: description(resourceDataCenterAWSAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC data center cloud account ID (UUID).",
			},
			keyAccessKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS access key.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Data center account description.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Data center account name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keySecretKey: {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				Description:  "AWS secret key.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func dataCenterAWSCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] dataCenterAWSCreateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := archival.Wrap(client).CreateAWSCloudAccount(ctx, gqlarchival.CreateAWSCloudAccountParams{
		Name:        d.Get(keyName).(string),
		Description: d.Get(keyDescription).(string),
		AccessKey:   d.Get(keyAccessKey).(string),
		SecretKey:   d.Get(keySecretKey).(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	dataCenterAWSReadAccount(ctx, d, m)
	return nil
}

func dataCenterAWSReadAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] dataCenterAWSReadAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccount, err := archival.Wrap(client).AWSCloudAccountByID(ctx, id)
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
	if err := d.Set(keyAccessKey, cloudAccount.AccessKey); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterAWSUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] dataCenterAWSUpdateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = archival.Wrap(client).UpdateAWSCloudAccount(ctx, id, gqlarchival.UpdateAWSCloudAccountParams{
		Name:        d.Get(keyName).(string),
		Description: d.Get(keyDescription).(string),
		AccessKey:   d.Get(keyAccessKey).(string),
		SecretKey:   d.Get(keySecretKey).(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterAWSDeleteAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] dataCenterAWSDeleteAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := archival.Wrap(client).DeleteAWSCloudAccount(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
