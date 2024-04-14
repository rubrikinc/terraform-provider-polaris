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
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceAzurePermissions defines the schema for the Azure permissions data
// source.
func dataSourceAzurePermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: azurePermissionsRead,

		Description: "The `polaris_azure_permissions` data source is used to access information about the " +
			"permissions required by RSC for a specified set of RSC features. The features currently supported for " +
			"Azure subscriptions are:\n" +
			"  * `AZURE_SQL_DB_PROTECTION`\n" +
			"  * `AZURE_SQL_MI_PROTECTION`\n" +
			"  * `CLOUD_NATIVE_ARCHIVAL`\n" +
			"  * `CLOUD_NATIVE_ARCHIVAL_ENCRYPTION`\n" +
			"  * `CLOUD_NATIVE_PROTECTION`\n" +
			"  * `EXOCOMPUTE`\n" +
			"\n" +
			"See the [subscription](azure_subscription) resource for more information on enabling features for an " +
			"Azure subscription added to RSC.\n" +
			"\n" +
			"The `polaris_azure_permissions` data source can be used with the `azurerm_role_definition` and the " +
			"`polaris_azure_service_principal` resources to automatically update the permissions of roles and notify " +
			"RSC about the updated permissions.\n" +
			"\n" +
			"-> **Note:** Due to backward compatibility, the `features` field allow the feature names to be given in " +
			"   3 different styles: `EXAMPLE_FEATURE_NAME`, `example-feature-name` or `example_feature_name`. The " +
			"   recommended style is `EXAMPLE_FEATURE_NAME` as it is what the RSC API itself uses.",
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the required permissions, will be updated as the required permissions " +
					"changes.",
			},
			keyActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed actions.",
			},
			keyDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed data actions.",
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:    1,
				Required:    true,
				Description: "RSC features.",
			},
			keyHash: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the permissions, can be used to detect changes to the permissions. " +
					"Deprecated, use `id` instead.",
				Deprecated: "Use `id` instead.",
			},
			keyNotActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed actions.",
			},
			keyNotDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed data actions.",
			},
		},
	}
}

// azurePermissionsRead run the Read operation for the Azure permissions data
// source. Reads the permissions required for the specified RSC features.
func azurePermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azurePermissionsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var features []core.Feature
	for _, f := range d.Get(keyFeatures).(*schema.Set).List() {
		features = append(features, core.ParseFeatureNoValidation(f.(string)))
	}

	perms, err := azure.Wrap(client).Permissions(ctx, features)
	if err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()

	sort.Strings(perms.Actions)
	var actions []interface{}
	for _, perm := range perms.Actions {
		actions = append(actions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyActions, actions); err != nil {
		return diag.FromErr(err)
	}

	sort.Strings(perms.DataActions)
	var dataActions []interface{}
	for _, perm := range perms.DataActions {
		dataActions = append(dataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyDataActions, dataActions); err != nil {
		return diag.FromErr(err)
	}

	sort.Strings(perms.NotActions)
	var notActions []interface{}
	for _, perm := range perms.NotActions {
		notActions = append(notActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyNotActions, notActions); err != nil {
		return diag.FromErr(err)
	}

	sort.Strings(perms.NotDataActions)
	var notDataActions []interface{}
	for _, perm := range perms.NotDataActions {
		notDataActions = append(notDataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyNotDataActions, notDataActions); err != nil {
		return diag.FromErr(err)
	}

	hashValue := fmt.Sprintf("%x", hash.Sum(nil))
	if err := d.Set(keyHash, hashValue); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(hashValue)
	return nil
}
