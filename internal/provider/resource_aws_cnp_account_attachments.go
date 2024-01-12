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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

var instanceProfileResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"key": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Instance profile key.",
			ValidateFunc: validation.StringIsNotWhiteSpace,
		},
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "AWS instance profile name.",
			ValidateFunc: validation.StringIsNotWhiteSpace,
		},
	},
}

var roleResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"key": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Role key.",
			ValidateFunc: validation.StringIsNotWhiteSpace,
		},
		"arn": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "AWS role ARN.",
			ValidateFunc: validation.StringIsNotWhiteSpace,
		},
	},
}

func resourceAwsCnpAccountAttachments() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccountAttachments,
		ReadContext:   awsReadCnpAccountAttachments,
		UpdateContext: awsUpdateCnpAccountAttachments,
		DeleteContext: awsDeleteCnpAccountAttachments,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC account id.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
			"instance_profile": {
				Type:        schema.TypeSet,
				Elem:        instanceProfileResource,
				Optional:    true,
				Description: "Instance profiles to attach to the cloud account.",
			},
			"role": {
				Type:        schema.TypeSet,
				Elem:        roleResource,
				Required:    true,
				Description: "Roles to attach to the cloud account.",
			},
		},
	}
}

func awsCreateCnpAccountAttachments(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateCnpAccountAttachments")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	accountID, err := uuid.Parse(d.Get("account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}
	profiles := make(map[string]string)
	for _, roleAttr := range d.Get("instance_profile").(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		profiles[block["key"].(string)] = block["name"].(string)
	}
	roles := make(map[string]string)
	for _, roleAttr := range d.Get("role").(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		roles[block["key"].(string)] = block["arn"].(string)
	}

	// Request artifacts be added to account.
	id, err := aws.Wrap(client).AddAccountArtifacts(ctx, aws.CloudAccountID(accountID), features, profiles, roles)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	d.SetId(id.String())

	awsReadCnpAccountAttachments(ctx, d, m)
	return nil
}

func awsReadCnpAccountAttachments(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadCnpAccountAttachments")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Request the cloud account.
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	features := &schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		features.Add(string(feature.Feature.Name))
	}

	// Request the cloud account artifacts.
	instanceProfiles, roles, err := aws.Wrap(client).AccountArtifacts(ctx, aws.CloudAccountID(id))
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	if err := d.Set("features", features); err != nil {
		return diag.FromErr(err)
	}

	instanceProfilesAttr := &schema.Set{F: schema.HashResource(instanceProfileResource)}
	for key, name := range instanceProfiles {
		instanceProfilesAttr.Add(map[string]any{"key": key, "name": name})
	}
	if err := d.Set("instance_profile", instanceProfilesAttr); err != nil {
		return diag.FromErr(err)
	}

	rolesAttr := &schema.Set{F: schema.HashResource(roleResource)}
	for key, arn := range roles {
		rolesAttr.Add(map[string]any{"key": key, "arn": arn})
	}
	if err := d.Set("role", rolesAttr); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCnpAccountAttachments(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateCnpAccountAttachments")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	var features []core.Feature
	for _, feature := range d.Get("features").(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}
	profiles := make(map[string]string)
	for _, roleAttr := range d.Get("instance_profile").(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		profiles[block["key"].(string)] = block["name"].(string)
	}
	roles := make(map[string]string)
	for _, roleAttr := range d.Get("role").(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		roles[block["key"].(string)] = block["arn"].(string)
	}

	// Request artifacts be added to account.
	_, err = aws.Wrap(client).AddAccountArtifacts(ctx, aws.CloudAccountID(id), features, profiles, roles)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsDeleteCnpAccountAttachments(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteCnpAccountAttachments")

	// Reset ID.
	d.SetId("")

	return nil
}
