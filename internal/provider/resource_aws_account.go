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
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"
)

const resourceAWSAccountDescription = `
The ´polaris_aws_account´ resource adds an AWS account to RSC. To grant RSC
permissions to perform certain operations on the account, a Cloud Formation
stack is created from a template provided by RSC.

There are two ways to specify the AWS account to onboard:
 1. Using the ´profile´ field. The AWS profile is used to create the Cloud
    Formation stack and lookup the AWS account ID.
 2. Using the ´assume_role´ field with, or without, the ´profile´ field. If the
    ´profile´ field is omitted, the default profile is used. The profile is used
    to assume the role. The assumed role is then used and create the Cloud
    Formation stack and lookup the account ID.

Any combination of different RSC features, except as noted below, can be enabled
for an account:
  * ´cloud_discovery´ - Enable the Cloud Discovery feature for the account.
    Required when onboarding a new account with protection features. Optional
    for existing accounts.
  * ´cloud_native_archival´ - Enable the Cloud Native Archival feature for the
    account.
  * ´cloud_native_protection´ - Enable the Cloud Native Protection feature for
    the account.
  * ´cloud_native_dynamodb_protection´ - Enable the Cloud Native DynamoDB
    Protection feature for the account.
  * ´cloud_native_s3_protection´ - Enable the Cloud Native S3 Protection feature
    for the account.
  * ´cyber_recovery_data_scanning´ - Enable the Cyber Recovery Data Scanning
    feature for the account. Requires the Outpost feature to be enabled, either
    in the same account or in a separate account onboarded before this resource.
  * ´data_scanning´ - Enable the Data Scanning feature for the account. Requires
    the Outpost feature to be enabled, either in the same account or in a
    separate account onboarded before this resource.
  * ´dspm´ - Enable the DSPM feature for the account. Requires the Outpost
    feature to be enabled, either in the same account or in a separate account
    onboarded before this resource.
  * ´exocompute´ - Enable the Exocompute feature for the account. Only required
    for accounts hosting the Exocompute cluster.
  * ´kubernetes_protection´ - Enable the Kubernetes Protection feature for the
    account.
  * ´outpost´ - Enable the Outpost feature for the account. Required for the
    Cyber Recovery Data Scanning, Data Scanning and DSPM features. The outpost
    account can be the same account as where the features are enabled, or it can
    be a separate account. When using a separate account, the outpost account
    must be onboarded first, using ´depends_on´ to enforce the ordering. The
    ´outpost_account_id´ and ´outpost_account_profile´ fields are legacy and not
    recommended.
  * ´rds_protection´ - Enable the RDS Protection feature for the account.
  * ´servers_and_apps´ - Enable the Servers and Apps feature for the account.
    Required to run CCES clusters.

## Outpost Account

The Cyber Recovery Data Scanning, Data Scanning and DSPM features require an
outpost account to be onboarded. The outpost account can be the same account as
where the features are enabled, or it can be a separate account.

When the outpost account is the same account:
´´´terraform
resource "polaris_aws_account" "main" {
  profile = "main"

  data_scanning {
    permission_groups = ["BASIC"]
    regions           = ["us-east-2"]
  }

  outpost {
    permission_groups = ["BASIC"]
  }
}
´´´

When the outpost account is a separate account, the outpost account must be
onboarded first. Use ´depends_on´ to enforce the ordering:
´´´terraform
resource "polaris_aws_account" "outpost" {
  profile = "outpost"

  outpost {
    permission_groups = ["BASIC"]
  }
}

resource "polaris_aws_account" "main" {
  profile = "main"

  data_scanning {
    permission_groups = ["BASIC"]
    regions           = ["us-east-2"]
  }

  depends_on = [
    polaris_aws_account.outpost,
  ]
}
´´´

-> **Note:** To onboard an account using IAM roles instead of a CloudFormation
   stack, use the ´polaris_aws_cnp_account´ resource.

-> **Note:** When importing the ´polaris_aws_account´ resource, the
   ´outpost_account_id´ and ´outpost_account_profile´ fields are not imported.
`

func resourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAccount,
		ReadContext:   awsReadAccount,
		UpdateContext: awsUpdateAccount,
		DeleteContext: awsDeleteAccount,
		CustomizeDiff: awsCustomizeDiffAccount,

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
			keyCloudDiscovery: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Cloud Discovery feature for the account.",
			},
			keyCloudNativeArchival: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Cloud Native Archival feature for the account.",
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: awsCFTFeatureResource([]core.PermissionGroup{
					core.PermissionGroupBasic,
					// The following permission groups cannot be used when onboarding an AWS account.
					// They have been accepted in the past so we still silently allow them.
					core.PermissionGroupExportAndRestore,
					core.PermissionGroupFileLevelRecovery,
					core.PermissionGroupSnapshotPrivateAccess,
				}),
				MaxItems: 1,
				Optional: true,
				AtLeastOneOf: []string{
					keyCloudDiscovery,
					keyCloudNativeArchival,
					keyCloudNativeDynamoDBProtection,
					keyCloudNativeS3Protection,
					keyCyberRecoveryDataScanning,
					keyDataScanning,
					keyDSPM,
					keyExocompute,
					keyKubernetesProtection,
					keyOutpost,
					keyRDSProtection,
					keyServersAndApps,
				},
				Description: "Enable the Cloud Native Protection feature for the account.",
			},
			keyCloudNativeDynamoDBProtection: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Cloud Native DynamoDB Protection feature for the account.",
			},
			keyCloudNativeS3Protection: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Cloud Native S3 Protection feature for the account.",
			},
			keyCyberRecoveryDataScanning: {
				Type:     schema.TypeList,
				Elem:     awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems: 1,
				Optional: true,
				Description: "Enable the Cyber Recovery Data Scanning feature for the account. The Cyber Recovery " +
					"Data Scanning feature requires the Outpost feature to be enabled.",
			},
			keyDeleteSnapshotsOnDestroy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			keyDataScanning: {
				Type:     schema.TypeList,
				Elem:     awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems: 1,
				Optional: true,
				Description: "Enable the Data Scanning feature for the account. The Data Scanning feature requires " +
					"the Outpost feature to be enabled.",
			},
			keyDSPM: {
				Type:     schema.TypeList,
				Elem:     awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems: 1,
				Optional: true,
				Description: "Enable the DSPM feature for the account. The DSPM feature requires the Outpost " +
					"feature to be enabled.",
			},
			keyExocompute: {
				Type: schema.TypeList,
				Elem: awsCFTFeatureResource([]core.PermissionGroup{
					core.PermissionGroupBasic,
					core.PermissionGroupRSCManagedCluster,
					// The following permission groups cannot be used when onboarding an AWS account.
					// They have been accepted in the past so we still silently allow them.
					core.PermissionGroupPrivateEndpoints,
				}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Exocompute feature for the account.",
			},
			keyKubernetesProtection: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Kubernetes Protection feature for the AWS account.",
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
			keyOutpost: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyOutpostAccountID: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "AWS account ID of the outpost account. Defaults to the current account.",
							ValidateFunc: validateAwsAccountID,
						},
						keyOutpostAccountProfile: {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{keyOutpost + ".0." + keyOutpostAccountID},
							Description: "AWS named profile for the outpost account. Defaults to the profile used " +
								"for the current account.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyPermissionGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"BASIC"}, false),
							},
							Required: true,
							Description: "Permission groups to assign to the Outpost feature. Possible values are " +
								"`BASIC`.",
						},
						keyStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Outpost feature.",
						},
						keyStackARN: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "CloudFormation stack ARN.",
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				Description: "Enable the Outpost feature for the account. To use the DSPM, Data Scanning and Cyber " +
					"Recovery Data Scanning features, one account must have the Outpost feature enabled.",
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
			keyRDSProtection: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupBasic}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the RDS Protection feature for the account.",
			},
			keyServersAndApps: {
				Type:        schema.TypeList,
				Elem:        awsCFTFeatureResource([]core.PermissionGroup{core.PermissionGroupCCES}),
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the Servers and Apps feature for the account.",
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

func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	profile := d.Get(keyProfile).(string)
	roleARN := d.Get(keyAssumeRole).(string)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	// Account name.
	var opts []aws.OptionFunc
	if name, ok := d.GetOk(keyName); ok {
		opts = append(opts, aws.Name(name.(string)))
	}

	// Collect features and regions from the resource data.
	var featureBlocks []awsCFTFeatureBlock

	featureBlock, err := awsFromCFTFeatureBlock(keyCloudDiscovery, d.Get(keyCloudDiscovery))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyCloudNativeArchival, d.Get(keyCloudNativeArchival))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyCloudNativeProtection, d.Get(keyCloudNativeProtection))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyCloudNativeDynamoDBProtection, d.Get(keyCloudNativeDynamoDBProtection))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyCloudNativeS3Protection, d.Get(keyCloudNativeS3Protection))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyCyberRecoveryDataScanning, d.Get(keyCyberRecoveryDataScanning))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyDataScanning, d.Get(keyDataScanning))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyDSPM, d.Get(keyDSPM))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyExocompute, d.Get(keyExocompute))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyKubernetesProtection, d.Get(keyKubernetesProtection))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyOutpost, d.Get(keyOutpost))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyRDSProtection, d.Get(keyRDSProtection))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	featureBlock, err = awsFromCFTFeatureBlock(keyServersAndApps, d.Get(keyServersAndApps))
	if err != nil {
		return diag.FromErr(err)
	}
	if featureBlock != nil {
		featureBlocks = append(featureBlocks, *featureBlock)
	}

	// Verify that outpost-dependent features have an outpost account available,
	// either in this resource or already onboarded in RSC.
	if outpostBlock := d.Get(keyOutpost).([]any); len(outpostBlock) == 0 {
		outpostDependentKeys := []string{keyCyberRecoveryDataScanning, keyDataScanning, keyDSPM}
		var hasOutpostDependentFeature bool
		for _, key := range outpostDependentKeys {
			if block := d.Get(key).([]any); len(block) > 0 {
				hasOutpostDependentFeature = true
				break
			}
		}
		if hasOutpostDependentFeature {
			outposts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "",
				[]core.Status{core.StatusConnected, core.StatusMissingPermissions})
			if err != nil {
				return diag.Errorf("failed to check outpost account status: %s", err)
			}
			if len(outposts) == 0 {
				return diag.Errorf("cyber_recovery_data_scanning, data_scanning, and dspm features require an outpost account")
			}
		}
	}

	// Add cloud account with features.
	var id uuid.UUID
	for _, feature := range awsSquashCFTFeatureBlocks(featureBlocks) {
		featureOpts := slices.Clone(opts)
		for _, region := range feature.regions {
			featureOpts = append(featureOpts, aws.Region(region.Name()))
		}
		if outpostID := feature.outpostID; outpostID != "" {
			if outpostProfile := feature.outpostProfile; outpostProfile != "" {
				featureOpts = append(featureOpts, aws.OutpostAccountWithProfile(outpostID, outpostProfile))
			} else {
				featureOpts = append(featureOpts, aws.OutpostAccount(outpostID))
			}
		}

		id, err = aws.Wrap(client).AddAccountWithCFT(ctx, account, feature.features, featureOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(id.String())
	awsReadAccount(ctx, d, m)
	return nil
}

func awsReadAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
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
	account, err := aws.Wrap(client).AccountByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// Regular features.
	if err := d.Set(keyCloudDiscovery, awsToCFTFeatureBlock(account, keyCloudDiscovery)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudNativeArchival, awsToCFTFeatureBlock(account, keyCloudNativeArchival)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudNativeProtection, awsToCFTFeatureBlock(account, keyCloudNativeProtection)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudNativeDynamoDBProtection, awsToCFTFeatureBlock(account, keyCloudNativeDynamoDBProtection)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudNativeS3Protection, awsToCFTFeatureBlock(account, keyCloudNativeS3Protection)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCyberRecoveryDataScanning, awsToCFTFeatureBlock(account, keyCyberRecoveryDataScanning)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDataScanning, awsToCFTFeatureBlock(account, keyDataScanning)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDSPM, awsToCFTFeatureBlock(account, keyDSPM)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyExocompute, awsToCFTFeatureBlock(account, keyExocompute)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyKubernetesProtection, awsToCFTFeatureBlock(account, keyKubernetesProtection)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRDSProtection, awsToCFTFeatureBlock(account, keyRDSProtection)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyServersAndApps, awsToCFTFeatureBlock(account, keyServersAndApps)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, account.Name); err != nil {
		return diag.FromErr(err)
	}

	// Outpost feature. Note the outpost account can be separate from the cloud
	// account managed by the resource.
	var outpostBlock []any
	outpostFeatureBlock, err := awsFromCFTFeatureBlock(keyOutpost, d.Get(keyOutpost))
	if err != nil {
		return diag.FromErr(err)
	}
	if outpostFeatureBlock != nil && outpostFeatureBlock.outpostID != "" {
		outposts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "",
			[]core.Status{core.StatusConnected, core.StatusMissingPermissions})
		if err != nil {
			return diag.FromErr(err)
		}
		if len(outposts) > 0 {
			outpostBlock = awsToCFTOutpostBlock(outposts[0], outpostFeatureBlock.outpostProfile, true)
		}
	} else {
		outpostBlock = awsToCFTOutpostBlock(account, "", false)
	}
	if err := d.Set(keyOutpost, outpostBlock); err != nil {
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

func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	profile := d.Get(keyProfile).(string)
	roleARN := d.Get(keyAssumeRole).(string)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Check that the resource ID and AWS profile refers to the same account.
	config, err := account(ctx)
	if err != nil {
		return diag.Errorf("failed to lookup native account id: %s", err)
	}
	cloudAccount, err := aws.Wrap(client).AccountByNativeID(ctx, config.NativeID)
	if errors.Is(err, graphql.ErrNotFound) {
		return diag.Errorf("account identified by profile/role could not be found")
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if cloudAccount.ID != id {
		return diag.Errorf("resource id and profile/role refer to different accounts")
	}

	// Verify that outpost-dependent features have an outpost account available,
	// either in this resource or already onboarded in RSC.
	if outpostBlock := d.Get(keyOutpost).([]any); len(outpostBlock) == 0 {
		outpostDependentKeys := []string{keyCyberRecoveryDataScanning, keyDataScanning, keyDSPM}
		var addingOutpostDependentFeature bool
		for _, key := range outpostDependentKeys {
			if awsCFTFeatureBlockAdded(key, d) {
				addingOutpostDependentFeature = true
				break
			}
		}
		if addingOutpostDependentFeature {
			outposts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "",
				[]core.Status{core.StatusConnected, core.StatusMissingPermissions})
			if err != nil {
				return diag.Errorf("failed to check outpost account status: %s", err)
			}
			if len(outposts) == 0 {
				return diag.Errorf("cyber_recovery_data_scanning, data_scanning, and dspm features require an outpost account")
			}
		}
	}

	// When the outpost feature is added, it needs to be added first since
	// other features depends on it.
	if awsCFTFeatureBlockAdded(keyOutpost, d) {
		if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyOutpost, d); err != nil {
			return diag.FromErr(err)
		}
	}

	// Update regular features.
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCloudNativeArchival, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCloudNativeProtection, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCloudNativeDynamoDBProtection, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCloudNativeS3Protection, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCyberRecoveryDataScanning, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyDataScanning, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyDSPM, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyExocompute, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyKubernetesProtection, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyRDSProtection, d); err != nil {
		return diag.FromErr(err)
	}
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyServersAndApps, d); err != nil {
		return diag.FromErr(err)
	}

	// The Cloud Discovery feature needs to be updated after the protection
	// features.
	if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyCloudDiscovery, d); err != nil {
		return diag.FromErr(err)
	}

	// When the outpost feature is removed, it needs to be removed last since
	// other features depends on it.
	if awsCFTFeatureBlockRemoved(keyOutpost, d) {
		if err := awsUpdateCFTFeatureBlock(ctx, client, account, id, keyOutpost, d); err != nil {
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

	awsReadAccount(ctx, d, m)
	return nil
}

func awsDeleteAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	profile := d.Get(keyProfile).(string)
	roleARN := d.Get(keyAssumeRole).(string)
	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)

	var account aws.AccountFunc
	switch {
	case profile != "" && roleARN == "":
		account = aws.Profile(profile)
	case profile != "":
		account = aws.ProfileWithRole(profile, roleARN)
	default:
		account = aws.DefaultWithRole(roleARN)
	}

	// Check that the resource ID and account profile refer to the same account.
	config, err := account(ctx)
	if err != nil {
		return diag.Errorf("failed to lookup native account id: %s", err)
	}
	cloudAccount, err := aws.Wrap(client).AccountByNativeID(ctx, config.NativeID)
	if errors.Is(err, graphql.ErrNotFound) {
		return diag.Errorf("account identified by profile/role could not be found")
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if cloudAccount.ID != id {
		return diag.Errorf("resource id and profile/role refer to different accounts")
	}

	var features []core.Feature
	for blockKey, blockFeatures := range awsCFTFeatureBlockMap {
		if _, ok := d.GetOk(blockKey); ok {
			features = append(features, blockFeatures...)
		}
	}
	features = slices.DeleteFunc(features, func(f core.Feature) bool {
		return f.Equal(core.FeatureCloudDiscovery) || f.Equal(core.FeatureOutpost)
	})
	if len(features) != 0 {
		err = aws.Wrap(client).RemoveAccountWithCFT(ctx, account, features, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Cloud discovery should always be deleted after all protection features
	// have been deleted.
	if _, ok := d.GetOk(keyCloudDiscovery); ok {
		err = aws.Wrap(client).RemoveAccountWithCFT(ctx, account, []core.Feature{core.FeatureCloudDiscovery}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Outpost should always be deleted last due to its dependency on mapped
	// cloud accounts
	if block, ok := d.GetOk(keyOutpost); ok {
		// Check that the outpost account isn't used by other accounts
		// before removing.
		if err := awsCheckOutpostMappedAccounts(ctx, client); err != nil {
			return diag.FromErr(err)
		}

		// Create an AccountFunc from the information in the outpost feature
		// block.
		outpostAccount := account
		feature, err := awsFromCFTFeatureBlock(keyOutpost, block)
		if err != nil {
			return diag.FromErr(err)
		}
		if outpostID := feature.outpostID; outpostID != "" {
			if outpostProfile := feature.outpostProfile; outpostProfile != "" {
				outpostAccount = aws.Profile(outpostProfile)
			} else {
				outpostAccount = aws.ProfileWithAccountID(profile, outpostID)
			}
		}
		err = aws.Wrap(client).RemoveAccountWithCFT(ctx, outpostAccount, []core.Feature{core.FeatureOutpost}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

func awsCustomizeDiffAccount(ctx context.Context, diff *schema.ResourceDiff, m any) error {
	tflog.Trace(ctx, "awsCustomizeDiffAccount")

	var nonOutpostFeatureCount int
	for blockKey := range awsCFTFeatureBlockMap {
		if blockKey == keyOutpost {
			continue
		}
		if block := diff.Get(blockKey).([]any); len(block) > 0 {
			nonOutpostFeatureCount++
		}
	}

	if outpostBlock := diff.Get(keyOutpost).([]any); len(outpostBlock) > 0 {
		// The outpost feature is specified. Check if the outpost account ID is
		// specified too, if so we need at least one other feature.
		//
		// This is needed because if the outpost_account_id is specified, the
		// outpost account could be a separate account, and we don't know this
		// until we onboard the account, since we only have a profile. If it is
		// a separate account, and it's the only feature, we don't get an RSC
		// cloud account ID for the main account.
		block := outpostBlock[0].(map[string]any)
		if outpostAccountID := block[keyOutpostAccountID].(string); outpostAccountID != "" {
			if nonOutpostFeatureCount == 0 {
				return errors.New("when outpost_account_id is specified for the outpost feature, at least one " +
					"other feature must be enabled")
			}
		}
	}

	return nil
}

// awsCFTFeatureResource returns a schema resource for a CFT feature block.
func awsCFTFeatureResource(permissionGroups []core.PermissionGroup) *schema.Resource {
	// The following permission groups cannot be used when onboarding an AWS
	// account. They have been accepted in the past so we still silently allow
	// them.
	pgs := []core.PermissionGroup{
		core.PermissionGroupExportAndRestore,
		core.PermissionGroupFileLevelRecovery,
		core.PermissionGroupSnapshotPrivateAccess,
		core.PermissionGroupPrivateEndpoints,
	}

	var groups, names []string
	for _, group := range permissionGroups {
		groups = append(groups, string(group))
		if !slices.Contains(pgs, group) {
			names = append(names, fmt.Sprintf("`%s`", group))
		}
	}

	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyPermissionGroups: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(groups, false),
				},
				Required: true,
				Description: fmt.Sprintf("Permission groups to assign to the feature. Possible values are "+
					"%s.", strings.Join(names, ", ")),
			},
			keyRegions: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:    1,
				Required:    true,
				Description: "Regions the feature will be enabled in.",
			},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the feature.",
			},
			keyStackARN: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CloudFormation stack ARN.",
			},
		},
	}
}

