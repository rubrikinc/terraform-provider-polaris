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

// dataSourceAwsArtifacts defines the schema for the AWS artifacts data source.
func dataSourceAwsArtifacts() *schema.Resource {
	return &schema.Resource{
		ReadContext: awsArtifactsRead,

		Schema: map[string]*schema.Schema{
			"cloud": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "STANDARD",
				Description:  "AWS cloud type.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"feature": {
				Type:        schema.TypeSet,
				Elem:        featureResource,
				MinItems:    1,
				Required:    true,
				Description: "RSC feature with optional permission groups.",
			},
			"instance_profile_keys": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Instance profile keys for the RSC features.",
			},
			"role_keys": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Role keys for the RSC features.",
			},
		},
	}
}

// awsArtifactsRead run the Read operation for the AWS artifacts data source.
// Returns all the instance profiles and roles required for the specified cloud
// and feature set.
func awsArtifactsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsArtifactsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	cloud := d.Get("cloud").(string)
	var features []core.Feature
	for _, block := range d.Get("feature").(*schema.Set).List() {
		block := block.(map[string]interface{})
		feature := core.Feature{Name: block["name"].(string)}
		for _, group := range block["permission_groups"].(*schema.Set).List() {
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
	if err := d.Set("instance_profile_keys", profilesAttr); err != nil {
		return diag.FromErr(err)
	}

	rolesAttr := &schema.Set{F: schema.HashString}
	for _, role := range roles {
		rolesAttr.Add(role)
	}
	if err := d.Set("role_keys", rolesAttr); err != nil {
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
