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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

func resourceAwsCnpAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccount,
		ReadContext:   awsReadCnpAccount,
		UpdateContext: awsUpdateCnpAccount,
		DeleteContext: awsDeleteCnpAccount,

		Schema: map[string]*schema.Schema{
			"cloud": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Cloud type.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"external_id": { // needed to force full recreation of account if external id is changed.
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "External id.",
			},
			"features": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:    1,
				Required:    true,
				Description: "RSC features.",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Account name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"native_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "AWS account id.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"regions": {
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
	log.Print("[TRACE] awsCreateCnpAccount")

	client := m.(*polaris.Client)

	// Get attributes.
	cloud := d.Get("cloud").(string)
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature(feature.(string)))
	}
	name := d.Get("name").(string)
	nativeID := d.Get("native_id").(string)
	var regions []string
	for _, region := range d.Get("regions").(*schema.Set).List() {
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
	log.Print("[TRACE] awsReadCnpAccount")

	client := m.(*polaris.Client)

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
	features := &schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		features.Add(string(feature.Name))
	}
	if err := d.Set("features", features); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", account.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("native_id", account.NativeID); err != nil {
		return diag.FromErr(err)
	}
	regions := &schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		for _, region := range feature.Regions {
			regions.Add(region)
		}
	}
	if err := d.Set("regions", regions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCnpAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateCnpAccount")

	client := m.(*polaris.Client)

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cloud := d.Get("cloud").(string)
	deleteSnapshots := d.Get("delete_snapshots_on_destroy").(bool)
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature(feature.(string)))
	}
	name := d.Get("name").(string)
	nativeID := d.Get("native_id").(string)
	var regions []string
	for _, region := range d.Get("regions").(*schema.Set).List() {
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

	if d.HasChange("name") {
		if err := aws.Wrap(client).UpdateAccount(ctx, aws.CloudAccountID(id), core.FeatureAll, aws.Name(name)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("features") {
		oldAttr, newAttr := d.GetChange("features")
		var oldFeatures []core.Feature
		for _, feature := range oldAttr.(*schema.Set).List() {
			oldFeatures = append(oldFeatures, core.Feature(feature.(string)))
		}
		var newFeatures []core.Feature
		for _, feature := range newAttr.(*schema.Set).List() {
			newFeatures = append(newFeatures, core.Feature(feature.(string)))
		}
		addFeatures, removeFeatures := diffFeatures(newFeatures, oldFeatures)

		account := aws.AccountWithName(cloud, nativeID, name)
		if len(addFeatures) > 0 {
			// When adding new features the list should include all features.
			if _, err := aws.Wrap(client).AddAccount(ctx, account, newFeatures, aws.Regions(regions...)); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(removeFeatures) > 0 {
			// When removing features only the features to be removed should be
			// passed in.
			if err := aws.Wrap(client).RemoveAccount(ctx, account, removeFeatures, deleteSnapshots); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("regions") {
		var regions []string
		for _, region := range d.Get("regions").(*schema.Set).List() {
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
	log.Print("[TRACE] awsDeleteCnpAccount")

	client := m.(*polaris.Client)

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	deleteSnapshots := d.Get("delete_snapshots_on_destroy").(bool)

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
		features = append(features, feature.Name)
	}

	// Request account removal.
	if err := aws.Wrap(client).RemoveAccount(ctx, aws.AccountWithName(account.Cloud, account.NativeID, account.Name), features, deleteSnapshots); err != nil {
		return diag.FromErr(err)
	}

	// Reset ID.
	d.SetId("")

	return nil
}

func diffFeatures(newFeatures []core.Feature, oldFeatures []core.Feature) ([]core.Feature, []core.Feature) {
	newSet := make(map[core.Feature]struct{})
	for _, feature := range newFeatures {
		newSet[feature] = struct{}{}
	}
	oldSet := make(map[core.Feature]struct{})
	for _, feature := range oldFeatures {
		oldSet[feature] = struct{}{}
	}

	for feature := range oldSet {
		if _, ok := newSet[feature]; ok {
			delete(newSet, feature)
			delete(oldSet, feature)
		}
	}

	addFeatures := make([]core.Feature, 0, len(newSet))
	for feature := range newSet {
		addFeatures = append(addFeatures, feature)
	}
	removeFeatures := make([]core.Feature, 0, len(oldSet))
	for feature := range oldSet {
		removeFeatures = append(removeFeatures, feature)
	}

	return addFeatures, removeFeatures
}
