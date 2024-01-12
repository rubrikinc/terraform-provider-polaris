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

func resourceAwsCnpAccountTrustPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccountTrustPolicy,
		ReadContext:   awsReadCnpAccountTrustPolicy,
		UpdateContext: awsUpdateCnpAccountTrustPolicy,
		DeleteContext: awsDeleteCnpAccountTrustPolicy,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC account id.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"external_id": {
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
				ForceNew:    true,
				Description: "RSC features.",
			},
			"policy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Trust policy.",
			},
			"role_key": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Role key.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func awsCreateCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	accountID := d.Get("account_id").(string)
	externalID := d.Get("external_id").(string)
	roleKey := d.Get("role_key").(string)
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}

	// Request the trust policy matching the role key.
	policy, err := trustPolicy(ctx, client, accountID, features, roleKey, externalID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	if err := d.Set("policy", policy); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(accountID)

	awsReadCnpAccountTrustPolicy(ctx, d, m)
	return nil
}

func awsReadCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	roleKey := d.Get("role_key").(string)

	// Request the cloud account.
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// Request the trust policy.
	features := make([]core.Feature, 0, len(account.Features))
	for _, feature := range account.Features {
		features = append(features, feature.Feature)
	}
	policy, err := trustPolicy(ctx, client, id.String(), features, roleKey, "")
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	featuresAttr := &schema.Set{F: schema.HashString}
	for _, feature := range features {
		featuresAttr.Add(feature.Name)
	}
	if err := d.Set("features", featuresAttr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("policy", policy); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	roleKey := d.Get("role_key").(string)
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}

	// Request the trust policy matching the role key. Note that the external ID
	// cannot be updated.
	policy, err := trustPolicy(ctx, client, d.Id(), features, roleKey, "")
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	if err := d.Set("policy", policy); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// awsDeleteCnpAccountTrustPolicy destroys the account trust policy. Note that
// there is no need to destroy the trust policy in RSC, we simply remove the
// trust policy from the state.
func awsDeleteCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteCnpAccountTrustPolicy")

	// Reset ID.
	d.SetId("")

	return nil
}

// trustPolicy returns the external ID and the trust policy for the specified
// role key.
func trustPolicy(ctx context.Context, client *polaris.Client, accountID string, features []core.Feature, roleKey, externalID string) (string, error) {
	id, err := uuid.Parse(accountID)
	if err != nil {
		return "", err
	}
	trustPolicies, err := aws.Wrap(client).TrustPolicies(ctx, aws.CloudAccountID(id), features, externalID)
	if err != nil {
		return "", err
	}

	for key, policy := range trustPolicies {
		if key == roleKey {
			return policy, nil
		}
	}

	return "", fmt.Errorf("trust policy for role key %q not found", roleKey)
}
