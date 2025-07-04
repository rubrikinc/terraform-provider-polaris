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
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const dataSourceTagRuleDescription = `
The ´polaris_tag_rule´ data source is used to access information about RSC tag
rules. A tag rule is looked up using either the tag rule ID or the tag rule
name.
`

func dataSourceTagRule() *schema.Resource {
	return &schema.Resource{
		ReadContext: tagRuleRead,

		Description: description(dataSourceTagRuleDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyName},
				ValidateFunc: validation.IsUUID,
				Description:  "Tag rule ID (UUID).",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID},
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Tag rule name.",
			},
			keyObjectType: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Object type to which the tag rule will be applied. Possible values are " +
					"`AWS_EBS_VOLUME`, `AWS_EC2_INSTANCE`, `AWS_RDS_INSTANCE`, `AWS_S3_BUCKET`, " +
					"`AZURE_MANAGED_DISK`, `AZURE_SQL_DATABASE_DB`, `AZURE_SQL_DATABASE_SERVER`, " +
					"`AZURE_SQL_MANAGED_INSTANCE_SERVER`, `AZURE_STORAGE_ACCOUNT` and `AZURE_VIRTUAL_MACHINE`.",
			},
			keyTagKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Tag key to match.",
			},
			keyTagValue: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Tag value to match. If the tag value is empty, it matches empty values.",
			},
			keyTagAllValues: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "If true, all tag values are matched.",
			},
			keyCloudAccountIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "The RSC cloud account IDs (UUID) to which the tag rule should be applied. If empty, " +
					"the tag rule will be applied to all RSC cloud accounts.",
			},
		},
	}
}

func tagRuleRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "tagRuleRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var tagRule gqlsla.TagRule
	if id := d.Get(keyID).(string); id != "" {
		id, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		tagRule, err = sla.Wrap(client).TagRuleByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		tagRule, err = sla.Wrap(client).TagRuleByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, tagRule.Name); err != nil {
		return diag.FromErr(err)
	}

	tagObjectType, err := gqlsla.FromManagedObjectType(tagRule.ObjectType)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyObjectType, string(tagObjectType)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyTagKey, tagRule.Tag.Key); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTagValue, tagRule.Tag.Value); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyTagAllValues, tagRule.Tag.AllValues); err != nil {
		return diag.FromErr(err)
	}

	if !tagRule.AllACloudAccounts {
		cloudAccountIDs := &schema.Set{F: schema.HashString}
		for _, cloudAccount := range tagRule.CloudAccounts {
			cloudAccountIDs.Add(cloudAccount.ID.String())
		}
		if err := d.Set(keyCloudAccountIDs, cloudAccountIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(tagRule.ID.String())
	return nil
}
