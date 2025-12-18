// Copyright 2025 Rubrik, Inc.
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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/gcp"
)

const dataSourcePermissionGroupsDescription = `
The ´polaris_permission_groups´ data source provides information about the
required permission groups for each feature, organized by cloud provider.

This data source is useful for programmatically determining which permission
groups are needed when enabling specific features for AWS, Azure, or GCP cloud
accounts.

-> **Note:** The features and permission groups returned are queried dynamically
   from RSC and may change over time as new features are added.
`

func dataSourcePermissionGroups() *schema.Resource {
	return &schema.Resource{
		ReadContext: permissionGroupsRead,

		Description: description(dataSourcePermissionGroupsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the cloud provider.",
			},
			keyCloudProvider: {
				Type:     schema.TypeString,
				Required: true,
				Description: "Cloud provider to get permission groups for. " +
					"Possible values are `AWS`, `AZURE`, and `GCP`.",
				ValidateFunc: validation.StringInSlice([]string{"AWS", "AZURE", "GCP"}, false),
			},
			keyFeatures: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of features with their required permission groups.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Feature name.",
						},
						keyPermissionGroups: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed:    true,
							Description: "Permission groups required for the feature.",
						},
					},
				},
			},
		},
	}
}

// awsFeatures returns the AWS features to query for permission groups.
func awsFeatures() []core.Feature {
	return []core.Feature{
		core.FeatureCloudNativeArchival,
		core.FeatureCloudNativeProtection,
		core.FeatureCloudNativeDynamoDBProtection,
		core.FeatureCloudNativeS3Protection,
		core.FeatureExocompute,
		core.FeatureKubernetesProtection,
		core.FeatureRDSProtection,
		core.FeatureServerAndApps,
	}
}

// azureFeatures returns the Azure features to query for permission groups.
func azureFeatures() []core.Feature {
	return []core.Feature{
		core.FeatureAzureSQLDBProtection,
		core.FeatureAzureSQLMIProtection,
		core.FeatureCloudNativeArchival,
		core.FeatureCloudNativeArchivalEncryption,
		core.FeatureCloudNativeBlobProtection,
		core.FeatureCloudNativeProtection,
		core.FeatureExocompute,
		core.FeatureServerAndApps,
	}
}

// gcpFeatures returns the GCP features to query for permission groups.
func gcpFeatures() []core.Feature {
	return []core.Feature{
		core.FeatureCloudNativeArchival,
		core.FeatureCloudNativeProtection,
		core.FeatureExocompute,
		core.FeatureGCPSharedVPCHost,
		core.FeatureServerAndApps,
	}
}

func permissionGroupsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "permissionGroupsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudProvider := d.Get(keyCloudProvider).(string)

	var featuresAttr []map[string]any
	switch cloudProvider {
	case "AWS":
		result, err := aws.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, awsFeatures())
		if err != nil {
			return diag.FromErr(err)
		}
		for _, fp := range result {
			var permissionGroups []string
			for _, pg := range fp.PermissionsGroupPermissions {
				permissionGroups = append(permissionGroups, string(pg.PermissionsGroup))
			}
			featuresAttr = append(featuresAttr, map[string]any{
				keyName:             fp.Feature,
				keyPermissionGroups: permissionGroups,
			})
		}
	case "AZURE":
		result, err := azure.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, azureFeatures())
		if err != nil {
			return diag.FromErr(err)
		}
		for _, fp := range result {
			var permissionGroups []string
			for _, pg := range fp.PermissionsGroupPermissions {
				permissionGroups = append(permissionGroups, string(pg.PermissionsGroup))
			}
			featuresAttr = append(featuresAttr, map[string]any{
				keyName:             fp.Feature,
				keyPermissionGroups: permissionGroups,
			})
		}
	case "GCP":
		result, err := gcp.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, gcpFeatures())
		if err != nil {
			return diag.FromErr(err)
		}
		for _, fp := range result {
			var permissionGroups []string
			for _, pg := range fp.PermissionGroups {
				permissionGroups = append(permissionGroups, string(pg.PermissionGroupType))
			}
			featuresAttr = append(featuresAttr, map[string]any{
				keyName:             fp.Feature,
				keyPermissionGroups: permissionGroups,
			})
		}
	default:
		return diag.Errorf("unsupported cloud provider: %s", cloudProvider)
	}

	if err := d.Set(keyFeatures, featuresAttr); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	hash.Write([]byte(cloudProvider))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
