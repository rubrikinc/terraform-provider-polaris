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
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAWSCNPAccountTrustPolicyDescription = `
The ´aws_cnp_account_trust_policy´ resource gets the AWS IAM trust policies
required by RSC. The ´policy´ field of ´aws_cnp_account_trust_policy´ resource
should be used with the ´assume_role_policy´ of the ´aws_iam_role´ resource.

~> **Note:** The ´polaris_aws_cnp_account´ resource can now be used to get the
   IAM trust policies for all role keys. The ´aws_cnp_account_trust_policy´
   resource is no longer required and will be deprecated in a future version.

~> **Note:** Once ´external_id´ has been set it cannot be changed. Unless the
   cloud account is removed and onboarded again.

-> **Note:** The ´features´ field takes only the feature names and not the
   permission groups associated with the features.
`

var trustPolicyRoleKeys = []string{
	"CROSSACCOUNT",
	"EXOCOMPUTE_EKS_MASTERNODE",
	"EXOCOMPUTE_EKS_WORKERNODE",
	"EXOCOMPUTE_EKS_LAMBDA",
}

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceAwsCnpAccountTrustPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCnpAccountTrustPolicy,
		ReadContext:   awsReadCnpAccountTrustPolicy,
		UpdateContext: awsUpdateCnpAccountTrustPolicy,
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
				ForceNew: true,
				Description: "Trust policy external ID. If not specified, RSC will generate an external ID. " +
					"Note, once the external ID has been set it cannot be changed. Changing this forces a new " +
					"resource to be created.",
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "CLOUD_NATIVE_DYNAMODB_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION",
						"EXOCOMPUTE", "RDS_PROTECTION", "SERVERS_AND_APPS",
					}, false),
				},
				MinItems: 1,
				Optional: true,
				Description: "RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, " +
					"`CLOUD_NATIVE_DYNAMODB_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, `SERVERS_AND_APPS`, `EXOCOMPUTE` and `RDS_PROTECTION`. " +
					"**Deprecated:** no longer used by the provider, any value set is ignored.",
				Deprecated: "No longer used by the provider, any value set is ignored.",
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
					"`EXOCOMPUTE_EKS_MASTERNODE`, `EXOCOMPUTE_EKS_WORKERNODE` and `EXOCOMPUTE_EKS_LAMBDA`. Changing " +
					"this forces a new resource to be created.",
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

	account, err := aws.Wrap(client).AccountByID(ctx, accountID)
	if err != nil {
		return diag.FromErr(err)
	}

	policy, err := trustPolicyForRoleKey(ctx, client, roleKey, account, externalID)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPolicy, policy); err != nil {
		return diag.FromErr(err)
	}

	trustPolicyID, err := joinTrustPolicyID(roleKey, accountID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(trustPolicyID)
	return nil
}

func awsReadCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCnpAccountTrustPolicy")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	roleKey, accountID, _, err := splitTrustPolicyID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	externalID := d.Get(keyExternalID).(string)

	account, err := aws.Wrap(client).AccountByID(ctx, accountID)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	policy, err := trustPolicyForRoleKey(ctx, client, roleKey, account, externalID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyAccountID, accountID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPolicy, policy); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRoleKey, roleKey); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// awsUpdateCnpAccountTrustPolicy updates the account trust policy. Only the
// features field can be updated without forcing a new resource. This is to
// allow users to remove the deprecated field. It no longer has any effect on
// the configuration.
func awsUpdateCnpAccountTrustPolicy(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCnpAccountTrustPolicy")

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
	tflog.Trace(ctx, "awsImportCnpAccountTrustPolicy")

	roleKey, accountID, externalID, err := splitTrustPolicyID(d.Id())
	if err != nil {
		return nil, err
	}
	id, err := joinTrustPolicyID(roleKey, accountID)
	if err != nil {
		return nil, err
	}

	if err := d.Set(keyRoleKey, roleKey); err != nil {
		return nil, err
	}
	if externalID != "" {
		if err := d.Set(keyExternalID, externalID); err != nil {
			return nil, err
		}
	}

	d.SetId(id)
	return []*schema.ResourceData{d}, nil
}

func joinTrustPolicyID(roleKey string, accountID uuid.UUID) (string, error) {
	if slices.Contains(trustPolicyRoleKeys, roleKey) {
		return fmt.Sprintf("%s-%s", roleKey, accountID), nil
	}

	return "", fmt.Errorf("invalid role key: %s", roleKey)
}

// The external ID at the end is optional.
var reSplitTrustPolicyID = regexp.MustCompile(fmt.Sprintf(`^(%s)-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:-(.+))*$`, strings.Join(trustPolicyRoleKeys, "|")))

// splitTrustPolicyID splits the trust policy id into the role key and the
// account id. During import a trust policy ID can contain an optional external
// ID.
func splitTrustPolicyID(id string) (string, uuid.UUID, string, error) {
	match := reSplitTrustPolicyID.FindStringSubmatch(id)
	if len(match) != 3 && len(match) != 4 {
		return "", uuid.Nil, "", fmt.Errorf("invalid resource id: %s", id)
	}

	accountID, err := uuid.Parse(match[2])
	if err != nil {
		return "", uuid.Nil, "", err
	}
	var externalID string
	if len(match) == 4 {
		externalID = match[3]
	}

	return match[1], accountID, externalID, nil
}

// trustPolicyForRoleKey returns the trust policy for the specified role key.
func trustPolicyForRoleKey(ctx context.Context, client *polaris.Client, roleKey string, account aws.CloudAccount, externalID string) (string, error) {
	features := make([]core.Feature, 0, len(account.Features))
	for _, feature := range account.Features {
		features = append(features, feature.Feature)
	}
	trustPolicies, err := aws.Wrap(client).TrustPolicies(ctx, gqlaws.Cloud(account.Cloud), account.ID, features, externalID)
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
