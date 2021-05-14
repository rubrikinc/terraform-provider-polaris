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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
)

var awsRegions = []string{
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-south-1",
	"ca-central-1",
	"cn-northwest-1",
	"cn-north-1",
	"eu-central-1",
	"eu-north-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"sa-east-1",
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
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
				Description:      "Account name in Polaris.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "What should happen to snapshots when the account is removed from Polaris.",
			},
			"profile": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "AWS shared credentials file.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(awsRegions, true),
				},
				Required:    true,
				Description: "Polaris will auto-discover instances to be protected from the specified regions.",
			},
		},
	}
}

// awsCreateAccount run the Create operation for the AWS account resource. This
// adds the AWS account to the Polaris platform.
func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAccount")

	client := m.(*polaris.Client)
	profile := d.Get("profile").(string)

	// Check if the account already exist in Polaris.
	account, err := client.AwsAccount(ctx, polaris.FromAwsProfile(profile))
	switch {
	case errors.Is(err, polaris.ErrNotFound):
	case err == nil:
		return diag.Errorf("account %q already added to polaris", profile)
	case err != nil:
		return diag.FromErr(err)
	}

	var withOpts []polaris.AddOption
	if name, ok := d.GetOk("name"); ok {
		withOpts = append(withOpts, polaris.WithName(name.(string)))
	}
	for _, region := range d.Get("regions").(*schema.Set).List() {
		withOpts = append(withOpts, polaris.WithRegion(region.(string)))
	}

	// Add account to Polaris.
	if err := client.AwsAccountAdd(ctx, polaris.FromAwsProfile(profile), withOpts...); err != nil {
		return diag.FromErr(err)
	}

	// Lookup the ID and AWS account ID of the newly added account. Note that
	// the resource ID is created from both.
	account, err = client.AwsAccount(ctx, polaris.FromAwsProfile(profile))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toResourceID(account.ID, account.NativeID))

	// Populate the local Terraform state.
	awsReadAccount(ctx, d, m)

	return nil
}

// awsReadAccount run the Read operation for the AWS account resource. This
// reads the state of the AWS account in Polaris.
func awsReadAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadAccount")

	client := m.(*polaris.Client)

	// Get the AWS account ID from the local resource ID.
	_, awsAccountID, err := fromResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the Polaris cloud account using the AWS account ID.
	account, err := client.AwsAccount(ctx, polaris.WithAwsID(awsAccountID))
	if err != nil {
		return diag.FromErr(err)
	}

	// Get AWS regions for the CNP feature.
	regions := schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		if feature.Feature != "CLOUD_NATIVE_PROTECTION" {
			continue
		}

		for _, region := range feature.AwsRegions {
			regions.Add(region)
		}
		if err := d.Set("regions", &regions); err != nil {
			return diag.FromErr(err)
		}
		break
	}

	return nil
}

// awsUpdateAccount run the Update operation for the AWS account resource. This
// updates the state of the AWS account in Polaris.
func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAccount")

	client := m.(*polaris.Client)

	// Get the AWS account ID from the local resource ID.
	_, awsAccountID, err := fromResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Update the regions argument when changed.
	if d.HasChange("regions") {
		var regions []string
		for _, region := range d.Get("regions").(*schema.Set).List() {
			regions = append(regions, region.(string))
		}

		client.AwsAccountSetRegions(ctx, polaris.WithAwsID(awsAccountID), regions...)
	}

	return nil
}

// awsDeleteAccount run the Delete operation for the AWS account resource. This
// removes the AWS account from Polaris.
func awsDeleteAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteAccount")

	client := m.(*polaris.Client)

	// Get the old resource arguments.
	oldProfile, _ := d.GetChange("profile")
	profile := oldProfile.(string)

	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	// Remove the account.
	if err := client.AwsAccountRemove(ctx, polaris.FromAwsProfile(profile), deleteSnapshots); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
