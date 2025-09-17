// Copyright 2023 Rubrik, Inc.
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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAWSCNPAccount = `
The ´polaris_aws_cnp_account´ resource adds an AWS account to RSC using the
non-CFT (Cloud Formation Template) workflow. The ´polaris_aws_account´ resource
can be used to add an AWS account to RSC using the CFT workflow.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

´CLOUD_NATIVE_ARCHIVAL´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_S3_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´EXOCOMPUTE´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RSC_MANAGED_CLUSTER´ - Represents the set of permissions required for the
    Rubrik-managed Exocompute cluster.

´RDS_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´SERVERS_AND_APPS´
  * CLOUD_CLUSTER_ES - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the ´BASIC´ permission group
   is always required except for the ´SERVERS_AND_APPS´ feature.
`

func resourceAwsCnpAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccount,
		ReadContext:   awsReadCnpAccount,
		UpdateContext: awsUpdateCnpAccount,
		DeleteContext: awsDeleteCnpAccount,

		Description: description(resourceAWSCNPAccount),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyCloud: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "STANDARD",
				Description: "AWS cloud type. Possible values are `STANDARD`, `CHINA` and `GOV`. Default value is " +
					"`STANDARD`. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice([]string{"STANDARD", "CHINA", "GOV"}, false),
			},
			keyDeleteSnapshotsOnDestroy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			keyExternalID: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "External ID. Changing this forces a new resource to be created.",
			},
			keyFeature: {
				Type:        schema.TypeSet,
				Elem:        featureResource(),
				MinItems:    1,
				Required:    true,
				Description: "RSC feature with permission groups.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Account name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyNativeID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "AWS account ID. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegions: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:    1,
				Required:    true,
				Description: "Regions.",
			},
		},
	}
}

func awsCreateCnpAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	cloud := d.Get(keyCloud).(string)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block[keyName].(string)}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		features = append(features, feature)
	}

	name := d.Get(keyName).(string)
	nativeID := d.Get(keyNativeID).(string)
	var regions []string
	for _, region := range d.Get(keyRegions).(*schema.Set).List() {
		regions = append(regions, region.(string))
	}

	// Request account be added.
	id, err := aws.Wrap(client).AddAccount(ctx, aws.AccountWithName(cloud, nativeID, name), features, aws.Regions(regions...))
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	d.SetId(id.String())

	awsReadCnpAccount(ctx, d, m)
	return nil
}

func awsReadCnpAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Request cloud account.
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	if err := d.Set("cloud", account.Cloud); err != nil {
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
	if err := d.Set(keyName, account.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyNativeID, account.NativeID); err != nil {
		return diag.FromErr(err)
	}
	regions := &schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		for _, region := range feature.Regions {
			regions.Add(region)
		}
	}
	if err := d.Set(keyRegions, regions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCnpAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cloud := d.Get(keyCloud).(string)
	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block[keyName].(string)}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		features = append(features, feature)
	}
	name := d.Get(keyName).(string)
	nativeID := d.Get(keyNativeID).(string)
	var regions []string
	for _, region := range d.Get(keyRegions).(*schema.Set).List() {
		regions = append(regions, region.(string))
	}

	// Check that the cloud account exists.
	_, err = aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(keyName) {
		if err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureAll, aws.Name(name)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(keyFeature) {
		oldAttr, newAttr := d.GetChange(keyFeature)

		var oldFeatures []core.Feature
		for _, block := range oldAttr.(*schema.Set).List() {
			block := block.(map[string]interface{})
			feature := core.Feature{Name: block[keyName].(string)}
			for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
			oldFeatures = append(oldFeatures, feature)
		}

		var newFeatures []core.Feature
		for _, block := range newAttr.(*schema.Set).List() {
			block := block.(map[string]interface{})
			feature := core.Feature{Name: block[keyName].(string)}
			for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
			newFeatures = append(newFeatures, feature)
		}

		// When adding new features the list should include all features. When
		// removing features only the features to be removed should be passed
		// in.
		removeFeatures, updateFeatures := diffFeatures(oldFeatures, newFeatures)
		account := aws.AccountWithName(cloud, nativeID, name)
		if len(updateFeatures) > 0 {
			if _, err := aws.Wrap(client).AddAccount(ctx, account, updateFeatures, aws.Regions(regions...)); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(removeFeatures) > 0 {
			if err := aws.Wrap(client).RemoveAccount(ctx, account, removeFeatures, deleteSnapshots); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keyRegions) {
		var regions []string
		for _, region := range d.Get(keyRegions).(*schema.Set).List() {
			regions = append(regions, region.(string))
		}

		for _, feature := range features {
			if err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), feature, aws.Regions(regions...)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

func awsDeleteCnpAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)

	// Request the cloud account.
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	features := make([]core.Feature, 0, len(account.Features))
	for _, feature := range account.Features {
		features = append(features, feature.Feature)
	}

	// Request account removal.
	if err := aws.Wrap(client).RemoveAccount(ctx, aws.AccountWithName(account.Cloud, account.NativeID, account.Name), features, deleteSnapshots); err != nil {
		return diag.FromErr(err)
	}

	// Reset ID.
	d.SetId("")
	return nil
}

func featureResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyName: {
				Type:     schema.TypeString,
				Required: true,
				Description: "RSC feature name. Possible values are `CLOUD_NATIVE_ARCHIVAL`, " +
					"`CLOUD_NATIVE_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, `SERVERS_AND_APPS`, `EXOCOMPUTE` and `RDS_PROTECTION`.",
				ValidateFunc: validation.StringInSlice([]string{
					"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION", "EXOCOMPUTE",
					"RDS_PROTECTION", "SERVERS_AND_APPS",
				}, false),
			},
			keyPermissionGroups: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"BASIC", "RSC_MANAGED_CLUSTER", "CLOUD_CLUSTER_ES",
						// The following permission groups cannot be used when onboarding an AWS account.
						// They have been accepted in the past so we still silently allow them.
						"EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY", "SNAPSHOT_PRIVATE_ACCESS", "PRIVATE_ENDPOINT",
					}, false),
				},
				Required: true,
				Description: "RSC permission groups for the feature. Possible values are `BASIC` and " +
					"`RSC_MANAGED_CLUSTER`. For backwards compatibility, `[]` is interpreted as all applicable " +
					"permission groups.",
			},
		},
	}
}

func diffFeatures(oldFeatures []core.Feature, newFeatures []core.Feature) ([]core.Feature, []core.Feature) {
	oldSet := make(map[string]core.Feature)
	for _, feature := range oldFeatures {
		oldSet[feature.Name] = feature
	}
	newSet := make(map[string]core.Feature)
	for _, feature := range newFeatures {
		newSet[feature.Name] = feature
	}

	for name, oldFeature := range oldSet {
		if newFeature, ok := newSet[name]; ok {
			if oldFeature.DeepEqual(newFeature) {
				delete(newSet, name)
			}
			delete(oldSet, name)
		}
	}

	removeFeatures := make([]core.Feature, 0, len(oldSet))
	for _, feature := range oldSet {
		removeFeatures = append(removeFeatures, feature)
	}
	updateFeatures := make([]core.Feature, 0, len(newSet))
	for _, feature := range newSet {
		updateFeatures = append(updateFeatures, feature)
	}

	return removeFeatures, updateFeatures
}
