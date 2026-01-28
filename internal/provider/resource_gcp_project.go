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
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceGCPProjectDescription = `
The ´polaris_gcp_project´ resource adds a GCP project to RSC.

The ´permissions´ field of the each feature can be used with the 
´polaris_gcp_permissions´ data source to notify RSC about permission updates
when the Terraform configuration is applied.

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
  * ´AUTOMATED_NETWORKING_SETUP´ - Represents the set of permissions required
    for automated networking setup. When automated networking setup is enabled,
    RSC is responsible for creating and maintaining the networking resources for
    Exocompute. See the ´polaris_gcp_exocompute´ resource for more information.
`

func resourceGcpProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateProject,
		ReadContext:   gcpReadProject,
		UpdateContext: gcpUpdateProject,
		DeleteContext: gcpDeleteProject,

		Description: description(resourceGCPProjectDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyCloudNativeProtection: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyStatus: {
							Type:        schema.TypeString,
							Optional:    true, // Workaround for an issue with the TF doc generation.
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
					},
				},
				MaxItems:     1,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyFeature},
				Description:  "Enable the Cloud Native Protection feature for the GCP project. **Deprecated:** use `feature` instead.",
				Deprecated:   "Use `feature` instead.",
			},
			keyCredentials: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Description:  "Base64 encoded GCP service account private key or path to GCP service account key file.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyDeleteSnapshotsOnDestroy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed. Default value is `false`.",
			},
			keyFeature: {
				Type:         schema.TypeSet,
				Elem:         gcpFeatureResourceWithPermissionsAndStatus(),
				Optional:     true,
				Computed:     true,
				MinItems:     1,
				ExactlyOneOf: []string{keyCloudNativeProtection},
				Description:  "RSC feature to enable for the GCP project.",
			},
			keyOrganizationName: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "GCP organization name.",
			},
			keyPermissionsHash: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Signals that the permissions has been updated. **Deprecated:** use the `permissions` " +
					"field of `feature` instead.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Deprecated:   "Use the `permissions` field of `feature` instead.",
			},
			keyProject: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "GCP project ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyProjectName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "GCP project name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyProjectNumber: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "GCP project number.",
				ValidateFunc: validateStringIsNumber,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceGcpProjectV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpProjectStateUpgradeV0,
			Version: 0,
		}, {
			Type:    resourceGcpProjectV1().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpProjectStateUpgradeV1,
			Version: 1,
		}},
	}
}

func gcpCreateProject(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateProject")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	project, err := fromCredentials(d)
	if err != nil {
		return diag.FromErr(err)
	}
	var opts []gcp.OptionFunc
	if name, ok := d.GetOk(keyProjectName); ok {
		opts = append(opts, gcp.Name(name.(string)))
	}
	if orgName, ok := d.GetOk(keyOrganizationName); ok {
		opts = append(opts, gcp.Organization(orgName.(string)))
	}

	config, err := project(ctx)
	if err != nil {
		return diag.Errorf("failed to lookup native project id: %s", err)
	}
	account, err := gcp.Wrap(client).ProjectByNativeID(ctx, config.NativeID)
	if err == nil {
		return diag.Errorf("project %q already added to polaris", account.NativeID)
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return diag.FromErr(err)
	}

	var features []core.Feature
	if blocks, ok := d.GetOk(keyFeature); ok {
		for _, block := range blocks.(*schema.Set).List() {
			features = append(features, fromGCPFeatureResource(block.(map[string]any)))
		}
	}
	if _, ok := d.GetOk(keyCloudNativeProtection); ok {
		features = append(features, core.FeatureCloudNativeProtection)
	}

	id, err := gcp.Wrap(client).AddProject(ctx, project, features, opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	gcpReadProject(ctx, d, m)
	return nil
}

func gcpReadProject(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadProject")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := gcp.Wrap(client).ProjectByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyOrganizationName, account.OrganizationName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyProject, account.NativeID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyProjectName, account.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyProjectNumber, strconv.FormatInt(account.ProjectNumber, 10)); err != nil {
		return diag.FromErr(err)
	}

	// Keep the permissions from the state.
	permissions := make(map[string]string)
	for _, feature := range d.Get(keyFeature).(*schema.Set).List() {
		f := feature.(map[string]any)
		permissions[f[keyName].(string)] = f[keyPermissions].(string)
	}

	featureSet := &schema.Set{F: schema.HashResource(gcpFeatureResourceWithPermissionsAndStatus())}
	for _, feature := range account.Features {
		featureSet.Add(toGCPFeatureResourceWithPermissionsAndStatus(feature.Feature, permissions[feature.Name], feature.Status))
	}
	if err := d.Set(keyFeature, featureSet); err != nil {
		return diag.FromErr(err)
	}

	var cnpBlock []any
	if cnpFeature, ok := account.Feature(core.FeatureCloudNativeProtection); ok {
		cnpBlock = append(cnpBlock, map[string]any{
			keyStatus: core.FormatStatus(cnpFeature.Status),
		})
	}
	if err := d.Set(keyCloudNativeProtection, cnpBlock); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateProject(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateProject")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(keyCredentials) {
		if err := gcp.Wrap(client).SetProjectServiceAccount(ctx, id, gcp.Key(d.Get(keyCredentials).(string))); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(keyFeature) {
		project, err := fromCredentials(d)
		if err != nil {
			return diag.FromErr(err)
		}

		deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
		var opts []gcp.OptionFunc
		if name, ok := d.GetOk(keyProjectName); ok {
			opts = append(opts, gcp.Name(name.(string)))
		}
		if orgName, ok := d.GetOk(keyOrganizationName); ok {
			opts = append(opts, gcp.Organization(orgName.(string)))
		}

		addFeatures, removeFeatures, permFeatures := diffGCPFeatureResource(d)
		if len(addFeatures) > 0 {
			if _, err := gcp.Wrap(client).AddProject(ctx, project, addFeatures, opts...); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(removeFeatures) > 0 {
			if err := gcp.Wrap(client).RemoveProject(ctx, id, removeFeatures, deleteSnapshots); err != nil {
				return diag.FromErr(err)
			}
		}
		if len(permFeatures) > 0 {
			if err := gcp.Wrap(client).PermissionsUpdated(ctx, id, permFeatures); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(keyPermissionsHash) {
		err = gcp.Wrap(client).PermissionsUpdated(ctx, id, nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gcpReadProject(ctx, d, m)
	return nil
}

func gcpDeleteProject(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteProject")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	deleteSnapshots := d.Get(keyDeleteSnapshotsOnDestroy).(bool)
	var features []core.Feature
	if blocks, ok := d.GetOk(keyFeature); ok {
		for _, block := range blocks.(*schema.Set).List() {
			features = append(features, fromGCPFeatureResource(block.(map[string]any)))
		}
	}

	err = gcp.Wrap(client).RemoveProject(ctx, id, features, deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func fromCredentials(d *schema.ResourceData) (gcp.ProjectFunc, error) {
	credentials := d.Get(keyCredentials).(string)
	projectID := d.Get(keyProject).(string)

	var projectNumber int64
	if pn, ok := d.GetOk(keyProjectNumber); ok {
		var err error
		projectNumber, err = strconv.ParseInt(pn.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("project_number should be an integer: %s", err)
		}
	}

	if credentials != "" {
		return gcp.KeyWithProjectAndNumber(credentials, projectID, projectNumber), nil
	}

	return gcp.Project(projectID, projectNumber), nil
}

func gcpFeatureResourceWithPermissionsAndStatus() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyName: {
				Type:     schema.TypeString,
				Required: true,
				Description: "RSC feature name. Possible values are `CLOUD_NATIVE_ARCHIVAL`, " +
					"`CLOUD_NATIVE_PROTECTION`, `GCP_SHARED_VPC_HOST` and `EXOCOMPUTE`.",
				ValidateFunc: validation.StringInSlice([]string{
					"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "GCP_SHARED_VPC_HOST", "EXOCOMPUTE",
				}, false),
			},
			keyPermissionGroups: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"BASIC", "ENCRYPTION", "EXPORT_AND_RESTORE", "FILE_LEVEL_RECOVERY",
						"AUTOMATED_NETWORKING_SETUP",
					}, false),
				},
				Required: true,
				Description: "Permission groups for the RSC feature. Possible values are `BASIC`, `ENCRYPTION`, " +
					"`EXPORT_AND_RESTORE`, `FILE_LEVEL_RECOVERY` and `AUTOMATED_NETWORKING_SETUP`.",
			},
			keyPermissions: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Permissions updated signal. When this field changes, the provider will notify " +
					"RSC that the permissions for the feature has been updated. Use this field with the " +
					"`polaris_gcp_permissions` data source.",
				ValidateFunc: validation.StringIsNotWhiteSpace},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the feature.",
			},
		},
	}
}

