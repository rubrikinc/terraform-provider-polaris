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

const resourceAWSCNPAccountAttachmentsDescription = `
The ´aws_cnp_account_attachments´ resource attaches AWS instance profiles and AWS
roles to an RSC cloud account.

-> **Note:** The ´features´ field takes only the feature names and not the permission
   groups associated with the features.
`

func resourceAwsCnpAccountAttachments() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccountAttachments,
		ReadContext:   awsReadCnpAccountAttachments,
		UpdateContext: awsUpdateCnpAccountAttachments,
		DeleteContext: awsDeleteCnpAccountAttachments,

		Description: description(resourceAWSCNPAccountAttachmentsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION",
						"EXOCOMPUTE", "RDS_PROTECTION",
					}, false),
				},
				MinItems: 1,
				Required: true,
				Description: "RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, " +
					"`CLOUD_NATIVE_S3_PROTECTION`, `EXOCOMPUTE` and `RDS_PROTECTION`.",
			},
			keyInstanceProfile: {
				Type:        schema.TypeSet,
				Elem:        instanceProfileResource(),
				Optional:    true,
				Description: "Instance profiles to attach to the cloud account.",
			},
			keyRole: {
				Type:        schema.TypeSet,
				Elem:        roleResource(),
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

	accountID, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	var features []core.Feature
	for _, feature := range d.Get(keyFeatures).(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}
	profiles := make(map[string]string)
	for _, roleAttr := range d.Get(keyInstanceProfile).(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		profiles[block["key"].(string)] = block[keyName].(string)
	}
	roles := make(map[string]string)
	for _, roleAttr := range d.Get(keyRole).(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		roles[block[keyKey].(string)] = block[keyARN].(string)
	}

	// Request artifacts be added to account.
	id, err := aws.Wrap(client).AddAccountArtifacts(ctx, aws.CloudAccountID(accountID), features, profiles, roles)
	if err != nil {
		return diag.FromErr(err)
	}

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
		features.Add(feature.Feature.Name)
	}

	// Request the cloud account artifacts.
	instanceProfiles, roles, err := aws.Wrap(client).AccountArtifacts(ctx, aws.CloudAccountID(id))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyFeatures, features); err != nil {
		return diag.FromErr(err)
	}

	instanceProfilesAttr := &schema.Set{F: schema.HashResource(instanceProfileResource())}
	for key, name := range instanceProfiles {
		instanceProfilesAttr.Add(map[string]any{keyKey: key, keyName: name})
	}
	if err := d.Set(keyInstanceProfile, instanceProfilesAttr); err != nil {
		return diag.FromErr(err)
	}

	oldRoles := make(map[string]string)
	for _, role := range d.Get(keyRole).(*schema.Set).List() {
		block := role.(map[string]any)
		oldRoles[block[keyKey].(string)] = block[keyPermissions].(string)
	}
	rolesAttr := &schema.Set{F: schema.HashResource(roleResource())}
	for key, arn := range roles {
		rolesAttr.Add(map[string]any{keyKey: key, keyARN: arn, keyPermissions: oldRoles[key]})
	}
	if err := d.Set(keyRole, rolesAttr); err != nil {
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

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var features []core.Feature
	for _, feature := range d.Get(keyFeatures).(*schema.Set).List() {
		features = append(features, core.Feature{Name: feature.(string)})
	}
	profiles := make(map[string]string)
	for _, roleAttr := range d.Get(keyInstanceProfile).(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		profiles[block[keyKey].(string)] = block[keyName].(string)
	}
	roles := make(map[string]string)
	for _, roleAttr := range d.Get(keyRole).(*schema.Set).List() {
		block := roleAttr.(map[string]any)
		roles[block[keyKey].(string)] = block[keyARN].(string)
	}

	// Update artifacts.
	_, err = aws.Wrap(client).AddAccountArtifacts(ctx, aws.CloudAccountID(id), features, profiles, roles)
	if err != nil {
		return diag.FromErr(err)
	}

	// Notify RSC about updated permissions. Note, we notify RSC that the
	// permissions for all features have been updated without checking the
	// permissions hash, the reason is there is no way for us to connect a role
	// to a feature.
	if err := aws.Wrap(client).PermissionsUpdated(ctx, id, nil); err != nil {
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

func instanceProfileResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "RSC artifact key for the AWS instance profile.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS instance profile name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func roleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "RSC artifact key for the AWS role.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyARN: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS role ARN.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermissions: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Permissions updated signal. When this field changes, the provider will notify " +
					"RSC that the permissions for the feature has been updated. Use this field with the `id` field " +
					"of the `polaris_aws_cnp_permissions` data source.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}
