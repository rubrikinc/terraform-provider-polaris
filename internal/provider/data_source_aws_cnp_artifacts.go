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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAWSArtifactsDescription = `
The ´polaris_aws_archival_location´ data source is used to access information
about instance profiles and roles required by RSC for a specified feature set.

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

´KUBERNETES_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´RDS_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´SERVERS_AND_APPS´
  * ´CLOUD_CLUSTER_ES´ - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the ´BASIC´ permission group
   is always required except for the ´SERVERS_AND_APPS´ feature.
`

func dataSourceAwsArtifacts() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsArtifactsRead,

		Description: description(dataSourceAWSArtifactsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the instance profile keys and the roles" +
					"keys.",
			},
			keyCloud: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD",
				Description: "AWS cloud type. Possible values are `STANDARD`, `CHINA` and `GOV`. Default value is " +
					"`STANDARD`.",
				ValidateFunc: validation.StringInSlice([]string{"STANDARD", "CHINA", "GOV"}, false),
			},
			keyFeature: {
				Type:        schema.TypeSet,
				Elem:        featureResource(),
				MinItems:    1,
				Required:    true,
				Description: "RSC feature with permission groups.",
			},
			keyInstanceProfileKeys: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Instance profile keys for the RSC features.",
			},
			keyRoleKeys: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Role keys for the RSC features.",
			},
		},
	}
}

func awsArtifactsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsArtifactsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	cloud := d.Get(keyCloud).(string)
	var features []core.Feature
	for _, block := range d.Get(keyFeature).(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block[keyName].(string)}
		for _, group := range block[keyPermissionGroups].(*schema.Set).List() {
			feature = feature.WithPermissionGroups(core.PermissionGroup(group.(string)))
		}

		features = append(features, feature)
	}

	// Request artifacts.
	profiles, roles, err := aws.Wrap(client).Artifacts(ctx, cloud, features)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set attributes.
	profilesAttr := &schema.Set{F: schema.HashString}
	for _, profile := range profiles {
		profilesAttr.Add(profile)
	}
	if err := d.Set(keyInstanceProfileKeys, profilesAttr); err != nil {
		return diag.FromErr(err)
	}

	rolesAttr := &schema.Set{F: schema.HashString}
	for _, role := range roles {
		rolesAttr.Add(role)
	}
	if err := d.Set(keyRoleKeys, rolesAttr); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, profile := range profiles {
		hash.Write([]byte(profile))
	}
	for _, role := range roles {
		hash.Write([]byte(role))
	}
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
