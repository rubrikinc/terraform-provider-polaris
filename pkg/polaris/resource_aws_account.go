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

package polaris

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
)

var awsRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
}

// cleanRegionName makes sure that the region is all lower case letters and
// uses hyphens to separate the parts of the region name.
func cleanRegionName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

// fromAccountResourceID converts an account resource ID to a Polaris account
// ID and a native cloud account ID.
func fromAccountResourceID(accountResourceID string) (accountID string, nativeID string, err error) {
	parts := strings.Split(accountResourceID, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("polaris: invalid account resource id: %s", accountResourceID)
	}

	return parts[0], parts[1], nil
}

// toAccountResourceID converts a Polaris account ID and a native cloud account
// ID to a account resource ID.
func toAccountResourceID(accountID, nativeID string) string {
	return fmt.Sprintf("%s:%s", accountID, nativeID)
}

// resourceAwsAccount -
func resourceAwsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAccount,
		ReadContext:   awsReadAccount,
		UpdateContext: awsUpdateAccount,
		DeleteContext: awsDeleteAccount,

		Schema: map[string]*schema.Schema{
			"profile": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "AWS shared credentials file profile.",
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(awsRegions, true),
				},
				Required:    true,
				Description: "AWS regions to use with Polaris.",
			},
		},
	}
}

func awsCreateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAccount")

	polClient := m.(*polaris.Client)

	// Get resource parameters.
	profile := d.Get("profile").(string)
	var regions []string
	for _, region := range d.Get("regions").(*schema.Set).List() {
		regions = append(regions, region.(string))
	}

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return diag.FromErr(err)
	}

	// Check if the account already exist in Polaris.
	account, err := polClient.AwsAccountFromConfig(ctx, awsConfig)
	if err == nil || err != polaris.ErrAccountNotFound {
		return diag.FromErr(err)
	}

	// Add the account.
	if err := polClient.AwsAccountAdd(ctx, awsConfig, regions); err != nil {
		return diag.FromErr(err)
	}

	// Lookup the newly added account.
	account, err = polClient.AwsAccountFromConfig(ctx, awsConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toAccountResourceID(account.ID, account.NativeID))

	awsReadAccount(ctx, d, m)

	return nil
}

// awsReadAccount -
func awsReadAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadAccount")

	polClient := m.(*polaris.Client)

	_, awsAccountID, err := fromAccountResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := polClient.AwsAccountFromID(ctx, awsAccountID)
	if err != nil {
		return diag.FromErr(err)
	}

	regions := schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		if feature.Feature != "CLOUD_NATIVE_PROTECTION" {
			continue
		}

		for _, region := range feature.AwsRegions {
			regions.Add(cleanRegionName(region))
		}
		if err := d.Set("regions", &regions); err != nil {
			return diag.FromErr(err)
		}
		break
	}

	return nil
}

// awsUpdateAccount
func awsUpdateAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAccount")

	polClient := m.(*polaris.Client)

	_, awsAccountID, err := fromAccountResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("regions") {
		var regions []string
		for _, region := range d.Get("regions").(*schema.Set).List() {
			regions = append(regions, region.(string))
		}

		polClient.AwsAccountSetRegions(ctx, awsAccountID, regions)
	}

	return nil
}

// awsDeleteAccount -
func awsDeleteAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteAccount")

	polClient := m.(*polaris.Client)

	_, awsAccountID, err := fromAccountResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	oldProfile, _ := d.GetChange("profile")
	profile := oldProfile.(string)

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := polClient.AwsAccountRemove(ctx, awsConfig, awsAccountID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
