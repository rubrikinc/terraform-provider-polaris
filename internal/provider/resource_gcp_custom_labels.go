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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqltags "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/tags"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/tags"
)

var resourceGCPCustomLabelsDescription = `
The ´polaris_gcp_custom_labels´ resource manages RSC custom GCP labels.
Simplify your cloud resource management by assigning custom labels for easy
identification. These custom labels will be used on all existing and future GCP
projects in your RSC account.

-> **Note:** The newly updated custom labels will be applied to all existing and
   new resources, while the previously applied labels will remain unchanged.

~> **Warning:** When using multiple ´polaris_gcp_custom_labels´ resources in the
   same RSC account, there is a risk of a race condition when the resources are
   destroyed. This can result in custom labels remaining in RSC even after all
   ´polaris_gcp_custom_labels´ resources have been destroyed. The race condition
   can be avoided by either managing all custom labels using a single
   ´polaris_gcp_custom_labels´ resource or by using ´depends_on´ to ensure that
   the resources are destroyed in a serial fashion.

~> **Warning:** The ´override_resource_labels´ field refers to a single global
   value in RSC. So multiple ´polaris_gcp_custom_labels´ resources with
   different values for the ´override_resource_labels´ field will result in a
   perpetual diff.
`

const gcpCustomLabelsID = "31e3cbd5c7bd25c4de00fdd6635f2d0bf237930e0d6a4e6b1bbf8a4fcccc6c4c"

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceGcpCustomLabels() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateCustomLabels,
		ReadContext:   gcpReadCustomLabels,
		UpdateContext: gcpUpdateCustomLabels,
		DeleteContext: gcpDeleteCustomLabels,

		Description: description(resourceGCPCustomLabelsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the string \"GCP\".",
			},
			keyCustomLabels: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Custom labels to add to cloud resources.",
			},
			keyOverrideResourceLabels: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Should custom labels overwrite existing labels with the same keys. Default value is `true`.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: gcpImportCustomLabels,
		},
	}
}

func gcpCreateCustomLabels(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateCustomLabels")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var customTags []core.Tag
	for key, value := range d.Get(keyCustomLabels).(map[string]any) {
		customTags = append(customTags, core.Tag{Key: key, Value: value.(string)})
	}

	if err := tags.Wrap(client).AddCustomerTags(ctx, gqltags.CustomerTags{
		CloudVendor:          core.CloudVendorGCP,
		Tags:                 customTags,
		OverrideResourceTags: d.Get(keyOverrideResourceLabels).(bool),
	}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(gcpCustomLabelsID)
	return nil
}

func gcpReadCustomLabels(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadCustomLabels")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	customerTags, err := tags.Wrap(client).CustomerTags(ctx, core.CloudVendorGCP)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := setCustomTags(d, keyCustomLabels, customerTags.Tags); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyOverrideResourceLabels, customerTags.OverrideResourceTags); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateCustomLabels(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateCustomLabels")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	customerTags, err := tags.Wrap(client).CustomerTags(ctx, core.CloudVendorGCP)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a set holding the keys of the labels being removed.
	oldTags, newTags := d.GetChange(keyCustomLabels)
	removeSet := make(map[string]struct{}, len(oldTags.(map[string]any)))
	for k := range oldTags.(map[string]any) {
		removeSet[k] = struct{}{}
	}
	for k := range newTags.(map[string]any) {
		delete(removeSet, k)
	}

	// Merge customer labels in RSC with custom labels defined in the resource
	// data, ignoring the labels being removed. Values of custom labels defined
	// in the resource data takes precedence.
	mergeSet := make(map[string]string, len(customerTags.Tags)+len(newTags.(map[string]any)))
	for _, tag := range customerTags.Tags {
		if _, ok := removeSet[tag.Key]; !ok {
			mergeSet[tag.Key] = tag.Value
		}
	}
	for k, v := range newTags.(map[string]any) {
		mergeSet[k] = v.(string)
	}

	customerTags.Tags = make([]core.Tag, 0, len(mergeSet))
	for k, v := range mergeSet {
		customerTags.Tags = append(customerTags.Tags, core.Tag{Key: k, Value: v})
	}
	customerTags.OverrideResourceTags = d.Get(keyOverrideResourceLabels).(bool)
	if err := tags.Wrap(client).ReplaceCustomerTags(ctx, customerTags); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpDeleteCustomLabels(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteCustomLabels")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var customTagKeys []string
	for key := range d.Get(keyCustomLabels).(map[string]any) {
		customTagKeys = append(customTagKeys, key)
	}

	if err := tags.Wrap(client).RemoveCustomerTags(ctx, core.CloudVendorGCP, customTagKeys); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Note, the custom labels resource is designed to only manage custom labels
// owned by the resource. An import on the other hand will take ownership of all
// custom labels for a cloud vendor.
func gcpImportCustomLabels(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	tflog.Trace(ctx, "gcpImportCustomLabels")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	customerTags, err := tags.Wrap(client).CustomerTags(ctx, core.CloudVendorGCP)
	if err != nil {
		return nil, err
	}
	if err := importCustomTags(d, keyCustomLabels, customerTags.Tags); err != nil {
		return nil, err
	}

	d.SetId(gcpCustomLabelsID)
	return []*schema.ResourceData{d}, nil
}
