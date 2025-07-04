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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
)

const resourceAzureServicePrincipalDescription = `
The ´polaris_azure_service_principal´ resource adds an Azure service principal to
RSC. A service principal must be added for each Azure tenant before subscriptions
for the tenants can be added to RSC.

There are 3 ways to create a ´polaris_azure_service principal´ resource:
  1. Using the ´app_id´, ´app_name´, ´app_secret´, ´tenant_id´ and ´tenant_domain´
     fields.
  2. Using the ´credentials´ field which is the path to a custom service principal 
     file. A description of the custom format can be found
     [here](https://github.com/rubrikinc/rubrik-polaris-sdk-for-go?tab=readme-ov-file#azure-credentials).
  3. Using the ´sdk_auth´ field which is the path to an Azure service principal
     created with the Azure SDK using the ´--sdk-auth´ parameter.

Prefer to use option 1, as the ´app_name´ and the ´app_secret´ can be updated
without replacing the service principal.

~> **Note:** Removing the last subscription from an RSC tenant will automatically
   remove the tenant, which also removes the service principal. If this happens,
   the service principal can be replaced using
   ´terraform apply -replace=<address-of-service-principal>´.

~> **Note:** Destroying the ´polaris_azure_service_principal´ resource only updates
   the local state, it does not remove the service principal from RSC. However,
   creating another ´polaris_azure_service_principal´ resource for the same Azure
   tenant will overwrite the old service principal in RSC.

-> **Note:** There is no way to verify if a service principal has been added to RSC
   using the UI. RSC tenants don't show up in the UI until the first subscription is
   added.
`

// resourceAzureServicePrincipal defines the schema for the Azure service
// principal resource. Note that the delete function cannot remove the service
// principal since there is no delete operation in the RSC API.
func resourceAzureServicePrincipal() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateServicePrincipal,
		ReadContext:   azureReadServicePrincipal,
		UpdateContext: azureUpdateServicePrincipal,
		DeleteContext: azureDeleteServicePrincipal,

		Description: description(resourceAzureServicePrincipalDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Azure app registration application ID (UUID). Also known as the client ID. " +
					"Note, this might change in the future, use the `app_id` field to reference the application ID " +
					"in configurations.",
			},
			keyAppID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyCredentials, keySDKAuth},
				RequiredWith: []string{keyAppName, keyAppSecret, keyTenantID},
				Description: "Azure app registration application ID. Also known as the client ID. Changing this " +
					"forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyAppName: {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{keyAppID, keyAppSecret, keyTenantID},
				Description:  "Azure app registration display name. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyAppSecret: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				RequiredWith: []string{keyAppID, keyAppName, keyTenantID},
				Description:  "Azure app registration client secret. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyCredentials: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyAppID, keySDKAuth},
				Description: "Path to a custom service principal file. Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: isExistingFile,
			},
			keySDKAuth: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyAppID, keyCredentials},
				Description: "Path to an Azure service principal created with the Azure SDK using the `--sdk-auth` " +
					"parameter. Changing this forces a new resource to be created.",
				ValidateFunc: isExistingFile,
			},
			keyPermissions: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Permissions updated signal. When this field is updated, the provider will notify RSC " +
					"that permissions has been updated. Use this field with the `polaris_azure_permissions` data " +
					"source. **Deprecated:** use the `polaris_azure_subscription` resource's `permissions` fields " +
					"instead.",
				Deprecated:   "use the `polaris_azure_subscription` resource's `permissions` fields instead.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermissionsHash: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Permissions updated signal. **Deprecated:** use `permissions` instead.",
				Deprecated:   "use `permissions` instead.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTenantDomain: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure tenant primary domain. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTenantID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				RequiredWith: []string{keyAppID, keyAppName, keyAppSecret},
				Description: "Azure tenant ID. Also known as the directory ID. Changing this forces a new resource to " +
					"be created.",
				ValidateFunc: validation.IsUUID,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureServicePrincipalV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureServicePrincipalStateUpgradeV0,
			Version: 0,
		}},
	}
}

// azureCreateServicePrincipal run the Create operation for the Azure service
// principal resource. This adds the Azure service principal to the RSC
// platform.
func azureCreateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureCreateServicePrincipal")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	tenantDomain := d.Get(keyTenantDomain).(string)
	var principal azure.ServicePrincipalFunc
	switch {
	case d.Get(keyCredentials).(string) != "":
		principal = azure.KeyFile(d.Get(keyCredentials).(string), tenantDomain)
	case d.Get(keySDKAuth).(string) != "":
		principal = azure.SDKAuthFile(d.Get(keySDKAuth).(string), tenantDomain)
	default:
		appID, err := uuid.Parse(d.Get(keyAppID).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		tenantID, err := uuid.Parse(d.Get(keyTenantID).(string))
		if err != nil {
			return diag.FromErr(err)
		}

		principal = azure.ServicePrincipal(appID, d.Get(keyAppName).(string), d.Get(keyAppSecret).(string), tenantID, tenantDomain)
	}

	appID, err := azure.Wrap(client).SetServicePrincipal(ctx, principal)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(appID.String())
	azureReadServicePrincipal(ctx, d, m)
	return nil
}

// azureReadServicePrincipal run the Read operation for the Azure service
// principal resource. This reads the state of the Azure service principal in
// RSC.
func azureReadServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadServicePrincipal")

	return nil
}

// azureUpdateServiceAccount run the Update operation for the Azure service
// principal resource. This updates the Azure service principal in RSC.
func azureUpdateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureUpdateServicePrincipal")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges(keyAppName, keyAppSecret) {
		id, err := uuid.Parse(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		tenantID, err := uuid.Parse(d.Get(keyTenantID).(string))
		if err != nil {
			return diag.FromErr(err)
		}

		principal := azure.ServicePrincipal(id, d.Get(keyAppName).(string), d.Get(keyAppSecret).(string), tenantID, d.Get(keyTenantDomain).(string))
		if _, err := azure.Wrap(client).SetServicePrincipal(ctx, principal); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges(keyPermissions, keyPermissionsHash) {
		err := azure.Wrap(client).PermissionsUpdatedForTenantDomain(ctx, d.Get(keyTenantDomain).(string), nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadServicePrincipal(ctx, d, m)
	return nil
}

// azureDeleteServicePrincipal run the Delete operation for the Azure service
// principal resource. This only removes the local state of the GCP service
// account since the service account cannot be removed using the RSC API.
func azureDeleteServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteServicePrincipal")

	d.SetId("")
	return nil
}
