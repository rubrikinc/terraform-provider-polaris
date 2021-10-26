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
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	graphql_aws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// validateAwsRegion verifies that the name is a valid AWS region name.
func validateAwsRegion(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := graphql_aws.ParseRegion(m.(string))
	return diag.FromErr(err)
}

// validatePermissions verifies that the permissions value is valid.
func validatePermissions(m interface{}, p cty.Path) diag.Diagnostics {
	if m.(string) != "update" {
		return diag.Errorf("invalid permissions value")
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
			"cloud_native_protection": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateAwsRegion,
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
					},
				},
				MaxItems:    1,
				Required:    true,
				Description: "Enable the Cloud Native Protection feature for the GCP project.",
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
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateAwsRegion,
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
				Required:         true,
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

	client := m.(*polaris.Client)

	profile := d.Get("profile").(string)

	var opts []aws.OptionFunc
	if name, ok := d.GetOk("name"); ok {
		opts = append(opts, aws.Name(name.(string)))
	}

	// Check if the account already exist in Polaris.
	account, err := client.AWS().Account(ctx, aws.ID(aws.Profile(profile)), core.FeatureAll)
	if err == nil {
		return diag.Errorf("account %q already added to polaris", account.NativeID)
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return diag.FromErr(err)
	}

	// Polaris Cloud Account id. Returned when the account is added for the
	// cloud native protection feature.
	var id uuid.UUID

	cnpBlock, ok := d.GetOk("cloud_native_protection")
	if ok {
		block := cnpBlock.([]interface{})[0].(map[string]interface{})

		var cnpOpts []aws.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
			cnpOpts = append(cnpOpts, aws.Region(region.(string)))
		}

		cnpOpts = append(cnpOpts, opts...)
		id, err = client.AWS().AddAccount(ctx, aws.Profile(profile), core.FeatureCloudNativeProtection, cnpOpts...)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	exoBlock, ok := d.GetOk("exocompute")
	if ok {
		block := exoBlock.([]interface{})[0].(map[string]interface{})

		var exoOpts []aws.OptionFunc
		for _, region := range block["regions"].(*schema.Set).List() {
			exoOpts = append(exoOpts, aws.Region(region.(string)))
		}

		exoOpts = append(exoOpts, opts...)
		_, err := client.AWS().AddAccount(ctx, aws.Profile(profile), core.FeatureExocompute, exoOpts...)
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

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the Polaris cloud account using the cloud account id.
	account, err := client.AWS().Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if err != nil {
		return diag.FromErr(err)
	}

	cnpFeature, ok := account.Feature(core.FeatureCloudNativeProtection)
	if ok {
		regions := schema.Set{F: schema.HashString}
		for _, region := range cnpFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(cnpFeature.Status)
		err := d.Set("cloud_native_protection", []interface{}{
			map[string]interface{}{
				"regions": &regions,
				"status":  &status,
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
		regions := schema.Set{F: schema.HashString}
		for _, region := range exoFeature.Regions {
			regions.Add(region)
		}

		status := core.FormatStatus(exoFeature.Status)
		err := d.Set("exocompute", []interface{}{
			map[string]interface{}{
				"regions": &regions,
				"status":  &status,
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

	client := m.(*polaris.Client)
	profile := d.Get("profile").(string)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Make sure that the resource id and account profile refers to the same
	// account.
	account, err := client.AWS().Account(ctx, aws.ID(aws.Profile(profile)), core.FeatureAll)
	if err != nil {
		return diag.FromErr(err)
	}
	if account.ID != id {
		return diag.Errorf("id and profile refer to different accounts")
	}

	if d.HasChange("cloud_native_protection") {
		cnpBlock, ok := d.GetOk("cloud_native_protection")
		if ok {
			block := cnpBlock.([]interface{})[0].(map[string]interface{})

			var opts []aws.OptionFunc
			for _, region := range block["regions"].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			if err := client.AWS().UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureCloudNativeProtection, opts...); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if _, ok := d.GetOk("exocompute"); ok {
				return diag.Errorf("cloud native protection is required by exocompute")
			}

			snapshots := d.Get("delete_snapshots_on_destroy").(bool)
			if err := client.AWS().RemoveAccount(ctx, aws.Profile(profile), core.FeatureCloudNativeProtection, snapshots); err != nil {
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
			var opts []aws.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			_, err = client.AWS().AddAccount(ctx, aws.Profile(profile), core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		case len(newExoList) == 0:
			err := client.AWS().RemoveAccount(ctx, aws.Profile(profile), core.FeatureExocompute, false)
			if err != nil {
				return diag.FromErr(err)
			}
		default:
			var opts []aws.OptionFunc
			for _, region := range newExoList[0].(map[string]interface{})["regions"].(*schema.Set).List() {
				opts = append(opts, aws.Region(region.(string)))
			}

			err = client.AWS().UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureExocompute, opts...)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("permissions") {
		oldPerms, newPerms := d.GetChange("permissions")

		if oldPerms == "update-required" && newPerms == "update" {
			var features []core.Feature
			for _, feature := range account.Features {
				if feature.Status != core.StatusMissingPermissions {
					continue
				}
				features = append(features, feature.Name)
			}

			err := client.AWS().UpdatePermissions(ctx, aws.Profile(profile), features)
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

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the old resource arguments.
	oldProfile, _ := d.GetChange("profile")
	profile := oldProfile.(string)

	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	// Make sure that the resource id and account profile refers to the same
	// account.
	account, err := client.AWS().Account(ctx, aws.ID(aws.Profile(profile)), core.FeatureAll)
	if err != nil {
		return diag.FromErr(err)
	}
	if account.ID != id {
		return diag.Errorf("id and profile refer to different accounts")
	}

	// Removing Cloud Native Protection also removes Exocompute.
	err = client.AWS().RemoveAccount(ctx, aws.Profile(profile), core.FeatureCloudNativeProtection, deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
