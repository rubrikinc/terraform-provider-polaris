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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceAzurePermissionsDescription = `
The ´polaris_azure_permissions´ data source is used to access information about
the permissions required by RSC for a specified RSC feature. The features currently
supported for Azure subscriptions are:
  * ´AZURE_SQL_DB_PROTECTION´
  * ´AZURE_SQL_MI_PROTECTION´
  * ´CLOUD_NATIVE_ARCHIVAL´
  * ´CLOUD_NATIVE_ARCHIVAL_ENCRYPTION´
  * ´CLOUD_NATIVE_BLOB_PROTECTION´
  * ´CLOUD_NATIVE_PROTECTION´
  * ´EXOCOMPUTE´

See the [subscription](../resources/azure_subscription) resource for more information
on enabling features for an Azure subscription added to RSC.

The ´polaris_azure_permissions´ data source can be used with the ´azurerm_role_definition´
and the ´permissions´ fields of the ´polaris_azure_subscription´ resources to
automatically update the permissions of roles and notify RSC about the updated
permissions.

-> **Note:** To better fit the RSC Azure permission model where each RSC feature have
   two Azure roles, the ´features´ field has been deprecated and replaced with the
   ´feature´ field.

-> **Note:** Due to the RSC Azure permission model having been refined into subscription
   level permissions and resource group level permissions, the ´actions´, ´data_actions´,
   ´not_actions´ and ´not_data_actions´ fields have been deprecated and replaced with the
   corresponding subscription and resource group fields.

-> **Note:** Due to backward compatibility, the ´features´ field allow the feature names
   to be given in 3 different styles: ´EXAMPLE_FEATURE_NAME´, ´example-feature-name´ or
   ´example_feature_name´. The recommended style is ´EXAMPLE_FEATURE_NAME´ as it is what
   the RSC API itself uses.
`

func dataSourceAzurePermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: azurePermissionsRead,

		Description: description(dataSourceAzurePermissionsDescription),
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
				Computed: true,
				Description: "Azure allowed actions. **Deprecated:** use `subscription_actions` and " +
					"`resource_group_actions` instead.",
				Deprecated: "use `subscription_actions` and `resource_group_actions` instead.",
			},
			keyDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "Azure allowed data actions. **Deprecated:** use `subscription_data_actions` and " +
					"`resource_group_data_actions` instead.",
				Deprecated: "use `subscription_data_actions` and `resource_group_data_actions` instead.",
			},
			keyFeature: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyFeature, keyFeatures},
				Description: "RSC feature. Note that the feature name must be given in the `EXAMPLE_FEATURE_NAME` " +
					"style. Possible values are `AZURE_SQL_DB_PROTECTION`, `AZURE_SQL_MI_PROTECTION`,  " +
					"`CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_ARCHIVAL_ENCRYPTION`, `CLOUD_NATIVE_BLOB_PROTECTION`, " +
					"`CLOUD_NATIVE_PROTECTION` and `EXOCOMPUTE`.",
				ValidateFunc: validation.StringInSlice([]string{
					"AZURE_SQL_DB_PROTECTION", "AZURE_SQL_MI_PROTECTION", "CLOUD_NATIVE_ARCHIVAL",
					"CLOUD_NATIVE_ARCHIVAL_ENCRYPTION", "CLOUD_NATIVE_BLOB_PROTECTION", "CLOUD_NATIVE_PROTECTION",
					"EXOCOMPUTE",
				}, false),
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"AZURE_SQL_DB_PROTECTION", "AZURE_SQL_MI_PROTECTION", "CLOUD_NATIVE_ARCHIVAL",
						"CLOUD_NATIVE_ARCHIVAL_ENCRYPTION", "CLOUD_NATIVE_BLOB_PROTECTION", "CLOUD_NATIVE_PROTECTION",
						"EXOCOMPUTE",
					}, false),
				},
				MinItems: 1,
				Optional: true,
				Description: "RSC features. Possible values are `AZURE_SQL_DB_PROTECTION`, " +
					"`AZURE_SQL_MI_PROTECTION`, `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_ARCHIVAL_ENCRYPTION`, " +
					"`CLOUD_NATIVE_BLOB_PROTECTION`, `CLOUD_NATIVE_PROTECTION` and `EXOCOMPUTE`. **Deprecated:** " +
					"use `feature` instead.",
				Deprecated: "use `feature` instead",
			},
			keyHash: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "SHA-256 hash of the permissions, can be used to detect changes to the permissions. " +
					"**Deprecated:** use `id` instead.",
				Deprecated: "use `id` instead.",
			},
			keyNotActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "Azure disallowed actions. **Deprecated:** use `subscription_not_actions` and " +
					"`resource_group_not_actions` instead.",
				Deprecated: "use `subscription_not_actions` and `resource_group_not_actions` instead.",
			},
			keyNotDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "Azure disallowed data actions. **Deprecated:** use `subscription_not_data_actions` and " +
					"`resource_group_not_data_actions` instead.",
				Deprecated: "use `subscription_not_data_actions` and `resource_group_not_data_actions` instead.",
			},
			keyResourceGroupActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed actions on the resource group level.",
			},
			keyResourceGroupDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed data actions on the resource group level.",
			},
			keyResourceGroupNotActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed actions on the resource group level.",
			},
			keyResourceGroupNotDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed data actions on the resource group level.",
			},
			keySubscriptionActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed actions on the subscription level.",
			},
			keySubscriptionDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure allowed data actions on the subscription level.",
			},
			keySubscriptionNotActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed actions on the subscription level.",
			},
			keySubscriptionNotDataActions: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Azure disallowed data actions on the subscription level.",
			},
		},
	}
}

func azurePermissionsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] azurePermissionsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Check both feature and features.
	var perms []azure.Permissions
	var groups []azure.PermissionGroupWithVersion
	if f := d.Get(keyFeature).(string); f != "" {
		perms, groups, err = azure.Wrap(client).ScopedPermissions(ctx, core.Feature{Name: f})
	} else {
		var features []core.Feature
		for _, f := range d.Get(keyFeatures).(*schema.Set).List() {
			features = append(features, core.Feature{Name: f.(string)})
		}
		perms, err = azure.Wrap(client).ScopedPermissionsForFeatures(ctx, features)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()

	// Legacy scope. The legacy scope contains the union of the subscription
	// and the resource group scopes, so we only need to update the hash value
	// here, with the added benefit of keeping it backwards compatible.
	var actions []any
	for _, perm := range perms[azure.ScopeLegacy].Actions {
		actions = append(actions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyActions, actions); err != nil {
		return diag.FromErr(err)
	}

	var dataActions []any
	for _, perm := range perms[azure.ScopeLegacy].DataActions {
		dataActions = append(dataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyDataActions, dataActions); err != nil {
		return diag.FromErr(err)
	}

	var notActions []any
	for _, perm := range perms[azure.ScopeLegacy].NotActions {
		notActions = append(notActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyNotActions, notActions); err != nil {
		return diag.FromErr(err)
	}

	var notDataActions []any
	for _, perm := range perms[azure.ScopeLegacy].NotDataActions {
		notDataActions = append(notDataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set(keyNotDataActions, notDataActions); err != nil {
		return diag.FromErr(err)
	}

	// Subscription scope.
	var subActions []any
	for _, perm := range perms[azure.ScopeSubscription].Actions {
		subActions = append(subActions, perm)
	}
	if err := d.Set(keySubscriptionActions, subActions); err != nil {
		return diag.FromErr(err)
	}

	var subDataActions []any
	for _, perm := range perms[azure.ScopeSubscription].DataActions {
		subDataActions = append(subDataActions, perm)
	}
	if err := d.Set(keySubscriptionDataActions, subDataActions); err != nil {
		return diag.FromErr(err)
	}

	var subNotActions []any
	for _, perm := range perms[azure.ScopeSubscription].NotActions {
		subNotActions = append(subNotActions, perm)
	}
	if err := d.Set(keySubscriptionNotActions, subNotActions); err != nil {
		return diag.FromErr(err)
	}

	var subNotDataActions []any
	for _, perm := range perms[azure.ScopeSubscription].NotDataActions {
		subNotDataActions = append(subNotDataActions, perm)
	}
	if err := d.Set(keySubscriptionNotDataActions, subNotDataActions); err != nil {
		return diag.FromErr(err)
	}

	// Resource group scope.
	var rgActions []any
	for _, perm := range perms[azure.ScopeResourceGroup].Actions {
		rgActions = append(rgActions, perm)
	}
	if err := d.Set(keyResourceGroupActions, rgActions); err != nil {
		return diag.FromErr(err)
	}

	var rgDataActions []any
	for _, perm := range perms[azure.ScopeResourceGroup].DataActions {
		rgDataActions = append(rgDataActions, perm)
	}
	if err := d.Set(keyResourceGroupDataActions, rgDataActions); err != nil {
		return diag.FromErr(err)
	}

	var rgNotActions []any
	for _, perm := range perms[azure.ScopeResourceGroup].NotActions {
		rgNotActions = append(rgNotActions, perm)
	}
	if err := d.Set(keyResourceGroupNotActions, rgNotActions); err != nil {
		return diag.FromErr(err)
	}

	var rgNotDataActions []any
	for _, perm := range perms[azure.ScopeResourceGroup].NotDataActions {
		rgNotDataActions = append(rgNotDataActions, perm)
	}
	if err := d.Set(keyResourceGroupNotDataActions, rgNotDataActions); err != nil {
		return diag.FromErr(err)
	}

	// Hash permission groups. This generates a diff for subscription onboarded
	// with the old onboarding workflow. Applying the diff fixes the backend
	// state.
	for _, group := range groups {
		hash.Write([]byte(group.Name))
		hash.Write([]byte(fmt.Sprintf("%d", group.Version)))
	}

	hashValue := fmt.Sprintf("%x", hash.Sum(nil))
	if err := d.Set(keyHash, hashValue); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(hashValue)
	return nil
}