// awsCFTFeatureBlockMap maps feature block keys to the corresponding RSC
// features. This map must be updated when support for new features are added
// to the resource.
// Note, if a feature block has more than one RSC feature, they must use the
// same set of permission groups.
var awsCFTFeatureBlockMap = map[string][]core.Feature{
	keyCloudDiscovery: {
		core.FeatureCloudDiscovery,
	},
	keyCloudNativeArchival: {
		core.FeatureCloudNativeArchival,
	},
	keyCloudNativeProtection: {
		core.FeatureCloudNativeProtection,
	},
	keyCloudNativeDynamoDBProtection: {
		core.FeatureCloudNativeDynamoDBProtection,
	},
	keyCloudNativeS3Protection: {
		core.FeatureCloudNativeS3Protection,
	},
	keyCyberRecoveryDataScanning: {
		core.FeatureCyberRecoveryDataClassificationData,
		core.FeatureCyberRecoveryDataClassificationMetadata,
	},
	keyDataScanning: {
		core.FeatureLaminarInternal,
		core.FeatureLaminarCrossAccount,
	},
	keyDSPM: {
		core.FeatureDSPMData,
		core.FeatureDSPMMetadata,
	},
	keyExocompute: {
		core.FeatureExocompute,
	},
	keyKubernetesProtection: {
		core.FeatureKubernetesProtection,
	},
	keyOutpost: {
		core.FeatureOutpost,
	},
	keyRDSProtection: {
		core.FeatureRDSProtection,
	},
	keyServersAndApps: {
		core.FeatureServerAndApps,
	},
}

