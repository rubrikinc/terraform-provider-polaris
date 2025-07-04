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
)

const resourceAWSCNPAccountTrustPolicyDescription = `
The ´aws_cnp_account_trust_policy´ resource gets the AWS IAM trust policies
required by RSC. The ´policy´ field of ´aws_cnp_account_trust_policy´ resource
should be used with the ´assume_role_policy´ of the ´aws_iam_role´ resource.

~> **Node:** Once ´external_id´ has been set it cannot be changed. Unless the
   cloud account is removed and onboarded again.

-> **Note:** The ´features´ field takes only the feature names and not the
   permission groups associated with the features.
`

var trustPolicyRoleKeys = []string{
	"CROSSACCOUNT",
	"EXOCOMPUTE_EKS_MASTERNODE",
	"EXOCOMPUTE_EKS_WORKERNODE",
}

func resourceAwsCnpAccountTrustPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccountTrustPolicy,
		ReadContext:   awsReadCnpAccountTrustPolicy,
		DeleteContext: awsDeleteCnpAccountTrustPolicy,

		Description: description(resourceAWSCNPAccountTrustPolicyDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID) with the role key as a prefix.",
			},
			keyAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyExternalID: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Trust policy external ID. If not specified, RSC will generate an external ID. " +
					"Note, once the external ID has been set it cannot be changed.",
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
				Optional: true,
				Computed: true,
				Description: "RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, " +
					"`CLOUD_NATIVE_S3_PROTECTION`, `EXOCOMPUTE` and `RDS_PROTECTION`. **Deprecated:** no longer used " +
					"by the provider, any value set is ignored.",
				Deprecated: "no longer used by the provider, any value set is ignored.",
			},
			keyPolicy: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS IAM trust policy.",
			},
			keyRoleKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "RSC artifact key for the AWS role. Possible values are `CROSSACCOUNT`, " +
					"`EXOCOMPUTE_EKS_MASTERNODE` and `EXOCOMPUTE_EKS_WORKERNODE`.",
				ValidateFunc: validation.StringInSlice(trustPolicyRoleKeys, false),
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: awsImportCnpAccountTrustPolicy,
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAwsCnpAccountTrustPolicyV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAwsCnpAccountTrustPolicyStateUpgradeV0,
			Version: 0,
		}},
	}
}

func awsCreateCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	externalID := d.Get(keyExternalID).(string)
	roleKey := d.Get(keyRoleKey).(string)

	policy, err := trustPolicy(ctx, client, accountID, roleKey, externalID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyPolicy, policy); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(joinTrustPolicyID(accountID, roleKey))
	awsReadCnpAccountTrustPolicy(ctx, d, m)
	return nil
}

func awsReadCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, roleKey, err := splitTrustPolicyID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	policy, err := trustPolicy(ctx, client, accountID, roleKey, "")
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyAccountID, accountID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPolicy, policy); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRoleKey, roleKey); err != nil {
		return diag.FromErr(err)
	}

	// This can be removed when the features field of the resource is removed.
	account, err := aws.Wrap(client).AccountByID(ctx, core.FeatureAll, accountID)
	if err != nil {
		return diag.FromErr(err)
	}

	features := &schema.Set{F: schema.HashString}
	for _, feature := range account.Features {
		features.Add(feature.Feature.Name)
	}
	if err := d.Set(keyFeatures, features); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// awsDeleteCnpAccountTrustPolicy destroys the account trust policy. Note that
// there is no need to destroy the trust policy in RSC, we simply remove the
// trust policy from the state.
func awsDeleteCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCnpAccountTrustPolicy")

	d.SetId("")
	return nil
}

func awsImportCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	log.Print("[TRACE] awsCreateCnpAccountTrustPolicy")

	id, roleKey, err := splitTrustPolicyID(strings.ToLower(d.Id()))
	if err != nil {
		return nil, err
	}

	d.SetId(id.String())
	if err := d.Set(keyRoleKey, roleKey); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func joinTrustPolicyID(accountID uuid.UUID, roleKey string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(roleKey), accountID)
}

func splitTrustPolicyID(id string) (uuid.UUID, string, error) {
	id = strings.ToLower(id)

	for _, roleKey := range trustPolicyRoleKeys {
		lcRoleKey := strings.ToLower(roleKey)
		if strings.HasPrefix(id, lcRoleKey) {
			id, err := uuid.Parse(strings.TrimPrefix(id, lcRoleKey))
			if err != nil {
				return uuid.Nil, "", err
			}

			return id, roleKey, nil
		}
	}

	return uuid.Nil, "", fmt.Errorf("invalid resource id: %s", id)
}

// trustPolicy returns the trust policy for the specified role key.
func trustPolicy(ctx context.Context, client *polaris.Client, accountID uuid.UUID, roleKey, externalID string) (string, error) {
	trustPolicies, err := aws.Wrap(client).TrustPolicies(ctx, accountID, externalID)
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
