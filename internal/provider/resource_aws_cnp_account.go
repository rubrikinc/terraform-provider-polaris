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
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAWSCNPAccount = `
The ´polaris_aws_cnp_account´ resource adds an AWS account to RSC. To grant RSC
permissions to perform certain operations on the account, IAM roles needs to be
created and communicated to RSC using the ´polaris_aws_cnp_attachment´ resource.
The roles and permissions needed by RSC can be looked up using the
´polaris_aws_cnp_artifact´ and ´polaris_aws_cnp_permissions´ data sources.

The ´CLOUD_DISCOVERY´ feature enables RSC to discover resources in the AWS
account without enabling protection. It is currently optional but will become
required when onboarding protection features. Once onboarded, it cannot be
removed unless all protection features are removed first.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

´CLOUD_DISCOVERY´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_ARCHIVAL´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_DYNAMODB_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_S3_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´EXPORT´ - Represents the set of permissions required to export an S3
    recovery to a newly created target bucket.
  * ´RECOVERY´ - Represents the set of elevated permissions required to perform
    recovery operations.

´EXOCOMPUTE´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RSC_MANAGED_CLUSTER´ - Represents the set of permissions required for the
    Rubrik-managed Exocompute cluster.

´KUBERNETES_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´RDS_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´ROLE_CHAINING´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´SERVERS_AND_APPS´
  * ´CLOUD_CLUSTER_ES´ - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the ´BASIC´ permission group
   is always required except for the ´SERVERS_AND_APPS´ feature.

-> **Note:** To onboard an account using a CloudFormation stack instead of IAM
   roles, use the ´polaris_aws_account´ resource.
`

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
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
			keyRoleChainingAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description: "RSC cloud account ID of the role chaining account. When specified, " +
					"the account will use cross-account role chaining.",
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
			keyTrustPolicies: {
				Type:     schema.TypeSet,
				Elem:     trustPolicyResource(),
				Computed: true,
				Description: "AWS IAM trust policies required by RSC. The ´policy´ field should be used with the " +
					"´assume_role_policy´ of the ´aws_iam_role´ resource.",
			},
		},
		CustomizeDiff: awsCustomizeDiffCnpAccount,
		Importer: &schema.ResourceImporter{
			StateContext: awsImportCnpAccount,
		},
	}
}

func awsCreateCnpAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloud := d.Get(keyCloud).(string)
	externalID := d.Get(keyExternalID).(string)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]any)
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

	var roleChainingAccountID uuid.UUID
	if id, ok := d.GetOk(keyRoleChainingAccountID); ok {
		roleChainingAccountID, err = uuid.Parse(id.(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	id, err := aws.Wrap(client).AddAccountWithIAM(ctx, aws.AccountWithName(cloud, nativeID, name), features, aws.Regions(regions...))
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := aws.Wrap(client).TrustPolicies(ctx, aws.TrustPoliciesParams{
		Cloud:                 gqlaws.Cloud(cloud),
		CloudAccountID:        id,
		Features:              features,
		ExternalID:            externalID,
		RoleChainingAccountID: roleChainingAccountID,
	}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	awsReadCnpAccount(ctx, d, m)
	return nil
}

func awsReadCnpAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := aws.Wrap(client).AccountByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// RSC enables some features on its own: CLOUD_COST_REPORT is added to any
	// account carrying a workload feature which accrues AWS spend, regardless of
	// the feature set passed when onboarding. Such a feature cannot be declared
	// in the feature block, so tracking it would show up as a diff removing it.
	// The diff applies, but it silently disables cost reporting for the account,
	// and it returns the next time the features are onboarded, as RSC adds cost
	// reporting again. Only the features which can be declared are tracked,
	// which also keeps import from writing an undeclarable feature into state.
	accountFeatures := slices.DeleteFunc(slices.Clone(account.Features), func(feature aws.Feature) bool {
		return !slices.Contains(awsCnpFeatureNames, feature.Name)
	})

	externalID := d.Get(keyExternalID).(string)
	features := make([]core.Feature, 0, len(accountFeatures))
	for _, feature := range accountFeatures {
		features = append(features, feature.Feature)
	}
	roleChainingAccountID := account.RoleChainingAccountID
	if roleChainingAccountID == uuid.Nil {
		if id, ok := d.GetOk(keyRoleChainingAccountID); ok {
			roleChainingAccountID, err = uuid.Parse(id.(string))
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	policies, err := aws.Wrap(client).TrustPolicies(ctx, aws.TrustPoliciesParams{
		Cloud:                 gqlaws.Cloud(account.Cloud),
		CloudAccountID:        id,
		Features:              features,
		ExternalID:            externalID,
		RoleChainingAccountID: roleChainingAccountID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyCloud, account.Cloud); err != nil {
		return diag.FromErr(err)
	}
	featureSet := &schema.Set{F: schema.HashResource(featureResource())}
	for _, feature := range accountFeatures {
		groups := &schema.Set{F: schema.HashString}
		for _, group := range feature.Feature.PermissionGroups {
			groups.Add(string(group))
		}
		featureSet.Add(map[string]any{
			keyName:             feature.Feature.Name,
			keyPermissionGroups: groups,
		})
	}
	if err := d.Set(keyFeature, featureSet); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, account.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyNativeID, account.NativeID); err != nil {
		return diag.FromErr(err)
	}
	regions := &schema.Set{F: schema.HashString}
	for _, feature := range accountFeatures {
		for _, region := range feature.Regions {
			regions.Add(region)
		}
	}
	if err := d.Set(keyRegions, regions); err != nil {
		return diag.FromErr(err)
	}
	if account.RoleChainingAccountID != uuid.Nil {
		if err := d.Set(keyRoleChainingAccountID, account.RoleChainingAccountID.String()); err != nil {
			return diag.FromErr(err)
		}
	}
	policySet := &schema.Set{F: schema.HashResource(trustPolicyResource())}
	for roleKey, policy := range policies {
		policySet.Add(map[string]any{
			keyRoleKey: roleKey,
			keyPolicy:  policy,
		})
	}
	if err := d.Set(keyTrustPolicies, policySet); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCnpAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cloud := d.Get(keyCloud).(string)
	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]any)
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

	if d.HasChange(keyName) {
		if err := aws.Wrap(client).UpdateAccount(ctx, id, core.FeatureAll, aws.Name(name)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(keyFeature) {
		oldAttr, newAttr := d.GetChange(keyFeature)

		var oldFeatures []core.Feature
		for _, block := range oldAttr.(*schema.Set).List() {
			block := block.(map[string]any)
			feature := core.Feature{Name: block[keyName].(string)}
			for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
			oldFeatures = append(oldFeatures, feature)
		}

		var newFeatures []core.Feature
		for _, block := range newAttr.(*schema.Set).List() {
			block := block.(map[string]any)
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
			if _, err := aws.Wrap(client).AddAccountWithIAM(ctx, account, updateFeatures, aws.Regions(regions...)); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(removeFeatures) > 0 {
			if err := aws.Wrap(client).RemoveAccountWithIAM(ctx, account, removeFeatures, deleteSnapshots); err != nil {
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
			if err := aws.Wrap(client).UpdateAccount(ctx, id, feature, aws.Regions(regions...)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	awsReadCnpAccount(ctx, d, m)
	return nil
}

func awsDeleteCnpAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCnpAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloud := d.Get(keyCloud).(string)
	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
	var features []core.Feature
	for _, feature := range d.Get(keyFeature).(*schema.Set).List() {
		feature := feature.(map[string]any)
		features = append(features, core.Feature{Name: feature[keyName].(string)})
	}
	name := d.Get(keyName).(string)
	nativeID := d.Get(keyNativeID).(string)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the account to find out which features RSC enabled on its own, see
	// featuresToRemove.
	account, err := aws.Wrap(client).AccountByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// RSC removes every feature on the account when passed an empty feature
	// list, so an empty result must not be handed to RemoveAccountWithIAM. What
	// is left on the account then belongs to RSC or to another resource.
	features = featuresToRemove(features, account)
	if len(features) == 0 {
		remaining := make([]string, 0, len(account.Features))
		for _, feature := range account.Features {
			remaining = append(remaining, feature.Name)
		}
		slices.Sort(remaining)

		d.SetId("")
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  "AWS account not removed",
			Detail: fmt.Sprintf("None of the features tracked in state are enabled on cloud account %s, so "+
				"nothing was removed and the account was left in RSC. Features still enabled on the account: %s.",
				id, strings.Join(remaining, ", ")),
		}}
	}

	if err := aws.Wrap(client).RemoveAccountWithIAM(ctx, aws.AccountWithName(cloud, nativeID, name), features, deleteSnapshots); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// featuresToRemove returns the features to remove when destroying an account:
// the declared features tracked in state which RSC still has on the account,
// plus CLOUD_COST_REPORT when RSC has it on the account.
//
// RSC removes the account once its last feature is removed, and it enables
// CLOUD_COST_REPORT on its own (see awsReadCnpAccount) without ever removing it
// along with the features it was enabled for, so removing only the declared
// features would leave cost reporting behind, and the account with it. The
// position of cost reporting in the result carries no meaning: it has no parent
// and no child features, and the SDK reorders the list before sending it,
// moving CLOUD_DISCOVERY last.
//
// Declared features which RSC no longer has, e.g. removed out of band, are
// dropped. RSC rejects the removal of a feature the account does not have,
// which would otherwise leave the resource impossible to destroy. The declared
// features are matched by name, so the permission groups tracked in state are
// kept.
//
// The other features RSC enables on its own are left to RSC. They are deletion
// children of the declared features, e.g. CLOUDACCOUNTS is a deletion child of
// CLOUD_NATIVE_PROTECTION, and are archived along with their parent. Naming
// them here would be redundant, as RSC skips a feature already archived by the
// time it is reached.
//
// The result is empty when RSC has none of the declared features and no cost
// reporting. Callers must not pass an empty feature list to RSC, see
// awsDeleteCnpAccount.
func featuresToRemove(declared []core.Feature, account aws.CloudAccount) []core.Feature {
	features := slices.DeleteFunc(slices.Clone(declared), func(feature core.Feature) bool {
		_, ok := account.Feature(feature)
		return !ok
	})
	if _, ok := account.Feature(core.FeatureCloudCostReport); ok {
		features = append(features, core.FeatureCloudCostReport)
	}

	return features
}

func awsCustomizeDiffCnpAccount(ctx context.Context, diff *schema.ResourceDiff, m any) error {
	tflog.Trace(ctx, "awsCustomizeDiffCnpAccount")

	// Prevent ROLE_CHAINING from being combined with other features.
	var features []core.Feature
	for _, block := range diff.Get(keyFeature).(*schema.Set).List() {
		features = append(features, core.Feature{Name: block.(map[string]any)[keyName].(string)})
	}
	if err := core.ValidateRoleChaining(features); err != nil {
		return err
	}

	if diff.HasChange(keyFeature) {
		if err := diff.SetNewComputed(keyTrustPolicies); err != nil {
			return err
		}

		// During update, prevent removing CLOUD_DISCOVERY while protection
		// features are still onboarded. The Cloud Discovery feature is
		// currently not required when onboarding protection features for a new
		// account.
		if diff.Id() != "" {
			oldBlock, newBlock := diff.GetChange(keyFeature)
			oldFeatures := oldBlock.(*schema.Set).List()
			newFeatures := newBlock.(*schema.Set).List()

			cloudDiscovery := func(feature any) bool {
				return feature.(map[string]any)[keyName].(string) == core.FeatureCloudDiscovery.Name
			}
			if slices.ContainsFunc(oldFeatures, cloudDiscovery) && !slices.ContainsFunc(newFeatures, cloudDiscovery) {
				protectionFeatures := []core.Feature{
					core.FeatureCloudNativeProtection,
					core.FeatureCloudNativeDynamoDBProtection,
					core.FeatureCloudNativeS3Protection,
					core.FeatureKubernetesProtection,
					core.FeatureRDSProtection,
				}
				for _, feature := range protectionFeatures {
					if slices.ContainsFunc(newFeatures, func(f any) bool {
						return f.(map[string]any)[keyName].(string) == feature.Name
					}) {
						return errors.New("CLOUD_DISCOVERY cannot be removed while protection features are enabled")
					}
				}
			}
		}
	}
	return nil
}

func awsImportCnpAccount(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	tflog.Trace(ctx, "awsImportCnpAccount")

	accountID, externalID, err := splitAccountID(d.Id())
	if err != nil {
		return nil, err
	}

	if externalID != "" {
		if err := d.Set(keyExternalID, externalID); err != nil {
			return nil, err
		}
	}

	d.SetId(accountID.String())
	return []*schema.ResourceData{d}, nil
}

// The external ID at the end is optional.
var reSplitAccountID = regexp.MustCompile("^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:-(.+))*$")

func splitAccountID(id string) (uuid.UUID, string, error) {
	match := reSplitAccountID.FindStringSubmatch(id)
	if len(match) != 2 && len(match) != 3 {
		return uuid.Nil, "", fmt.Errorf("invalid resource id: %s", id)
	}

	accountID, err := uuid.Parse(match[1])
	if err != nil {
		return uuid.Nil, "", err
	}
	var externalID string
	if len(match) == 3 {
		externalID = match[2]
	}

	return accountID, externalID, nil
}

func featureResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyName: {
				Type:     schema.TypeString,
				Required: true,
				Description: "RSC feature name. Possible values are `CLOUD_DISCOVERY`, `CLOUD_NATIVE_ARCHIVAL`, " +
					"`CLOUD_NATIVE_DYNAMODB_PROTECTION`, `CLOUD_NATIVE_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, " +
					"`EXOCOMPUTE`, `KUBERNETES_PROTECTION`, `RDS_PROTECTION`, `ROLE_CHAINING` and `SERVERS_AND_APPS`.",
				ValidateFunc: validation.StringInSlice(awsCnpFeatureNames, false),
			},
			keyPermissionGroups: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"BASIC", "RECOVERY", "RSC_MANAGED_CLUSTER", "CLOUD_CLUSTER_ES",
						"EXPORT", "EXPORT_POWER_ON", "EXPORT_POWER_OFF", "RESTORE",
						"DOWNLOAD_FILE",
						// The following permission groups cannot be used when onboarding an AWS account.
						// They have been accepted in the past so we still silently allow them.
						"EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY", "SNAPSHOT_PRIVATE_ACCESS", "PRIVATE_ENDPOINT",
					}, false),
				},
				Required: true,
				Description: "RSC permission groups for the feature. Possible values are `BASIC`, " +
					"`CLOUD_CLUSTER_ES`, `DOWNLOAD_FILE`, `EXPORT`, `EXPORT_POWER_ON`, `EXPORT_POWER_OFF`, " +
					"`RECOVERY`, `RESTORE` and `RSC_MANAGED_CLUSTER`. For backwards compatibility, `[]` is " +
					"interpreted as all applicable permission groups.",
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

func trustPolicyResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyRoleKey: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "RSC artifact key for the AWS role. Possible values are `CROSSACCOUNT`, " +
					"`EXOCOMPUTE_EKS_LAMBDA`, `EXOCOMPUTE_EKS_MASTERNODE`, `EXOCOMPUTE_EKS_WORKERNODE` and " +
					"`ROLE_CHAINING`.",
			},
			keyPolicy: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS IAM trust policy.",
			},
		},
	}
}