// awsCFTFeatureBlock represents a feature block in the Terraform configuration.
type awsCFTFeatureBlock struct {
	features       []core.Feature
	regions        []gqlaws.Region
	outpostID      string
	outpostProfile string
}

func (f *awsCFTFeatureBlock) mergeFeature(rhs awsCFTFeatureBlock) {
	f.features = append(f.features, rhs.features...)

	if rhs.outpostID != "" {
		f.outpostID = rhs.outpostID
		f.outpostProfile = rhs.outpostProfile
	}
}

// awsFromCFTFeatureBlock returns the awsCFTFeatureBlock for the specified
// configuration feature key. If the feature block isn't specified in the
// configuration, nil is returned.
func awsFromCFTFeatureBlock(featureKey string, featureBlock any) (*awsCFTFeatureBlock, error) {
	if featureBlock == nil {
		return nil, nil
	}
	if block, ok := featureBlock.([]any); !ok || len(block) == 0 {
		return nil, nil
	}
	block := featureBlock.([]any)[0].(map[string]any)

	features := slices.Clone(awsCFTFeatureBlockMap[featureKey])
	for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
		for i, feature := range features {
			features[i] = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}
	}

	var regions []gqlaws.Region
	if regionSet, ok := block[keyRegions]; ok {
		for _, regionName := range regionSet.(*schema.Set).List() {
			region := gqlaws.RegionFromName(regionName.(string))
			if region == gqlaws.RegionUnknown {
				return nil, fmt.Errorf("unknown region %q", regionName)
			}
			regions = append(regions, region)
		}
	}

	var outpostID string
	if id, ok := block[keyOutpostAccountID]; ok {
		outpostID = id.(string)
	}
	var outpostProfile string
	if profile, ok := block[keyOutpostAccountProfile]; ok {
		outpostProfile = profile.(string)
	}

	return &awsCFTFeatureBlock{
		features:       features,
		regions:        regions,
		outpostID:      outpostID,
		outpostProfile: outpostProfile,
	}, nil
}

