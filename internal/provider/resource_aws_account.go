// Copyright 2021 Rubrik, Inc.
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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAWSAccountDescription = `
The ´polaris_aws_account´ resource adds an AWS account to RSC. To grant RSC
permissions to perform certain operations on the account, a Cloud Formation stack
is created from a template provided by RSC.

There are two ways to specify the AWS account to onboard:
 1. Using the ´profile´ field. The AWS profile is used to create the Cloud
    Formation stack and lookup the AWS account ID.
 2. Using the ´assume_role´field with, or without, the ´profile´ field. If the
    ´profile´ field is omitted, the default profile is used. The profile is used
    to assume the role. The assumed role is then used and create the Cloud
    Formation stack and lookup the account ID.

Any combination of different RSC features can be enabled for an account:
  1. ´cloud_native_protection´ - Provides protection for AWS EC2 instances and
     EBS volumes through the rules and policies of SLA Domains.
  2. ´exocompute´ - Provides snapshot indexing, file recovery and application
     protection of AWS objects.
`

func resourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAccount,
		ReadContext:   awsReadAccount,
		UpdateContext: awsUpdateAccount,
		DeleteContext: awsDeleteAccount,

		Description: description(resourceAWSAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyAssumeRole: {
				Type:             schema.TypeString,
				Optional:         true,
				AtLeastOneOf:     []string{keyProfile},
				Description:      "Role ARN of role to assume.",
				ValidateDiagFunc: validateRoleARN,
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY", "SNAPSHOT_PRIVATE_ACCESS",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Protection feature. " +
								"Possible values are `BASIC`, `EXPORT_AND_RESTORE`, `FILE_LEVEL_RECOVERY` and " +
								"`SNAPSHOT_PRIVATE_ACCESS`.",
						},
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions that RSC will monitor for instances to automatically protect.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
					},
				},
				MaxItems:    1,
				Required:    true,
				Description: "Enable the Cloud Native Protection feature for the AWS account.",
			},
			keyDeleteSnapshotsOnDestroy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			keyExocompute: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC", "PRIVATE_ENDPOINT", "RSC_MANAGED_CLUSTER",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Exocompute feature. Possible values " +
								"are `BASIC`, `PRIVATE_ENDPOINT` and `RSC_MANAGED_CLUSTER`.",
						},
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the Exocompute feature in.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Exocompute feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Exocompute feature for the account.",
			},
			keyName: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: "Account name in Polaris. If not given the name is taken from AWS Organizations " +
					"or, if the required permissions are missing, is derived from the AWS account ID and the " +
					"named profile.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermissions: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: "When set to 'update' feature permissions can be updated by applying the " +
					"configuration.",
				ValidateDiagFunc: validatePermissions,
			},
			keyProfile: {
				Type:         schema.TypeString,
				Optional:     true,
				AtLeastOneOf: []string{keyAssumeRole},
				Description:  "AWS named profile.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},

		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAwsAccountV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAwsAccountStateUpgradeV0,
			Version: 0,
		}, {
			Type:    resourceAwsAccountV1().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAwsAccountStateUpgradeV1,
			Version: 1,
		}},
	}
}

