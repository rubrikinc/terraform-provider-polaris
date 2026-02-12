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
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

-> **Note:** Tag key and tag value are case sensitive.
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
					"`AWS_DYNAMODB_TABLE`, `AZURE_MANAGED_DISK`, `AZURE_SQL_DATABASE_DB`, `AZURE_SQL_DATABASE_SERVER`, " +
					"`AZURE_SQL_MANAGED_INSTANCE_SERVER`, `AZURE_STORAGE_ACCOUNT` and `AZURE_VIRTUAL_MACHINE`. " +
					"Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(gqlsla.AllCloudNativeTagObjectTypesAsStrings(), false),
			},
			keyTagKey: {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyTag},
				Description: "Tag key to match. Changing this forces a new resource to be created. " +
					"**Deprecated:** Use `tag` instead.",
				Deprecated:   "Use tag instead.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTagValue: {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyTag, keyTagAllValues},
				Description: "Tag value to match. If the tag value is empty, it matches empty values. " +
					"Changing this forces a new resource to be created. **Deprecated:** Use `tag` instead.",
				Deprecated: "Use tag instead.",
			},
			keyTagAllValues: {
				Type:          schema.TypeBool,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyTag, keyTagValue},
				Description: "If true, all tag values are matched. Changing this forces a new resource to be created. " +
					"**Deprecated:** Use `tag` instead.",
				Deprecated: "Use tag instead.",
			},
			keyTag: {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyTagKey, keyTagValue, keyTagAllValues},
				Description:   "Tag conditions to match. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyKey: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Tag key to match.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyValues: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Tag values to match. If empty and `match_all` is false, matches empty values.",
						},
						keyTagMatchAll: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, all tag values for this key are matched. Default is false.",
						},
					},
				},
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
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, m any) error {
			tflog.Trace(ctx, "customizeDiffTagRule")

			// Validate that at least one of tag or tag_key is specified.
			tags := diff.Get(keyTag).([]any)
			if len(tags) == 0 {
				if _, ok := diff.GetOk(keyTagKey); !ok {
					return errors.New("one of `tag` or `tag_key` must be specified")
				}

				// When using the deprecated tag_key field, require either tag_value
				// or tag_all_values to preserve the original ExactlyOneOf validation.
				_, tagValueSet := diff.GetOk(keyTagValue)
				_, tagAllValuesSet := diff.GetOk(keyTagAllValues)
				if !tagValueSet && !tagAllValuesSet {
					return fmt.Errorf("one of `%s` or `%s` must be specified when using `%s`",
						keyTagValue, keyTagAllValues, keyTagKey)
				}
			}

			// Validate that values is empty when match_all is true.
			for _, t := range tags {
				tMap := t.(map[string]any)
				if tMap[keyTagMatchAll].(bool) {
					if values := tMap[keyValues].([]any); len(values) > 0 {
						return fmt.Errorf("`%s` must be empty when `%s` is true", keyValues, keyTagMatchAll)
					}
				}
			}

			return nil
		},
	}
}

func createTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "createTagRule")

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

	params := gqlsla.CreateTagRuleParams{
		Name:             d.Get(keyName).(string),
		ObjectType:       gqlsla.CloudNativeTagObjectType(d.Get(keyObjectType).(string)),
		CloudAccounts:    cloudAccounts,
		AllCloudAccounts: cloudAccounts == nil,
	}

	// Check if new style tag block is used.
	if tags, ok := d.GetOk(keyTag); ok {
		tagList := tags.([]any)
		tagPairs := make([]gqlsla.TagPair, 0, len(tagList))
		for _, t := range tagList {
			tMap := t.(map[string]any)
			values := make([]string, 0)
			for _, v := range tMap[keyValues].([]any) {
				values = append(values, v.(string))
			}
			tagPairs = append(tagPairs, gqlsla.TagPair{
				Key:               tMap[keyKey].(string),
				MatchAllTagValues: tMap[keyTagMatchAll].(bool),
				Values:            values,
			})
		}
		params.TagConditions = &gqlsla.TagConditions{TagPairs: tagPairs}
	} else {
		// Use deprecated tag fields for backward compatibility.
		//lint:ignore SA1019 using deprecated field for backward compatibility
		params.Tag = gqlsla.Tag{
			Key:       d.Get(keyTagKey).(string),
			Value:     d.Get(keyTagValue).(string),
			AllValues: d.Get(keyTagAllValues).(bool),
		}
	}

	tagRuleID, err := sla.Wrap(client).CreateTagRule(ctx, params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tagRuleID.String())
	// Read back the created resource to populate computed fields. A failed
	// readback must not be returned as an error: the resource was successfully
	// created and returning an error here would leave Terraform unable to
	// manage it. A plan diff on the next run is an acceptable outcome.
	if diags := readTagRule(ctx, d, m); diags.HasError() {
		for _, diagnostic := range diags {
			tflog.Warn(ctx, "failed to read back tag rule after create", map[string]any{
				"summary": diagnostic.Summary,
				"detail":  diagnostic.Detail,
			})
		}
	}
	return nil
}

func readTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "readTagRule")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tagRule, err := sla.Wrap(client).TagRuleByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// useLegacyFields is true when prior state exists (name is set, ruling
	// out an import) and the state uses the deprecated tag_key/tag_value/
	// tag_all_values fields. In that case we populate those fields and skip
	// the new tag block to avoid spurious plan diffs for configurations that
	// have not yet migrated. On import we always use the new tag block since
	// there is no prior state to preserve.
	_, nameSet := d.GetOk(keyName)
	_, tagKeySet := d.GetOk(keyTagKey)
	if useLegacyFields := nameSet && tagKeySet; useLegacyFields {
		//lint:ignore SA1019 using deprecated field for backward compatibility
		if tagRule.Tag.Key != "" {
			//lint:ignore SA1019 using deprecated field for backward compatibility
			if err := d.Set(keyTagKey, tagRule.Tag.Key); err != nil {
				return diag.FromErr(err)
			}
			//lint:ignore SA1019 using deprecated field for backward compatibility
			if err := d.Set(keyTagValue, tagRule.Tag.Value); err != nil {
				return diag.FromErr(err)
			}
			//lint:ignore SA1019 using deprecated field for backward compatibility
			if err := d.Set(keyTagAllValues, tagRule.Tag.AllValues); err != nil {
				return diag.FromErr(err)
			}
		}
	} else {
		// New tag block style (also used for imports). Sort by key and by values
		// within each tag for deterministic state output regardless of API return
		// order.
		tags := make([]map[string]any, 0, len(tagRule.TagConditions.TagPairs))
		for _, pair := range tagRule.TagConditions.TagPairs {
			slices.Sort(pair.Values)
			tags = append(tags, map[string]any{
				keyKey:         pair.Key,
				keyValues:      pair.Values,
				keyTagMatchAll: pair.MatchAllTagValues,
			})
		}
		slices.SortFunc(tags, func(a, b map[string]any) int {
			return cmp.Compare(a[keyKey].(string), b[keyKey].(string))
		})
		if err := d.Set(keyTag, tags); err != nil {
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

	if !tagRule.AllACloudAccounts {
		cloudAccountIDs := &schema.Set{F: schema.HashString}
		for _, nativeCloudAccount := range tagRule.CloudAccounts {
			cloudAccountID, err := lookupCloudAccountID(ctx, client, nativeCloudAccount.ID)
			if err != nil {
				return diag.FromErr(err)
			}
			cloudAccountIDs.Add(cloudAccountID.String())
		}
		if err := d.Set(keyCloudAccountIDs, cloudAccountIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func updateTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "updateTagRule")

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

	// Read back the updated resource to populate computed fields. A failed
	// readback must not be returned as an error: the resource was successfully
	// updated and returning an error here would leave Terraform unable to
	// manage it. A plan diff on the next run is an acceptable outcome.
	if diags := readTagRule(ctx, d, m); diags.HasError() {
		for _, diagnostic := range diags {
			tflog.Warn(ctx, "failed to read back tag rule after update", map[string]any{
				"summary": diagnostic.Summary,
				"detail":  diagnostic.Detail,
			})
		}
	}
	return nil
}

func deleteTagRule(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "deleteTagRule")

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
		cloudVendor, nativeCloudAccountID, err := lookupNativeCloudAccountID(ctx, client, cloudAccountID)
		if err != nil {
			return nil, err
		}

		switch cloudVendor {
		case core.CloudVendorAWS:
			cloudAccounts.AWSAccountIDs = append(cloudAccounts.AWSAccountIDs, nativeCloudAccountID)
		case core.CloudVendorAzure:
			cloudAccounts.AzureSubscriptionIDs = append(cloudAccounts.AzureSubscriptionIDs, nativeCloudAccountID)
		case core.CloudVendorGCP:
			cloudAccounts.GCPProjectIDs = append(cloudAccounts.GCPProjectIDs, nativeCloudAccountID)
		default:
			return nil, fmt.Errorf("unknown cloud vendor: %s", cloudVendor)
		}
	}

	return cloudAccounts, nil
}

// Looks up the cloud account ID for the specified native cloud account ID.
// Note, AWS uses the same ID for both cloud accounts and native cloud accounts.
func lookupCloudAccountID(ctx context.Context, client *polaris.Client, nativeCloudAccountID uuid.UUID) (uuid.UUID, error) {
	_, err := aws.Wrap(client).AccountByID(ctx, nativeCloudAccountID)
	if err == nil {
		return nativeCloudAccountID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return uuid.Nil, err
	}

	subscription, err := azure.Wrap(client).SubscriptionByNativeCloudAccountID(ctx, nativeCloudAccountID)
	if err == nil {
		return subscription.ID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return uuid.Nil, err
	}

	project, err := gcp.Wrap(client).ProjectByNativeCloudAccountID(ctx, nativeCloudAccountID)
	if err == nil {
		return project.ID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return uuid.Nil, err
	}

	return uuid.Nil, fmt.Errorf("cloud account id for native cloud account %q %w", nativeCloudAccountID, graphql.ErrNotFound)
}

// Looks up the native cloud account ID for the specified cloud account ID.
// Note, AWS uses the same ID for both cloud accounts and native cloud accounts.
func lookupNativeCloudAccountID(ctx context.Context, client *polaris.Client, cloudAccountID uuid.UUID) (core.CloudVendor, uuid.UUID, error) {
	_, err := aws.Wrap(client).AccountByID(ctx, cloudAccountID)
	if err == nil {
		return core.CloudVendorAWS, cloudAccountID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnknown, uuid.Nil, err
	}

	nativeSubscription, err := azure.Wrap(client).NativeSubscriptionByCloudAccountID(ctx, cloudAccountID)
	if err == nil {
		return core.CloudVendorAzure, nativeSubscription.ID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnknown, uuid.Nil, err
	}

	nativeProject, err := gcp.Wrap(client).NativeProjectByCloudAccountID(ctx, cloudAccountID)
	if err == nil {
		return core.CloudVendorGCP, nativeProject.ID, nil
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return core.CloudVendorUnknown, uuid.Nil, err
	}

	return core.CloudVendorUnknown, uuid.Nil, fmt.Errorf("native cloud account id for cloud account %q %w", cloudAccountID, graphql.ErrNotFound)
}