// awsToCFTFeatureBlock returns the feature block for the specified AWS cloud
// account and feature key. If the feature is not onboarded, nil is returned.
func awsToCFTFeatureBlock(account aws.CloudAccount, featureKey string) []any {
	features := awsCFTFeatureBlockMap[featureKey]
	if len(features) == 0 {
		return nil
	}

	// Check that all RSC features for a feature block is onboarded. If not we
	// return nil, which cause a diff to be generated so that the feature can be
	// re-onboarded.
	// Note, if a feature block has more than one RSC feature, they must use the
	// same set of permission groups.
	var feature aws.Feature
	for _, f := range features {
		var ok bool
		feature, ok = account.Feature(f)
		if !ok {
			return nil
		}
	}

	groups := schema.Set{F: schema.HashString}
	for _, group := range feature.PermissionGroups {
		groups.Add(string(group))
	}

	regions := schema.Set{F: schema.HashString}
	for _, region := range feature.Regions {
		regions.Add(region)
	}

	return []any{
		map[string]any{
			keyPermissionGroups: &groups,
			keyRegions:          &regions,
			keyStatus:           core.FormatStatus(feature.Status),
			keyStackARN:         feature.StackArn,
		},
	}
}

// awsToCFTOutpostBlock returns the Outpost feature block for the specified AWS
// cloud account. If the feature is not onboarded, nil is returned.
func awsToCFTOutpostBlock(account aws.CloudAccount, outpostProfile string, outpostFields bool) []any {
	feature, ok := account.Feature(core.FeatureOutpost)
	if !ok {
		return nil
	}

	groups := schema.Set{F: schema.HashString}
	for _, group := range feature.PermissionGroups {
		groups.Add(string(group))
	}

	block := map[string]any{
		keyPermissionGroups: &groups,
		keyStatus:           core.FormatStatus(feature.Status),
		keyStackARN:         feature.StackArn,
	}
	if outpostFields {
		block[keyOutpostAccountID] = account.NativeID
		block[keyOutpostAccountProfile] = outpostProfile
	}

	return []any{block}
}

