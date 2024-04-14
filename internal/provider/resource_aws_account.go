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

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// validatePermissions verifies that the permissions value is valid.
func validatePermissions(m interface{}, p cty.Path) diag.Diagnostics {
	if m.(string) != "update" {
		return diag.Errorf("invalid permissions value")
	}

	return nil
}

// validateRoleARN verifies that the role ARN is a valid AWS ARN.
func validateRoleARN(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := arn.Parse(m.(string)); err != nil {
		return diag.Errorf("failed to parse role ARN: %v", err)
	}

	return nil
}

// resourceAwsAccount defines the schema for the AWS account resource.
func resourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAccount,
		ReadContext:   awsReadAccount,
		UpdateContext: awsUpdateAccount,
		DeleteContext: awsDeleteAccount,

		Schema: map[string]*schema.Schema{
			"assume_role": {
				Type:             schema.TypeString,
				Optional:         true,
				AtLeastOneOf:     []string{"profile"},
				Description:      "Role ARN of role to assume.",
				ValidateDiagFunc: validateRoleARN,
			},
			"cloud_native_protection": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"permission_groups": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Permission groups to assign to the cloud native protection feature.",
						},
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions that Polaris will monitor for instances to automatically protect.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
						"stack_arn": {
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
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"exocompute": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"permission_groups": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Permission groups to assign to the exocompute feature.",
						},
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotWhiteSpace,
							},
							MinItems:    1,
							Required:    true,
							Description: "Regions to enable the Exocompute feature in.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Exocompute feature.",
						},
						"stack_arn": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloudformation stack ARN.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the exocompute feature for the account.",
			},
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Account name in Polaris. If not given the name is taken from AWS Organizations or, if the required permissions are missing, is derived from the AWS account ID and the named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"permissions": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "When set to 'update' feature permissions can be updated by applying the configuration.",
				ValidateDiagFunc: validatePermissions,
			},
			"profile": {
				Type:             schema.TypeString,
				Optional:         true,
				AtLeastOneOf:     []string{"assume_role"},
				Description:      "AWS named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
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

// awsCreateAccount run the Create operation for the AWS account resource. This
// adds the AWS account to the Polaris platform.
func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Initialize to empty string if missing from the configuration.
	profile, _ := d.Get("profile").(string)
	roleARN, _ := d.Get("assume_role").(string)

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
	if name, ok := d.GetOk("name"); ok {
		opts = append(opts, aws.Name(name.(string)))
	}

	// Polaris Cloud Account id. Returned when the account is added for the
	// cloud native protection feature.
	var id uuid.UUID

	cnpBlock, ok := d.GetOk("cloud_native_protection")
	if ok {
		block := cnpBlock.([]interface{})[0].(map[string]interface{})

		feature := core.FeatureCloudNativeProtection
		for _, group := range block["permission_groups"].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var cnpOpts []aws.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
			cnpOpts = append(cnpOpts, aws.Region(region.(string)))
		}

		var err error
		cnpOpts = append(cnpOpts, opts...)
		id, err = aws.Wrap(client).AddAccount(ctx, account, []core.Feature{feature}, cnpOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	exoBlock, ok := d.GetOk("exocompute")
	if ok {
		block := exoBlock.([]interface{})[0].(map[string]interface{})

		feature := core.FeatureExocompute
		for _, group := range block["permission_groups"].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		var exoOpts []aws.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
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

// awsReadAccount run the Read operation for the AWS account resource. This
// reads the state of the AWS account in Polaris.
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
				"permission_groups": &groups,
				"regions":           &regions,
				"status":            &status,
				"stack_arn":         &cnpFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("cloud_native_protection", nil); err != nil {
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
				"permission_groups": &groups,
				"regions":           &regions,
				"status":            &status,
				"stack_arn":         &exoFeature.StackArn,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("exocompute", nil); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("name", account.Name); err != nil {
		return diag.FromErr(err)
	}

	// Check if any feature is missing permissions.
	for _, feature := range account.Features {
		if feature.Status != core.StatusMissingPermissions {
			continue
		}

		if err := d.Set("permissions", "update-required"); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// awsUpdateAccount run the Update operation for the AWS account resource. This
// updates the state of the AWS account in Polaris.
func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Initialize to empty string if missing from the configuration.
	profile, _ := d.Get("profile").(string)
	roleARN, _ := d.Get("assume_role").(string)

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

	if d.HasChange("cloud_native_protection") {
		cnpBlock, ok := d.GetOk("cloud_native_protection")
		if ok {
			block := cnpBlock.([]interface{})[0].(map[string]interface{})

			feature := core.FeatureCloudNativeProtection
			for _, group := range block["permission_groups"].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}

			var opts []aws.OptionFunc
			for _, region := range block["regions"].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			if err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), feature, opts...); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if _, ok := d.GetOk("exocompute"); ok {
				return diag.Errorf("cloud native protection is required by exocompute")
			}

			snapshots := d.Get("delete_snapshots_on_destroy").(bool)
			if err := aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureCloudNativeProtection}, snapshots); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("exocompute") {
		oldExoBlock, newExoBlock := d.GetChange("exocompute")
		oldExoList := oldExoBlock.([]interface{})
		newExoList := newExoBlock.([]interface{})

		// Determine whether we are adding, removing or updating the Exocompute
		// feature.
		switch {
		case len(oldExoList) == 0:
			feature := core.FeatureExocompute
			for _, group := range newExoList[0].(map[string]interface{})["permission_groups"].(*schema.Set).List() {
				feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
			}

			var opts []aws.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
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
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			err = aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("permissions") {
		oldPerms, newPerms := d.GetChange("permissions")

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

			if err := d.Set("permissions", "update"); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	awsReadAccount(ctx, d, m)
	return nil
}

// awsDeleteAccount run the Delete operation for the AWS account resource. This
// removes the AWS account from Polaris.
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
	oldProfile, _ := d.GetChange("profile")
	oldRoleARN, _ := d.GetChange("assume_role")
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

	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
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

	if _, ok := d.GetOk("exocompute"); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureExocompute}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk("cloud_native_protection"); ok {
		err = aws.Wrap(client).RemoveAccount(ctx, account, []core.Feature{core.FeatureCloudNativeProtection}, deleteSnapshots)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")

	return nil
}
