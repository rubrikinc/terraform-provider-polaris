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

// validateAwsRegion validates the region name.
func validateAwsRegion(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := graphql_aws.ParseRegion(m.(string))
	return diag.FromErr(err)
}

// resourceAwsAccount defines the schema for the AWS account resource.
func resourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAccount,
		ReadContext:   awsReadAccount,
		UpdateContext: awsUpdateAccount,
		DeleteContext: awsDeleteAccount,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Account name in Polaris. If not given the name is taken from AWS Organizations or, if the required permissions are missing, is derived from the AWS account ID and the named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
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
							Description: "Regions to enable the exocompute feature in.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the exocompute feature for the account.",
			},
			"profile": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "AWS named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateAwsRegion,
				},
				Required:    true,
				Description: "Regions that Polaris will monitor for instances to automatically protect.",
			},
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAwsAccountV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAwsAccountStateUpgradeV0,
			Version: 0,
		}},
	}
}

// awsCreateAccount run the Create operation for the AWS account resource. This
// adds the AWS account to the Polaris platform.
func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAccount")

	client := m.(*polaris.Client)

	// Profile parameter, required.
	profile := d.Get("profile").(string)

	// Name parameter, optional.
	var opts []aws.OptionFunc
	if name, ok := d.GetOk("name"); ok {
		opts = append(opts, aws.Name(name.(string)))
	}

	// Regions parameter, required.
	regions := d.Get("regions").(*schema.Set)
	for _, region := range regions.List() {
		opts = append(opts, aws.Region(region.(string)))
	}

	// Exocompute parameter, optional. Verify the regions specified and
	// guarantee that it's nil if it's not specified.
	exocompute, ok := d.GetOk("exocompute")
	if ok {
		block := exocompute.([]interface{})[0].(map[string]interface{})
		for _, region := range block["regions"].(*schema.Set).List() {
			if !regions.Contains(region) {
				return diag.Errorf("exocompute can only have a subset of the account regions")
			}
		}
	} else {
		exocompute = nil
	}

	// Check if the account already exist in Polaris.
	account, err := client.AWS().Account(ctx, aws.ID(aws.Profile(profile)), core.CloudNativeProtection)
	switch {
	case errors.Is(err, graphql.ErrNotFound):
	case err == nil:
		return diag.Errorf("account %q already added to polaris", account.NativeID)
	case err != nil:
		return diag.FromErr(err)
	}

	// Add account to Polaris. Implicitly enables the Cloud Native Protection feature.
	id, err := client.AWS().AddAccount(ctx, aws.Profile(profile), opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	// Enable the Exocompute feature if specified.
	if exocompute != nil {
		block := exocompute.([]interface{})[0].(map[string]interface{})

		var regions []string
		for _, region := range block["regions"].(*schema.Set).List() {
			regions = append(regions, region.(string))
		}

		err := client.AWS().EnableExocompute(ctx, aws.Profile(profile), regions...)
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

	// Lookup the Polaris cloud account using the cloud account ID.
	account, err := client.AWS().Account(ctx, aws.CloudAccountID(id), core.AllFeatures)
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the name parameter.
	if err := d.Set("name", account.Name); err != nil {
		return diag.FromErr(err)
	}

	// Read the regions parmeter.
	cnpFeature, ok := account.Feature(core.CloudNativeProtection)
	if ok {
		regions := schema.Set{F: schema.HashString}
		for _, region := range cnpFeature.Regions {
			regions.Add(region)
		}
		err := d.Set("regions", &regions)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Read the exocompute parameter.
	exoFeature, ok := account.Feature(core.Exocompute)
	if ok {
		if len(exoFeature.Regions) > 0 {
			regions := schema.Set{F: schema.HashString}
			for _, region := range exoFeature.Regions {
				regions.Add(region)
			}
			block := []interface{}{
				map[string]interface{}{
					"regions": &regions,
				},
			}
			if err := d.Set("exocompute", block); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := d.Set("exocompute", nil); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

// awsUpdateAccount run the Update operation for the AWS account resource. This
// updates the state of the AWS account in Polaris.
func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAccount")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	profile := d.Get("profile").(string)

	// Regions needed due to the existing exocompute resources.
	exoConfigs, err := client.AWS().ExocomputeConfigs(ctx, aws.CloudAccountID(id))
	if err != nil {
		return diag.FromErr(err)
	}
	exoConfigRegions := make(map[string]struct{})
	for _, exoConfig := range exoConfigs {
		exoConfigRegions[exoConfig.Region] = struct{}{}
	}

	// Regions specified by the exocompute feature.
	exoRegions := make(map[string]struct{})
	if exocompute, ok := d.GetOk("exocompute"); ok {
		block := exocompute.([]interface{})[0].(map[string]interface{})
		for _, r := range block["regions"].(*schema.Set).List() {
			exoRegions[r.(string)] = struct{}{}
		}
	}
	for region := range exoConfigRegions {
		if _, ok := exoRegions[region]; !ok {
			return diag.Errorf("exocompute feature regions must be a superset of exocompute resource regions")
		}
	}

	// Regions specified by the cloud native protection feature.
	cnpRegions := make(map[string]struct{})
	for _, region := range d.Get("regions").(*schema.Set).List() {
		cnpRegions[region.(string)] = struct{}{}
	}
	for region := range exoRegions {
		if _, ok := cnpRegions[region]; !ok {
			return diag.Errorf("account regions must be a superset of exocompute feature regions")
		}
	}

	if d.HasChange("regions") {
		var regions []string
		for region := range cnpRegions {
			regions = append(regions, region)
		}

		err = client.AWS().UpdateAccount(ctx, aws.CloudAccountID(id), core.CloudNativeProtection,
			aws.Regions(regions...))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("exocompute") && len(exoRegions) > 0 {
		var regions []string
		for region := range exoRegions {
			regions = append(regions, region)
		}

		err := client.AWS().EnableExocompute(ctx, aws.Profile(profile), regions...)
		if errors.Is(err, graphql.ErrAlreadyEnabled) {
			err = client.AWS().UpdateAccount(ctx, aws.CloudAccountID(id), core.Exocompute,
				aws.Regions(regions...))
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

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
	account, err := client.AWS().Account(ctx, aws.ID(aws.Profile(profile)), core.CloudNativeProtection)
	if err != nil {
		return diag.FromErr(err)
	}
	if account.ID != id {
		return diag.Errorf("id and profile refer to different accounts")
	}

	// Remove the account.
	err = client.AWS().RemoveAccount(ctx, aws.Profile(profile), deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