// awsSquashCFTFeatureBlocks squashes the feature blocks so that features with
// the same regions are merged. This reduces the number of CloudFormation stack
// updates needed.
func awsSquashCFTFeatureBlocks(cftFeatures []awsCFTFeatureBlock) []awsCFTFeatureBlock {
	featureSet := make(map[string]*awsCFTFeatureBlock)

	for _, feature := range cftFeatures {
		key := awsRegionsToKey(feature.regions)
		if f, ok := featureSet[key]; ok {
			f.mergeFeature(feature)
		} else {
			featureSet[key] = &feature
		}
	}

	features := make([]awsCFTFeatureBlock, 0, len(featureSet))
	for _, feature := range featureSet {
		features = append(features, *feature)
	}

	// Sort the features on the number of regions, this cause the outpost
	// feature to be first in the slice.
	slices.SortFunc(features, func(i, j awsCFTFeatureBlock) int {
		return cmp.Compare(len(i.regions), len(j.regions))
	})

	return features
}

// awsRegionsToKey returns a key for the set of regions.
func awsRegionsToKey(regions []gqlaws.Region) string {
	slices.Sort(regions)

	var str strings.Builder
	for _, region := range regions {
		str.WriteString(strconv.Itoa(int(region)))
	}

	return str.String()
}

