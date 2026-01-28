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
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceGCPProjectDescription = `
The ´polaris_gcp_project´ data source is used to access information about a GCP
project added to RSC. A GCP project is looked up using either the GCP project
ID, the GCP project number, the RSC cloud account ID or the name.

-> **Note:** The project name is the name of the GCP project as it appears in
   RSC.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceGcpProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: gcpProjectRead,

		Description: description(dataSourceGCPProjectDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyName, keyProjectID, keyProjectNumber},
				Description:  "RSC cloud account ID (UUID).",
			},
			keyFeature: {
				Type:        schema.TypeSet,
				Elem:        gcpFeatureResourceWithStatus(),
				Computed:    true,
				Description: "RSC feature with permission groups and status.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyProjectID, keyProjectNumber},
				Description:  "GCP project name.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			keyOrganizationName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "GCP organization name.",
			},
			keyProjectID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyName, keyProjectNumber},
				Description:  "GCP project ID.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			keyProjectNumber: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{keyID, keyName, keyProjectID},
				Description:  "GCP project number.",
				ValidateFunc: validateStringIsNumber,
			},
		},
	}
}

func gcpProjectRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpProjectRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// We don't allow prefix searches since it would be impossible to uniquely
	// identify a project with a name being the prefix of another project.
	var project gcp.CloudAccount
	switch {
	case d.Get(keyProjectID).(string) != "":
		project, err = gcp.Wrap(client).ProjectByNativeID(ctx, d.Get(keyProjectID).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	case d.Get(keyProjectNumber).(string) != "":
		n, err := strconv.ParseInt(d.Get(keyProjectNumber).(string), 10, 64)
		if err != nil {
			return diag.Errorf("failed to parse project number: %s", err)
		}
		project, err = gcp.Wrap(client).ProjectByProjectNumber(ctx, n)
		if err != nil {
			return diag.FromErr(err)
		}
	case d.Get(keyName).(string) != "":
		project, err = gcp.Wrap(client).ProjectByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	default:
		id, err := uuid.Parse(d.Get(keyID).(string))
		if err != nil {
			return diag.Errorf("failed to parse id: %s", err)
		}
		project, err = gcp.Wrap(client).ProjectByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, project.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyOrganizationName, project.OrganizationName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyProjectID, project.NativeID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyProjectNumber, strconv.FormatInt(project.ProjectNumber, 10)); err != nil {
		return diag.FromErr(err)
	}

	features := &schema.Set{F: schema.HashResource(gcpFeatureResourceWithStatus())}
	for _, feature := range project.Features {
		features.Add(toGCPFeatureResourceWithStatus(feature.Feature, feature.Status))
	}
	if err := d.Set(keyFeature, features); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(project.ID.String())
	return nil
}

func gcpFeatureResourceWithStatus() *schema.Resource {
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
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the feature.",
			},
		},
	}
}

func fromGCPFeatureResource(block map[string]any) core.Feature {
	var pgs []core.PermissionGroup
	for _, pg := range block[keyPermissionGroups].(*schema.Set).List() {
		pgs = append(pgs, core.PermissionGroup(pg.(string)))
	}

	return core.Feature{Name: block[keyName].(string), PermissionGroups: pgs}
}

func toGCPFeatureResourceWithStatus(feature core.Feature, status core.Status) map[string]any {
	pgs := &schema.Set{F: schema.HashString}
	for _, pg := range feature.PermissionGroups {
		pgs.Add(string(pg))
	}

	return map[string]any{
		keyName:             feature.Name,
		keyPermissionGroups: pgs,
		keyStatus:           core.FormatStatus(status),
	}
}