func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Initialize to empty string if missing from the configuration.
	profile, _ := d.Get(keyProfile).(string)
	roleARN, _ := d.Get(keyAssumeRole).(string)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "" && roleARN != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	var opts []aws.OptionFunc
	if name, ok := d.GetOk(keyName); ok {
		opts = append(opts, aws.Name(name.(string)))
	}

	// Polaris Cloud Account id. Returned when the account is added for the
	// cloud native protection feature.
	var id uuid.UUID

	cnpBlock, ok := d.GetOk(keyCloudNativeProtection)
	if ok {
		block := cnpBlock.([]interface{})[0].(map[string]interface{})

		feature := core.FeatureCloudNativeProtection
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var cnpOpts []aws.OptionFunc
		for _, region := range block[keyRegions].(*schema.Set).List() {
			cnpOpts = append(cnpOpts, aws.Region(region.(string)))
		}

		var err error
		cnpOpts = append(cnpOpts, opts...)
		id, err = aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, cnpOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	exoBlock, ok := d.GetOk(keyExocompute)
	if ok {
		block := exoBlock.([]interface{})[0].(map[string]interface{})

		feature := core.FeatureExocompute
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var exoOpts []aws.OptionFunc
		for _, region := range block[keyRegions].(*schema.Set).List() {
			exoOpts = append(exoOpts, aws.Region(region.(string)))
		}

		exoOpts = append(exoOpts, opts...)
		_, err := aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, exoOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(id.String())

	awsReadAccount(ctx, d, m)
	return nil
}

func awsReadAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the Polaris cloud account using the cloud account id.
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	cnpFeature, ok := account.Feature(core.FeatureCloudNativeProtection)
	if ok {
		groups := schema.Set{F: schema.HashString}
		for _, group := range cnpFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		regions := schema.Set{F: schema.HashString}
		for _, region := range cnpFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(cnpFeature.Status)
		err := d.Set("cloud_native_protection", []interface{}{
			map[string]interface{}{
				keyPermissionGroups: &groups,
				keyRegions:          &regions,
				keyStatus:           &status,
				keyStackARN:         &cnpFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyCloudNativeProtection, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	exoFeature, ok := account.Feature(core.FeatureExocompute)
	if ok {
		groups := schema.Set{F: schema.HashString}
		for _, group := range exoFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		regions := schema.Set{F: schema.HashString}
		for _, region := range exoFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(exoFeature.Status)
		err := d.Set("exocompute", []interface{}{
			map[string]interface{}{
				keyPermissionGroups: &groups,
				keyRegions:          &regions,
				keyStatus:           &status,
				keyStackARN:         &exoFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyExocompute, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, account.Name); err != nil {
		return diag.FromErr(err)
	}

	// Check if any feature is missing permissions.
	for _, feature := range account.Features {
		if feature.Status != core.StatusMissingPermissions {
			continue
		}

		if err := d.Set(keyPermissions, "update-required"); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Initialize to empty string if missing from the configuration.
	profile, _ := d.Get(keyProfile).(string)
	roleARN, _ := d.Get(keyAssumeRole).(string)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "" && roleARN != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Make sure that the resource id and AWS profile refers to the same
	// account.
	cloudAccount, err := aws.Wrap(client).Account(ctx, aws.ID(account), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		return diag.Errorf("account identified by profile/role could not be found")
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if cloudAccount.ID != id {
		return diag.Errorf("resource id and profile/role refer to different accounts")
	}

	if d.HasChange(keyCloudNativeProtection) {
		cnpBlock, ok := d.GetOk(keyCloudNativeProtection)
		if ok {
			block := cnpBlock.([]interface{})[0].(map[string]interface{})

			feature := core.FeatureCloudNativeProtection
			for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}

			var opts []aws.OptionFunc
			for _, region := range block[keyRegions].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			if err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), feature, opts...); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if _, ok := d.GetOk(keyExocompute); ok {
				return diag.Errorf("cloud native protection is required by exocompute")
			}

			snapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
			if err := aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureCloudNativeProtection}, snapshots); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keyExocompute) {
		oldExoBlock, newExoBlock := d.GetChange(keyExocompute)
		oldExoList := oldExoBlock.([]interface{})
		newExoList := newExoBlock.([]interface{})

		// Determine whether we are adding, removing or updating the Exocompute
		// feature.
		switch {
		case len(oldExoList) == 0:
			feature := core.FeatureExocompute
			for _, group := range newExoList[0].(map[string]interface{})[keyPermissionGroups].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}

			var opts []aws.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})[keyRegions].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			_, err = aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		case len(newExoList) == 0:
			err := aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureExocompute}, false)
			if err != nil {
				return diag.FromErr(err)
			}
		default:
			var opts []aws.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})[keyRegions].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			err = aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keyPermissions) {
		oldPerms, newPerms := d.GetChange(keyPermissions)

		if oldPerms == "update-required" && newPerms == "update" {
			var features []core.Feature
			for _, feature := range cloudAccount.Features {
				if feature.Status != core.StatusMissingPermissions {
					continue
				}
				features = append(features, feature.Feature)
			}

			err := aws.Wrap(client).UpdatePermissions(ctx, account, features)
			if err != nil {
				return diag.FromErr(err)
			}

			if err := d.Set(keyPermissions, "update"); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	awsReadAccount(ctx, d, m)
	return nil
}

func awsDeleteAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the old resource arguments. Initialize to empty string if missing
	// from the configuration.
	oldProfile, _ := d.GetChange(keyProfile)
	oldRoleARN, _ := d.GetChange(keyAssumeRole)
	profile, _ := oldProfile.(string)
	roleARN, _ := oldRoleARN.(string)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "" && roleARN != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	oldSnapshots, _ := d.GetChange(keyDeleteSnapshotsOnDestroy)
	deleteSnapshots := oldSnapshots.(bool)

	// Make sure that the resource id and account profile refers to the same
	// account.
	cloudAccount, err := aws.Wrap(client).Account(ctx, aws.ID(account), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		return diag.Errorf("account identified by profile/role could not be found")
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if cloudAccount.ID != id {
		return diag.Errorf("resource id and profile/role refer to different accounts")
	}

	if _, ok := d.GetOk(keyExocompute); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureExocompute}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk(keyCloudNativeProtection); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureCloudNativeProtection}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}
