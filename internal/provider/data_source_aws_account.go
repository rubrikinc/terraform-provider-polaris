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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAwsAccountDescription = `
The ´polaris_aws_account´ data source is used to access information about an AWS account
added to RSC. An AWS account is looked up using either the AWS account ID or the name.

-> **Note:** The account name is the name of the AWS account as it appears in RSC.
`

func dataSourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsAccountRead,

		Description: description(dataSourceAwsAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyCloudAccountID, keyName},
				Description:  "AWS account ID.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyAccountID, keyName},
				Description:  "RSC cloud account ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyFeature: {
				Type:        schema.TypeSet,
				Elem:        featureResource(),
				Computed:    true,
				Description: "RSC feature with permission groups.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyAccountID, keyCloudAccountID},
				Description:  "AWS account name.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func awsAccountRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] awsAccountRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the AWS account using either the ID or the name. We don't allow
	// prefix searches since it would be impossible to uniquely identify an
	// account with a name being the prefix of another account.
	var account aws.CloudAccount
	switch {
	case d.Get(keyAccountID).(string) != "":
		account, err = aws.Wrap(client).AccountByNativeID(ctx, core.FeatureAll, d.Get(keyAccountID).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	case d.Get(keyName).(string) != "":
		account, err = aws.Wrap(client).AccountByName(ctx, core.FeatureAll, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	default:
		id, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		account, err = aws.Wrap(client).AccountByID(ctx, core.FeatureAll, id)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyAccountID, account.NativeID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudAccountID, account.ID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, account.Name); err != nil {
		return diag.FromErr(err)
	}

	features := &schema.Set{F: schema.HashResource(featureResource())}
	for _, feature := range account.Features {
		groups := &schema.Set{F: schema.HashString}
		for _, group := range feature.Feature.PermissionGroups {
			groups.Add(string(group))
		}
		features.Add(map[string]any{
			keyName:             feature.Feature.Name,
			keyPermissionGroups: groups,
		})
	}
	if err := d.Set(keyFeature, features); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(account.ID.String())
	return nil
}
