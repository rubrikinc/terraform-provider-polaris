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
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceAwsPermissions defines the schema for the AWS permissions data
// source.
func dataSourceAwsPermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsPermissionsRead,

		Schema: map[string]*schema.Schema{
			"cloud": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "STANDARD",
				Description:  "AWS cloud type.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"customer_managed_policies": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"feature": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "RSC Feature.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Policy name.",
						},
						"policy": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Policy.",
						},
					},
				},
				Computed:    true,
				Description: "Customer managed policies.",
			},
			"ec2_recovery_role_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "EC2 recovery role path.",
			},
			"feature": {
				Type:        schema.TypeSet,
				Elem:        featureResource,
				MinItems:    1,
				Required:    true,
				Description: "RSC feature with optional permission groups.",
			},
			"managed_policies": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Managed policies.",
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

// awsPermissionsRead run the Read operation for the AWS permissions data
// source. Returns all AWS permissions needed of for the specified cloud and
// feature set.
func awsPermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsPermissionsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	cloud := d.Get("cloud").(string)
	ec2RecoveryRolePath := d.Get("ec2_recovery_role_path").(string)
	var features []core.Feature
	for _, block := range d.Get("feature").(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block["name"].(string)}
		for _, group := range block["permission_groups"].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		features = append(features, feature)
	}
	roleKey := d.Get("role_key").(string)

	// Request permissions.
	customerPolicies, managedPolicies, err := aws.Wrap(client).Permissions(ctx, cloud, features, ec2RecoveryRolePath)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	var customerPoliciesAttr []map[string]string
	for _, policy := range customerPolicies {
		if roleKey == policy.Artifact {
			customerPoliciesAttr = append(customerPoliciesAttr, map[string]string{
				"feature": policy.Feature.Name,
				"name":    policy.Name,
				"policy":  policy.Policy,
			})
		}
	}
	if err := d.Set("customer_managed_policies", customerPoliciesAttr); err != nil {
		return diag.FromErr(err)
	}

	var managedPoliciesAttr []string
	for _, policy := range managedPolicies {
		if roleKey == policy.Artifact {
			managedPoliciesAttr = append(managedPoliciesAttr, policy.Name)
		}
	}
	if err := d.Set("managed_policies", managedPoliciesAttr); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, policy := range customerPolicies {
		hash.Write([]byte(policy.Artifact))
		hash.Write([]byte(policy.Feature.Name))
		hash.Write([]byte(policy.Name))
		hash.Write([]byte(policy.Policy))
	}
	for _, policy := range managedPolicies {
		hash.Write([]byte(policy.Artifact))
		hash.Write([]byte(policy.Name))
	}
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
