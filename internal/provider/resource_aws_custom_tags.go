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

var resourceAWSCustomTagsDescription = `
The ´polaris_aws_custom_tags´ resource manages RSC custom AWS tags. Simplify
your cloud resource management by assigning custom tags for easy identification.
These custom tags will be used on all existing and future AWS accounts in your
cloud account.

-> **Note:** The newly updated custom tags will be applied to all existing and
   new resources, while the previously applied tags will remain unchanged.

~> **Warning:** When using multiple ´polaris_aws_custom_tags´ resources in the
   same RSC account, there is a risk of a race condition when the resources are
   destroyed. This can result in custom tags remaining in RSC even after all
   ´polaris_aws_custom_tags´ resources have been destroyed. The race condition
   can be avoided by either managing all custom tags using a single
   ´polaris_aws_custom_tags´ resource or by using the ´depends_on´ field to
   ensure that the resources are destroyed in a serial fashion.

~> **Warning:** The ´override_resource_tags´ field refers to a single global
   value in RSC. So multiple ´polaris_aws_custom_tags´ resources with different
   values for the ´override_resource_tags´ field will result in a perpetual
   diff.
`

func resourceAwsCustomTags() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCustomTags,
		ReadContext:   awsReadCustomTags,
		UpdateContext: awsUpdateCustomTags,
		DeleteContext: awsDeleteCustomTags,

		Description: description(resourceAWSCustomTagsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the string \"AWS\".",
			},
			keyCustomTags: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Custom tags to add to cloud resources.",
			},
			keyOverrideResourceTags: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Should custom tags overwrite existing tags with the same keys. Default value is `true`.",
			},
		},
	}
}

func awsCreateCustomTags(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateCustomTags")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var customTags []core.Tag
	for key, value := range d.Get(keyCustomTags).(map[string]any) {
		customTags = append(customTags, core.Tag{Key: key, Value: value.(string)})
	}

	if err := tags.Wrap(client).AddCustomerTags(ctx, gqltags.CustomerTags{
		CloudVendor:          core.CloudVendorAWS,
		Tags:                 customTags,
		OverrideResourceTags: d.Get(keyOverrideResourceTags).(bool),
	}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("32fd72a0e0746043a1cce59f2e840490df6b9ea49e9bbcade136da5e8173d6c0")
	return nil
}

func awsReadCustomTags(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCustomTags")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	customerTags, err := tags.Wrap(client).CustomerTags(ctx, core.CloudVendorAWS)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := setCustomTags(d, customerTags.Tags); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyOverrideResourceTags, customerTags.OverrideResourceTags); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdateCustomTags(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCustomTags")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	customerTags, err := tags.Wrap(client).CustomerTags(ctx, core.CloudVendorAWS)
	if err != nil {
		return diag.FromErr(err)
	}

	// Merge customer tags in RSC with custom tags defined in the resource data.
	// Values of custom tags defined in the resource data takes precedence.
	cts := d.Get(keyCustomTags).(map[string]any)
	set := make(map[string]string, len(customerTags.Tags)+len(cts))
	for _, tag := range customerTags.Tags {
		set[tag.Key] = tag.Value
	}
	for k, v := range cts {
		set[k] = v.(string)
	}
	customerTags.Tags = make([]core.Tag, 0, len(set))
	for k, v := range set {
		customerTags.Tags = append(customerTags.Tags, core.Tag{Key: k, Value: v})
	}
	customerTags.OverrideResourceTags = d.Get(keyOverrideResourceTags).(bool)

	if err := tags.Wrap(client).ReplaceCustomerTags(ctx, customerTags); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsDeleteCustomTags(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCustomTags")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var customTagKeys []string
	for key := range d.Get(keyCustomTags).(map[string]any) {
		customTagKeys = append(customTagKeys, key)
	}

	if err := tags.Wrap(client).RemoveCustomerTags(ctx, core.CloudVendorAWS, customTagKeys); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// setCustomTags sets the custom tags in the resource data.
func setCustomTags(d *schema.ResourceData, customTags []core.Tag) error {
	cts := d.Get(keyCustomTags).(map[string]any)

	// Create a set holding the custom tag keys specified in the Terraform
	// configuration.
	set := make(map[string]struct{}, len(cts))
	for key := range cts {
		set[key] = struct{}{}
	}

	// Remove all tag keys in the customTags slice from the set. Afterward, the
	// set will contain TF configuration keys missing from the customTags slice.
	for _, tag := range customTags {
		delete(set, tag.Key)
	}

	// Remove the missing keys from the Terraform configuration.
	for key := range set {
		delete(cts, key)
	}

	// Update the Terraform configuration with values from the customTags slice.
	for _, tag := range customTags {
		if _, ok := cts[tag.Key]; ok {
			cts[tag.Key] = tag.Value
		}
	}

	if err := d.Set(keyCustomTags, cts); err != nil {
		return err
	}

	return nil
}
