// Copyright 2021 Rubrik, Inc.
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
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceGCPPermissionsDescription = `
The ´polaris_gcp_permissions´ data source is used to access information about
the permissions required by RSC for an RSC feature.

The ´polaris_gcp_permissions´ data source can be used with the
´google_project_iam_custom_role´ resource and the ´permissions´ field of the
´polaris_gcp_project´ resource to automatically update the permissions of roles
and notify RSC about the updated.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature.

´CLOUD_NATIVE_ARCHIVAL´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´ENCRYPTION´ - Represents the set of permissions required for encryption
    operation.

´CLOUD_NATIVE_PROTECTION´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.
  * ´EXPORT_AND_RESTORE´ - Represents the set of permissions required for export
    and restore operations.
  * ´FILE_LEVEL_RECOVERY´ - Represents the set of permissions required for
    file-level recovery operations.

´GCP_SHARED_VPC_HOST´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

´EXOCOMPUTE´
  * ´BASIC´ - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the ´BASIC´ permission group
   is always required .

-> **Note:** Due to backward compatibility, the ´features´ field allow the
   feature names to be given in 3 different styles: ´EXAMPLE_FEATURE_NAME´,
   ´example-feature-name´ or ´example_feature_name´. The recommended style is
   ´EXAMPLE_FEATURE_NAME´ as it is what the RSC API itself uses.
`

func dataSourceGcpPermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: gcpPermissionsRead,

		Description: description(dataSourceGCPPermissionsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the required permissions, will be updated as the required permissions " +
					"changes.",
			},
			keyConditions: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Conditions for the permissions with conditions.",
			},
			keyFeature: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyFeatures},
				Description: "RSC feature. Note that the feature must be given in the `EXAMPLE_FEATURE_NAME` style. " +
					"Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, `GCP_SHARED_VPC_HOST` " +
					"and `EXOCOMPUTE`.",
				ValidateFunc: validation.StringInSlice([]string{
					"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "GCP_SHARED_VPC_HOST", "EXOCOMPUTE",
				}, false),
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "GCP_SHARED_VPC_HOST", "EXOCOMPUTE",
					}, false),
				},
				Optional:     true,
				MinItems:     1,
				ExactlyOneOf: []string{keyFeature},
				Description: "RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, " +
					"`GCP_SHARED_VPC_HOST` and `EXOCOMPUTE`. **Deprecated:** use `feature` instead.",
				Deprecated: "Use `feature` instead",
			},
			keyHash: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the permissions, can be used to detect changes to the permissions. " +
					"**Deprecated:** use `id` instead.",
				Deprecated: "Use `id` instead.",
			},
			keyPermissionGroups: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"BASIC", "ENCRYPTION", "EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY",
					}, false),
				},
				Optional:      true,
				ConflictsWith: []string{keyFeatures},
				RequiredWith:  []string{keyFeature},
				Description: "Permission groups for the RSC feature. Possible values are `BASIC`, `ENCRYPTION`, " +
					"`EXPORT_AND_RESTORE` and `FILE_LEVEL_RECOVERY`.",
			},
			keyPermissions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "Permissions required for the set of RSC features. Includes permissions with " +
					"conditions. **Deprecated:** use `with_conditions` and `without_conditions` instead.",
				Deprecated: "use: `with_conditions` and `without_conditions` instead.",
			},
			keyServices: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "GCP services required for the RSC feature.",
			},
			keyWithConditions: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Permissions with conditions required for the RSC feature.",
			},
			keyWithoutConditions: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Permissions without conditions required for the RSC feature.",
			},
		},
	}
}

func gcpPermissionsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpPermissionsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Check both feature and features.
	var features []core.Feature
	if feature, ok := d.GetOk(keyFeature); ok {
		var pgs []core.PermissionGroup
		for _, group := range d.Get(keyPermissionGroups).(*schema.Set).List() {
			pgs = append(pgs, core.PermissionGroup(group.(string)))
		}
		features = append(features, core.Feature{Name: feature.(string), PermissionGroups: pgs})
	} else {
		for _, feature := range d.Get(keyFeatures).(*schema.Set).List() {
			features = append(features, core.ParseFeatureNoValidation(feature.(string)))
		}
	}
	featurePerms, err := gcp.Wrap(client).FeaturePermissions(ctx, features)
	if err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	if len(featurePerms) == 1 {
		var permissionGroups []any
		for _, pg := range featurePerms[0].Feature.PermissionGroups {
			permissionGroups = append(permissionGroups, string(pg))
		}
		if err := d.Set(keyPermissionGroups, permissionGroups); err != nil {
			return diag.FromErr(err)
		}

		var conditions []any
		for _, condition := range featurePerms[0].Conditions {
			conditions = append(conditions, condition)
			hash.Write([]byte(condition))
		}
		if err := d.Set(keyConditions, conditions); err != nil {
			return diag.FromErr(err)
		}

		var withConditions []any
		for _, perm := range featurePerms[0].WithConditions {
			withConditions = append(withConditions, perm)
			hash.Write([]byte(perm))
		}
		if err := d.Set(keyWithConditions, withConditions); err != nil {
			return diag.FromErr(err)
		}

		var withoutConditions []any
		for _, perm := range featurePerms[0].WithoutConditions {
			withoutConditions = append(withoutConditions, perm)
			hash.Write([]byte(perm))
		}
		if err := d.Set(keyWithoutConditions, withoutConditions); err != nil {
			return diag.FromErr(err)
		}

		var services []any
		for _, service := range featurePerms[0].Services {
			if service == "COMPUTE_ENGINE_API" {
				service = "COMPUTE"
			}
			services = append(services, strings.ToLower(strings.Replace(service, "_", "", -1)+".googleapis.com"))
			hash.Write([]byte(service))
		}
		if err := d.Set(keyServices, services); err != nil {
			return diag.FromErr(err)
		}
	}

	var perms []string
	for _, featurePerms := range featurePerms {
		perms = append(perms, featurePerms.WithConditions...)
		perms = append(perms, featurePerms.WithoutConditions...)
	}
	slices.Sort(perms)
	perms = slices.Compact(perms)
	var permissions []any
	for _, perm := range perms {
		permissions = append(permissions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyPermissions, &permissions); err != nil {
		return diag.FromErr(err)
	}

	id := fmt.Sprintf("%x", hash.Sum(nil))
	if err := d.Set(keyHash, id); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return nil
}
