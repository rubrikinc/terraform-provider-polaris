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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
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
									"BASIC",
									// The following permission groups cannot be used when onboarding an AWS account.
									// They have been accepted in the past so we still silently allow them.
									"EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY", "SNAPSHOT_PRIVATE_ACCESS",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Cloud Native Protection feature. " +
								"Possible values are `BASIC`.",
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
									"BASIC", "RSC_MANAGED_CLUSTER",
									// The following permission groups cannot be used when onboarding an AWS account.
									// They have been accepted in the past so we still silently allow them.
									"PRIVATE_ENDPOINT",
								}, false),
							},
							Optional: true,
							Description: "Permission groups to assign to the Exocompute feature. Possible values " +
								"are `BASIC` and `RSC_MANAGED_CLUSTER`.",
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
			keyDSPM: {
				Type:         schema.TypeList,
				RequiredWith: []string{keyOutpost},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the DSPM feature in.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the DSPM feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC",
								}, false),
							},
							Required: true,
							Description: "Permission groups to assign to the DSPM feature. " +
								"Possible values are `BASIC`.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the DSPM feature for the account.",
			},
			keyDataScanning: {
				Type:         schema.TypeList,
				RequiredWith: []string{keyOutpost},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the Data Scanning feature in.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Data Scanning feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC",
								}, false),
							},
							Required: true,
							Description: "Permission groups to assign to the Data Scanning feature. " +
								"Possible values are `BASIC`.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Data Scanning feature for the account.",
			},
			keyCyberRecoveryDataScanning: {
				Type:         schema.TypeList,
				RequiredWith: []string{keyOutpost},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRegions: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the Cyber Recovery Data Scanning feature in.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cyber Recovery Data Scanning feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC",
								}, false),
							},
							Required: true,
							Description: "Permission groups to assign to the Cyber Recovery Data Scanning feature. " +
								"Possible values are `BASIC`.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Cyber Recovery Data Scanning feature for the account.",
			},
			keyOutpost: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyOutpostAccountID: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "AWS account ID of the outpost account.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyOutpostAccountProfile: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "AWS named profile for the outpost account.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Outpost feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"BASIC",
								}, false),
							},
							Required: true,
							Description: "Permission groups to assign to the Outpost feature. " +
								"Possible values are `BASIC`.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Outpost feature for the account (Required for DSPM and Data Scanning features).",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
	tflog.Trace(ctx, "awsCreateAccount")

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

	outpostBlock, ok := d.GetOk(keyOutpost)
	if ok {
		block := outpostBlock.([]any)[0].(map[string]interface{})

		feature := core.FeatureOutpost
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var outpostOpts []aws.OptionFunc
		outpostAccountID := block[keyOutpostAccountID].(string)
		if outpostProfile := block[keyOutpostAccountProfile].(string); outpostProfile != "" {
			outpostOpts = append(outpostOpts, aws.OutpostAccountWithProfile(outpostAccountID, outpostProfile))
		} else {
			outpostOpts = append(outpostOpts, aws.OutpostAccount(outpostAccountID))
		}

		outpostOpts = append(outpostOpts, opts...)
		_, err := aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, outpostOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	dspmBlock, ok := d.GetOk(keyDSPM)
	if ok {
		block := dspmBlock.([]interface{})[0].(map[string]interface{})

		features := []core.Feature{core.FeatureDSPMData, core.FeatureDSPMMetadata}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			for i := range features {
				features[i] = features[i].WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
		}

		var dspmOpts []aws.OptionFunc
		for _, region := range block[keyRegions].(*schema.Set).List() {
			dspmOpts = append(dspmOpts, aws.Region(region.(string)))
		}

		dspmOpts = append(dspmOpts, opts...)
		_, err := aws.Wrap(client).AddAccount(ctx, account, features, dspmOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	dataScanningBlock, ok := d.GetOk(keyDataScanning)
	if ok {
		block := dataScanningBlock.([]interface{})[0].(map[string]interface{})

		features := []core.Feature{core.FeatureLaminarCrossAccount, core.FeatureLaminarInternal}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			for i := range features {
				features[i] = features[i].WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
		}

		var dataScanningOpts []aws.OptionFunc
		for _, region := range block[keyRegions].(*schema.Set).List() {
			dataScanningOpts = append(dataScanningOpts, aws.Region(region.(string)))
		}

		dataScanningOpts = append(dataScanningOpts, opts...)
		_, err := aws.Wrap(client).AddAccount(ctx, account, features, dataScanningOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	cyberRecoveryDataScanningBlock, ok := d.GetOk(keyCyberRecoveryDataScanning)
	if ok {
		block := cyberRecoveryDataScanningBlock.([]interface{})[0].(map[string]interface{})
		features := []core.Feature{core.FeatureCyberRecoveryDataClassificationData, core.FeatureCyberRecoveryDataClassificationMetadata}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			for i := range features {
				features[i] = features[i].WithPermissionGroups(core.PermissionGroup(group.(string)))
			}
		}
		var cyberRecoveryDataScanningOpts []aws.OptionFunc
		for _, region := range block[keyRegions].(*schema.Set).List() {
			cyberRecoveryDataScanningOpts = append(cyberRecoveryDataScanningOpts, aws.Region(region.(string)))
		}

		cyberRecoveryDataScanningOpts = append(cyberRecoveryDataScanningOpts, opts...)
		_, err := aws.Wrap(client).AddAccount(ctx, account, features, cyberRecoveryDataScanningOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(id.String())

	awsReadAccount(ctx, d, m)
	return nil
}

func awsReadAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadAccount")

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

	// Handle DSPM features
	dspmDataFeature, hasDSPMData := account.Feature(core.FeatureDSPMData)
	_, hasDSPMMetadata := account.Feature(core.FeatureDSPMMetadata)
	if hasDSPMData && hasDSPMMetadata {
		regions := schema.Set{F: schema.HashString}
		for _, region := range dspmDataFeature.Regions {
			regions.Add(region)
		}

		groups := schema.Set{F: schema.HashString}
		for _, group := range dspmDataFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		status := core.FormatStatus(dspmDataFeature.Status)

		err := d.Set(keyDSPM, []interface{}{
			map[string]interface{}{
				keyPermissionGroups: &groups,
				keyRegions:          &regions,
				keyStatus:           &status,
				keyStackARN:         &dspmDataFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyDSPM, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	// Handle Data Scanning features (Laminar)
	laminarCrossAccountFeature, hasLaminarCrossAccount := account.Feature(core.FeatureLaminarCrossAccount)
	_, hasLaminarInternal := account.Feature(core.FeatureLaminarInternal)
	if hasLaminarCrossAccount && hasLaminarInternal {

		regions := schema.Set{F: schema.HashString}
		for _, region := range laminarCrossAccountFeature.Regions {
			regions.Add(region)
		}

		groups := schema.Set{F: schema.HashString}
		for _, group := range laminarCrossAccountFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		status := core.FormatStatus(laminarCrossAccountFeature.Status)

		err := d.Set(keyDataScanning, []interface{}{
			map[string]interface{}{
				keyPermissionGroups: &groups,
				keyRegions:          &regions,
				keyStatus:           &status,
				keyStackARN:         &laminarCrossAccountFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyDataScanning, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	// Handle Cyber Recovery Data Scanning
	cyberRecoveryDataScanningFeature, hasCyberRecoveryDataScanning := account.Feature(core.FeatureCyberRecoveryDataClassificationData)
	_, hasCyberRecoveryDataScanningMetadata := account.Feature(core.FeatureCyberRecoveryDataClassificationMetadata)

	if hasCyberRecoveryDataScanning && hasCyberRecoveryDataScanningMetadata {
		regions := schema.Set{F: schema.HashString}
		for _, region := range cyberRecoveryDataScanningFeature.Regions {
			regions.Add(region)
		}

		groups := schema.Set{F: schema.HashString}
		for _, group := range cyberRecoveryDataScanningFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		status := core.FormatStatus(cyberRecoveryDataScanningFeature.Status)

		err := d.Set(keyCyberRecoveryDataScanning, []interface{}{
			map[string]interface{}{
				keyPermissionGroups: &groups,
				keyRegions:          &regions,
				keyStatus:           &status,
				keyStackARN:         &cyberRecoveryDataScanningFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyCyberRecoveryDataScanning, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	outpostFeature, hasOutpost := account.Feature(core.FeatureOutpost)
	if hasOutpost {
		groups := schema.Set{F: schema.HashString}
		for _, group := range outpostFeature.Feature.PermissionGroups {
			groups.Add(string(group))
		}

		status := core.FormatStatus(outpostFeature.Status)

		// Get outpost account details
		statusFilters := []core.Status{core.StatusConnected, core.StatusMissingPermissions}
		outpostAccounts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "", statusFilters)
		if err != nil {
			return diag.FromErr(err)
		}

		outpostData := map[string]interface{}{
			keyPermissionGroups:      &groups,
			keyStatus:                &status,
			keyStackARN:              &outpostFeature.StackArn,
			keyOutpostAccountProfile: d.Get(keyOutpostAccountProfile),
		}

		// Add outpost account ID if available
		if len(outpostAccounts) > 0 {
			outpostData[keyOutpostAccountID] = outpostAccounts[0].NativeID
		}

		err = d.Set(keyOutpost, []interface{}{outpostData})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set(keyOutpost, nil); err != nil {
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
	tflog.Trace(ctx, "awsUpdateAccount")

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

		err := updateToNewBlock(ctx, d, m, client, id, oldExoList, newExoList, account, core.FeatureExocompute)
		if err != nil {
			return diag.FromErr(err)
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

	// Handle Outpost, DSPM, and Data Scanning features with proper ordering
	if err := handleDspmFeatures(ctx, d, m, client, account, id); err != nil {
		return diag.FromErr(err)
	}

	awsReadAccount(ctx, d, m)
	return nil
}

// handleDspmFeatures handles Outpost, DSPM, and Data Scanning features with proper ordering.
// When adding features: Outpost first, then DSPM and Data Scanning
// When removing features: DSPM and Data Scanning first, then Outpost last (due to mapped account dependencies)
func handleDspmFeatures(ctx context.Context, d *schema.ResourceData, m interface{}, client *polaris.Client, account aws.AccountFunc, id uuid.UUID) error {
	// Check which features have changes
	hasOutpostChange := d.HasChange(keyOutpost)
	hasDSPMChange := d.HasChange(keyDSPM)
	hasDataScanningChange := d.HasChange(keyDataScanning)
	hasCyberRecoveryDataScanningChange := d.HasChange(keyCyberRecoveryDataScanning)

	if !hasOutpostChange && !hasDSPMChange && !hasDataScanningChange && !hasCyberRecoveryDataScanningChange {
		return nil
	}

	_, outpostExists := d.GetOk(keyOutpost)
	_, dspmExists := d.GetOk(keyDSPM)
	_, dataScanningExists := d.GetOk(keyDataScanning)
	_, cyberRecoveryDataScanningExists := d.GetOk(keyCyberRecoveryDataScanning)

	isAddingFeatures := (hasOutpostChange && outpostExists) ||
		(hasDSPMChange && dspmExists) ||
		(hasDataScanningChange && dataScanningExists) ||
		(hasCyberRecoveryDataScanningChange && cyberRecoveryDataScanningExists)

	isRemovingFeatures := (hasOutpostChange && !outpostExists) ||
		(hasDSPMChange && !dspmExists) ||
		(hasDataScanningChange && !dataScanningExists) ||
		(hasCyberRecoveryDataScanningChange && !cyberRecoveryDataScanningExists)

	if isAddingFeatures {
		// When adding: Outpost first, then DSPM and Data Scanning
		if hasOutpostChange && outpostExists {
			if err := handleOutpostUpdate(ctx, d, client, account); err != nil {
				return err
			}
		}
		if hasDSPMChange && dspmExists {
			if err := handleDSPMUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
		if hasDataScanningChange && dataScanningExists {
			if err := handleDataScanningUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
		if hasCyberRecoveryDataScanningChange && cyberRecoveryDataScanningExists {
			if err := handleCyberRecoveryDataScanningUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
	}

	if isRemovingFeatures {
		// When removing: DSPM and Data Scanning first, then Outpost last
		if hasDSPMChange && !dspmExists {
			if err := handleDSPMUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
		if hasDataScanningChange && !dataScanningExists {
			if err := handleDataScanningUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
		if hasCyberRecoveryDataScanningChange && !cyberRecoveryDataScanningExists {
			if err := handleCyberRecoveryDataScanningUpdate(ctx, d, m, client, id, account); err != nil {
				return err
			}
		}
		if hasOutpostChange && !outpostExists {
			if err := handleOutpostUpdate(ctx, d, client, account); err != nil {
				return err
			}
		}
	}

	return nil
}

func handleCyberRecoveryDataScanningUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, client *polaris.Client, id uuid.UUID, account aws.AccountFunc) error {
	features := []core.Feature{core.FeatureCyberRecoveryDataClassificationData, core.FeatureCyberRecoveryDataClassificationMetadata}
	oldCyberRecoveryDataScanningBlock, newCyberRecoveryDataScanningBlock := d.GetChange(keyCyberRecoveryDataScanning)
	oldCyberRecoveryDataScanningList := oldCyberRecoveryDataScanningBlock.([]interface{})
	newCyberRecoveryDataScanningList := newCyberRecoveryDataScanningBlock.([]interface{})
	for _, feature := range features {
		if err := updateToNewBlock(ctx, d, m, client, id, oldCyberRecoveryDataScanningList, newCyberRecoveryDataScanningList, account, feature); err != nil {
			return err
		}
	}

	return nil
}

func handleOutpostUpdate(ctx context.Context, d *schema.ResourceData, client *polaris.Client, account aws.AccountFunc) error {
	outpostBlock, ok := d.GetOk(keyOutpost)
	if ok {
		block := outpostBlock.([]interface{})[0].(map[string]interface{})
		feature := core.FeatureOutpost
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var opts []aws.OptionFunc
		if block[keyOutpostAccountProfile] != nil {
			opts = append(opts, aws.OutpostAccountWithProfile(block[keyOutpostAccountID].(string), block[keyOutpostAccountProfile].(string)))
		} else {
			opts = append(opts, aws.OutpostAccount(block[keyOutpostAccountID].(string)))
		}
		_, err := aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, opts...)
		if err != nil {
			return err
		}
	} else {
		accounts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "", []core.Status{core.StatusConnected, core.StatusMissingPermissions})
		if err != nil {
			return err
		}

		for _, account := range accounts {
			for _, feature := range account.Features {
				if len(feature.MappedAccounts) > 0 {
					return errors.New("outpost feature is still enabled for other accounts")
				}
			}
		}
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureOutpost}, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleDSPMUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, client *polaris.Client, id uuid.UUID, account aws.AccountFunc) error {
	features := []core.Feature{core.FeatureDSPMData, core.FeatureDSPMMetadata}
	oldDspmBlock, newDspmBlock := d.GetChange(keyDSPM)
	oldDspmList := oldDspmBlock.([]interface{})
	newDspmList := newDspmBlock.([]interface{})

	for _, feature := range features {
		if err := updateToNewBlock(ctx, d, m, client, id, oldDspmList, newDspmList, account, feature); err != nil {
			return err
		}
	}

	return nil
}

func handleDataScanningUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, client *polaris.Client, id uuid.UUID, account aws.AccountFunc) error {
	features := []core.Feature{core.FeatureLaminarCrossAccount, core.FeatureLaminarInternal}
	oldDataScanningBlock, newDataScanningBlock := d.GetChange(keyDataScanning)
	oldDataScanningList := oldDataScanningBlock.([]interface{})
	newDataScanningList := newDataScanningBlock.([]interface{})

	for _, feature := range features {
		if err := updateToNewBlock(ctx, d, m, client, id, oldDataScanningList, newDataScanningList, account, feature); err != nil {
			return err
		}
	}

	return nil
}

func updateToNewBlock(ctx context.Context, d *schema.ResourceData, m interface{}, client *polaris.Client, id uuid.UUID, oldBlock, newBlock []interface{}, account aws.AccountFunc, feature core.Feature) error {
	switch {
	case len(oldBlock) == 0:
		for _, group := range newBlock[0].(map[string]interface{})[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var opts []aws.OptionFunc
		for _, region := range newBlock[0].(map[string]interface{})[keyRegions].(*schema.Set).List() {
			opts = append(opts, aws.Region(region.(string)))
		}

		_, err := aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, opts...)
		if err != nil {
			return err
		}
	case len(newBlock) == 0:
		err := aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{feature}, false)
		if err != nil {
			return err
		}
	default:
		var opts []aws.OptionFunc
		for _, region := range newBlock[0].(map[string]interface{})[keyRegions].(*schema.Set).List() {
			opts = append(opts, aws.Region(region.(string)))
		}

		err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), feature, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func awsDeleteAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteAccount")

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

	if _, ok := d.GetOk(keyDSPM); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureDSPMData, core.FeatureDSPMMetadata}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk(keyDataScanning); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureLaminarCrossAccount, core.FeatureLaminarInternal}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if _, ok := d.GetOk(keyCyberRecoveryDataScanning); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureCyberRecoveryDataClassificationData, core.FeatureCyberRecoveryDataClassificationMetadata}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Outpost should always delete last due to its dependancy on mapped accounts
	if _, ok := d.GetOk(keyOutpost); ok {
		accounts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "", []core.Status{core.StatusConnected, core.StatusMissingPermissions})
		if err != nil {
			return diag.FromErr(err)
		}

		for _, account := range accounts {
			for _, feature := range account.Features {
				if len(feature.MappedAccounts) > 0 {
					return diag.Errorf("outpost feature is still enabled for other accounts")
				}
			}
		}
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureOutpost}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}
