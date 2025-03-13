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
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const resourceTagRuleDescription = `
The ´polaris_tag_rule´ resource manages RSC tag rules.

A tag is a key-value pair used to group cloud resources for a specific purpose.
This rule-based approach allows resource protection across multiple projects and
regions. A tag can be used to assign an SLA Domain to all resources belonging to
a specific application or department. When cloud resources are tagged
appropriately, they derive protection automatically when they are instantiated.
`

func resourceTagRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: createTagRule,
		ReadContext:   readTagRule,
		UpdateContext: updateTagRule,
		DeleteContext: deleteTagRule,

		Description: description(resourceTagRuleDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Tag rule ID (UUID).",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Tag rule name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyObjectType: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Object type to which the tag rule will be applied. Possible values are " +
					"`AWS_EBS_VOLUME`, `AWS_EC2_INSTANCE`, `AWS_RDS_INSTANCE`, `AWS_S3_BUCKET`, " +
					"`AZURE_MANAGED_DISK`, `AZURE_SQL_DATABASE_DB`, `AZURE_SQL_DATABASE_SERVER`, " +
					"`AZURE_SQL_MANAGED_INSTANCE_SERVER`, `AZURE_STORAGE_ACCOUNT` and `AZURE_VIRTUAL_MACHINE`. " +
					"Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice([]string{
					string(gqlsla.TagObjectAWSEBSVolume),
					string(gqlsla.TagObjectAWSEC2Instance),
					string(gqlsla.TagObjectAWSRDSInstance),
					string(gqlsla.TagObjectAWSS3Bucket),
					string(gqlsla.TagObjectAzureManagedDisk),
					string(gqlsla.TagObjectAzureSQLDatabaseDB),
					string(gqlsla.TagObjectAzureSQLDatabaseServer),
					string(gqlsla.TagObjectAzureSQLManagedInstanceServer),
					string(gqlsla.TagObjectAzureStorageAccount),
					string(gqlsla.TagObjectAzureVirtualMachine),
				}, false),
			},
			keyTagKey: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Tag key to match. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTagValue: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyTagAllValues},
				Description: "Tag value to match. If the tag value is empty, it matches empty values. To match all " +
					"tag values, use the `" + keyTagAllValues + "` field. Changing this forces a new resource to be " +
					"created.",
			},
			keyTagAllValues: {
				Type:         schema.TypeBool,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyTagValue},
				Description:  "If true, all tag values are matched. Changing this forces a new resource to be created.",
			},
			keyCloudAccountIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				Optional: true,
				Description: "The RSC cloud account IDs (UUID) to which the tag rule should be applied. If empty, " +
					"the tag rule will be applied to all RSC cloud accounts.",
			},
		},
	}
}

func createTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] createTagRule")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var cloudAccountIDs []uuid.UUID
	for _, cloudAccountID := range d.Get(keyCloudAccountIDs).(*schema.Set).List() {
		id, err := uuid.Parse(cloudAccountID.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		cloudAccountIDs = append(cloudAccountIDs, id)
	}
	cloudAccounts, err := groupCloudAccounts(ctx, client, cloudAccountIDs)
	if err != nil {
		return diag.FromErr(err)
	}

	tagRuleID, err := sla.Wrap(client).CreateTagRule(ctx, gqlsla.CreateTagRuleParams{
		Name:       d.Get(keyName).(string),
		ObjectType: gqlsla.CloudNativeTagObjectType(d.Get(keyObjectType).(string)),
		Tag: gqlsla.Tag{
			Key:       d.Get(keyTagKey).(string),
			Value:     d.Get(keyTagValue).(string),
			AllValues: d.Get(keyTagAllValues).(bool),
		},
		CloudAccounts:    cloudAccounts,
		AllCloudAccounts: cloudAccounts == nil,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tagRuleID.String())
	readTagRule(ctx, d, m)
	return nil
}

func readTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readTagRule")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tagRule, err := sla.Wrap(client).TagRuleByID(ctx, id)
	if err != nil {
		return diag.FromErr(err)
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

	return nil
}

func updateTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] updateTagRule")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var cloudAccountIDs []uuid.UUID
	for _, cloudAccountID := range d.Get(keyCloudAccountIDs).(*schema.Set).List() {
		id, err := uuid.Parse(cloudAccountID.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		cloudAccountIDs = append(cloudAccountIDs, id)
	}
	cloudAccounts, err := groupCloudAccounts(ctx, client, cloudAccountIDs)
	if err != nil {
		return diag.FromErr(err)
	}

	err = sla.Wrap(client).UpdateTagRule(ctx, id, gqlsla.UpdateTagRuleParams{
		Name:             d.Get(keyName).(string),
		CloudAccounts:    cloudAccounts,
		AllCloudAccounts: cloudAccounts == nil,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	readTagRule(ctx, d, m)
	return nil
}

func deleteTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteTagRule")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := sla.Wrap(client).DeleteTagRule(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func groupCloudAccounts(ctx context.Context, client *polaris.Client, cloudAccountIDs []uuid.UUID) (*gqlsla.TagRuleCloudAccounts, error) {
	if len(cloudAccountIDs) == 0 {
		return nil, nil
	}

	cloudAccounts := &gqlsla.TagRuleCloudAccounts{}
	for _, cloudAccountID := range cloudAccountIDs {
		cloudVendor, err := lookupCloudAccountID(ctx, client, cloudAccountID)
		if err != nil {
			return nil, err
		}

		switch cloudVendor {
		case core.CloudVendorAWS:
			cloudAccounts.AWSAccountIDs = append(cloudAccounts.AWSAccountIDs, cloudAccountID)
		case core.CloudVendorAzure:
			cloudAccounts.AzureSubscriptionIDs = append(cloudAccounts.AzureSubscriptionIDs, cloudAccountID)
		case core.CloudVendorGCP:
			cloudAccounts.GCPProjectIDs = append(cloudAccounts.GCPProjectIDs, cloudAccountID)
		default:
			return nil, fmt.Errorf("unknown cloud vendor: %s", cloudVendor)
		}
	}

	return cloudAccounts, nil
}

func lookupCloudAccountID(ctx context.Context, client *polaris.Client, cloudAccountID uuid.UUID) (core.CloudVendor, error) {
	_, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(cloudAccountID), core.FeatureAll)
	if err != nil && !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnspecified, err
	}
	if err == nil {
		return core.CloudVendorAWS, nil
	}

	_, err = azure.Wrap(client).Subscription(ctx, azure.CloudAccountID(cloudAccountID), core.FeatureAll)
	if err != nil && !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnspecified, err
	}
	if err == nil {
		return core.CloudVendorAzure, nil
	}

	_, err = gcp.Wrap(client).Project(ctx, gcp.CloudAccountID(cloudAccountID), core.FeatureAll)
	if err != nil && !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnspecified, err
	}
	if err == nil {
		return core.CloudVendorGCP, nil
	}

	return core.CloudVendorUnspecified, fmt.Errorf("cloud account %q %w", cloudAccountID, graphql.ErrNotFound)
}