// awsUpdateCFTFeatureBlock updates the RSC feature according to changes made
// to the feature block identified by the feature key.
func awsUpdateCFTFeatureBlock(ctx context.Context, client *polaris.Client, account aws.AccountFunc, accountID uuid.UUID, featureKey string, d *schema.ResourceData) error {
	tflog.Trace(ctx, "awsUpdateCFTFeatureBlock")

	if !d.HasChange(featureKey) {
		return nil
	}

	oldBlock, newBlock := d.GetChange(featureKey)
	oldFeatureBlock, err := awsFromCFTFeatureBlock(featureKey, oldBlock)
	if err != nil {
		return err
	}
	newFeatureBlock, err := awsFromCFTFeatureBlock(featureKey, newBlock)
	if err != nil {
		return err
	}

	var opts []aws.OptionFunc
	if newFeatureBlock != nil {
		for _, region := range newFeatureBlock.regions {
			opts = append(opts, aws.Region(region.Name()))
		}
	}

	// Add feature.
	if oldFeatureBlock == nil && newFeatureBlock != nil {
		if outpostID := newFeatureBlock.outpostID; outpostID != "" {
			if outpostProfile := newFeatureBlock.outpostProfile; outpostProfile != "" {
				opts = append(opts, aws.OutpostAccountWithProfile(outpostID, outpostProfile))
			} else {
				opts = append(opts, aws.OutpostAccount(outpostID))
			}
		}

		_, err = aws.Wrap(client).AddAccountWithCFT(ctx, account, newFeatureBlock.features, opts...)
		return err
	}

	// Remove feature.
	if oldFeatureBlock != nil && newFeatureBlock == nil {
		if featureKey == keyOutpost {
			// Check that the outpost account isn't used by other accounts
			// before removing.
			if err := awsCheckOutpostMappedAccounts(ctx, client); err != nil {
				return err
			}

			// Create an AccountFunc from the information in the outpost feature
			// block.
			if outpostID := oldFeatureBlock.outpostID; outpostID != "" {
				if outpostProfile := oldFeatureBlock.outpostProfile; outpostProfile != "" {
					account = aws.Profile(outpostProfile)
				} else {
					account = aws.ProfileWithAccountID(d.Get(keyProfile).(string), outpostID)
				}
			}
		}

		deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
		return aws.Wrap(client).RemoveAccountWithCFT(ctx, account, awsCFTFeatureBlockMap[featureKey], deleteSnapshots)
	}

	// Update feature.
	if oldFeatureBlock != nil {
		for _, feature := range newFeatureBlock.features {
			if err := aws.Wrap(client).UpdateAccount(ctx, accountID, feature, opts...); err != nil {
				return err
			}
		}
	}

	return nil
}

// awsCFTFeatureBlockAdded returns true if the feature block has been added.
func awsCFTFeatureBlockAdded(featureKey string, d *schema.ResourceData) bool {
	if !d.HasChange(featureKey) {
		return false
	}
	oldBlock, newBlock := d.GetChange(featureKey)
	return len(oldBlock.([]any)) == 0 && len(newBlock.([]any)) > 0
}

// awsCFTFeatureBlockRemoved returns true if the feature block has been removed.
func awsCFTFeatureBlockRemoved(featureKey string, d *schema.ResourceData) bool {
	if !d.HasChange(featureKey) {
		return false
	}
	oldBlock, newBlock := d.GetChange(featureKey)
	return len(oldBlock.([]any)) > 0 && len(newBlock.([]any)) == 0
}

// awsCheckOutpostMappedAccounts checks that the outpost account isn't used by
// other accounts.
func awsCheckOutpostMappedAccounts(ctx context.Context, client *polaris.Client) error {
	accounts, err := aws.Wrap(client).AccountsByFeatureStatus(ctx, core.FeatureOutpost, "",
		[]core.Status{core.StatusConnected, core.StatusMissingPermissions})
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

	return nil
}
