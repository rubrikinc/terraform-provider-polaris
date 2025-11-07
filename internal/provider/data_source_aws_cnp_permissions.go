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
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAWSPermissionsDescription = `
The ´polaris_aws_cnp_permissions´ data source is used to access information
about the permissions required by RSC for a specified feature set.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

´CLOUD_NATIVE_ARCHIVAL´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_DYNAMODB_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´CLOUD_NATIVE_S3_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´EXOCOMPUTE´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´RSC_MANAGED_CLUSTER´ - Represents the set of permissions required for the
    Rubrik-managed Exocompute cluster.

´RDS_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´SERVERS_AND_APPS´
  * ´CLOUD_CLUSTER_ES´ - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the ´BASIC´ permission group
   is always required except for the ´SERVERS_AND_APPS´ feature.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceAwsPermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsPermissionsRead,

		Description: description(dataSourceAWSPermissionsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the customer managed policies and the managed policies.",
			},
			keyCloud: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD",
				Description: "AWS cloud type. Possible values are `STANDARD`, `CHINA` and `GOV`. Default value is " +
					"`STANDARD`.",
				ValidateFunc: validation.StringInSlice([]string{"STANDARD", "CHINA", "GOV"}, false),
			},
			keyCustomerManagedPolicies: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFeature: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "RSC feature name.",
						},
						keyName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Policy name.",
						},
						keyPolicy: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "AWS policy.",
						},
					},
				},
				Computed:    true,
				Description: "Customer managed policies.",
			},
			keyEC2RecoveryRolePath: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS EC2 recovery role path.",
			},
			keyFeature: {
				Type:        schema.TypeSet,
				Elem:        featureResource(),
				MinItems:    1,
				Required:    true,
				Description: "RSC feature with permission groups.",
			},
			keyManagedPolicies: {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Managed policies.",
			},
			keyRoleKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "RSC artifact key for the AWS role.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func awsPermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsPermissionsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloud := d.Get(keyCloud).(string)
	ec2RecoveryRolePath := d.Get(keyEC2RecoveryRolePath).(string)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block[keyName].(string)}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		features = append(features, feature)
	}
	roleKey := d.Get(keyRoleKey).(string)

	customerPolicies, managedPolicies, err := aws.Wrap(client).Permissions(ctx, cloud, features, ec2RecoveryRolePath)
	if err != nil {
		return diag.FromErr(err)
	}
	slices.SortFunc(customerPolicies, func(i, j aws.CustomerManagedPolicy) int {
		if r := cmp.Compare(i.Artifact, j.Artifact); r != 0 {
			return r
		}
		if r := cmp.Compare(i.Feature.Name, j.Feature.Name); r != 0 {
			return r
		}
		return cmp.Compare(i.Name, j.Name)
	})
	slices.SortFunc(managedPolicies, func(i, j aws.ManagedPolicy) int {
		if r := cmp.Compare(i.Artifact, j.Artifact); r != 0 {
			return r
		}
		return cmp.Compare(i.Name, j.Name)
	})

	// The hash is created from customer managed policies and managed policies
	// matching the role key.
	hash := sha256.New()

	var customerPoliciesAttr []map[string]string
	for _, policy := range customerPolicies {
		if roleKey == policy.Artifact {
			customerPoliciesAttr = append(customerPoliciesAttr, map[string]string{
				keyFeature: policy.Feature.Name,
				keyName:    policy.Name,
				keyPolicy:  policy.Policy,
			})
			hash.Write([]byte(policy.Artifact))
			hash.Write([]byte(policy.Feature.Name))
			hash.Write([]byte(policy.Name))
			hash.Write([]byte(policy.Policy))
		}
	}
	if err := d.Set(keyCustomerManagedPolicies, customerPoliciesAttr); err != nil {
		return diag.FromErr(err)
	}

	var managedPoliciesAttr []string
	for _, policy := range managedPolicies {
		if roleKey == policy.Artifact {
			managedPoliciesAttr = append(managedPoliciesAttr, policy.Name)
			hash.Write([]byte(policy.Artifact))
			hash.Write([]byte(policy.Name))
		}
	}
	if err := d.Set(keyManagedPolicies, managedPoliciesAttr); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))
	return nil
}