func fromGCPFeatureResourceWithPermissions(block map[string]any) (core.Feature, string) {
	var pgs []core.PermissionGroup
	for _, pg := range block[keyPermissionGroups].(*schema.Set).List() {
		pgs = append(pgs, core.PermissionGroup(pg.(string)))
	}

	return core.Feature{Name: block[keyName].(string), PermissionGroups: pgs}, block[keyPermissions].(string)
}

func toGCPFeatureResourceWithPermissionsAndStatus(feature core.Feature, permissions string, status core.Status) map[string]any {
	pgs := &schema.Set{F: schema.HashString}
	for _, pg := range feature.PermissionGroups {
		pgs.Add(string(pg))
	}

	return map[string]any{
		keyName:             feature.Name,
		keyPermissionGroups: pgs,
		keyPermissions:      permissions,
		keyStatus:           core.FormatStatus(status),
	}
}

type featureWithPermissions struct {
	core.Feature
	permissions string
}

// add, remove and update perms.
func diffGCPFeatureResource(d *schema.ResourceData) ([]core.Feature, []core.Feature, []core.Feature) {
	oldResource, newResource := d.GetChange(keyFeature)

	newSet := make(map[string]featureWithPermissions)
	for _, block := range newResource.(*schema.Set).List() {
		feature, permissions := fromGCPFeatureResourceWithPermissions(block.(map[string]any))
		newSet[feature.Name] = featureWithPermissions{Feature: feature, permissions: permissions}
	}
	oldSet := make(map[string]featureWithPermissions)
	for _, block := range oldResource.(*schema.Set).List() {
		feature, permissions := fromGCPFeatureResourceWithPermissions(block.(map[string]any))
		oldSet[feature.Name] = featureWithPermissions{Feature: feature, permissions: permissions}
	}

	var permSlice []core.Feature
	for name, newResource := range newSet {
		if oldResource, ok := oldSet[name]; ok {
			if newResource.permissions != oldResource.permissions {
				permSlice = append(permSlice, newResource.Feature)
			}
		}
	}

	for name, oldResource := range oldSet {
		if newResource, ok := newSet[name]; ok {
			delete(oldSet, name)
			if newResource.Feature.DeepEqual(oldResource.Feature) {
				delete(newSet, name)
			}
		}
	}

	addSlice := make([]core.Feature, 0, len(newSet))
	for _, feature := range newSet {
		addSlice = append(addSlice, feature.Feature)
	}
	removeSlice := make([]core.Feature, 0, len(oldSet))
	for _, feature := range oldSet {
		removeSlice = append(removeSlice, feature.Feature)
	}

	return addSlice, removeSlice, permSlice
}
