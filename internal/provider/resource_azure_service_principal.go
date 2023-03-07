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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
)

// resourceAzureServicePrincipal defines the schema for the Azure service
// principal resource. Note that the delete function cannot remove the service
// principal since there is no delete operation in the Polaris API.
func resourceAzureServicePrincipal() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateServicePrincipal,
		ReadContext:   azureReadServicePrincipal,
		UpdateContext: azureUpdateServicePrincipal,
		DeleteContext: azureDeleteServicePrincipal,

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ExactlyOneOf:     []string{"app_id", "credentials", "sdk_auth"},
				ConflictsWith:    []string{"credentials", "sdk_auth"},
				RequiredWith:     []string{"app_name", "app_secret", "tenant_id"},
				Description:      "App registration application id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"app_name": {
				Type:             schema.TypeString,
				Optional:         true,
				RequiredWith:     []string{"app_id", "app_secret", "tenant_id"},
				Description:      "App registration display name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"app_secret": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				RequiredWith:     []string{"app_id", "app_name", "tenant_id"},
				Description:      "App registration client secret.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ConflictsWith:    []string{"app_id", "sdk_auth"},
				Description:      "Path to Azure service principal file.",
				ValidateDiagFunc: fileExists,
			},
			"sdk_auth": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ConflictsWith:    []string{"app_id", "credentials"},
				Description:      "Path to Azure service principal created with the Azure SDK using the --sdk-auth parameter",
				ValidateDiagFunc: fileExists,
			},
			"permissions_hash": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Signals that the permissions has been updated.",
				ValidateDiagFunc: validateHash,
			},
			"tenant_domain": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Tenant directory/domain name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				RequiredWith:     []string{"app_id", "app_name", "app_secret"},
				Description:      "Tenant/domain id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
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
// principal resource. This adds the Azure service principal to the Polaris
// platform.
func azureCreateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateServicePrincipal")

	client := m.(*polaris.Client)
	tenantDomain := d.Get("tenant_domain").(string)

	var principal azure.ServicePrincipalFunc
	switch {
	case d.Get("credentials").(string) != "":
		principal = azure.KeyFile(d.Get("credentials").(string), tenantDomain)
	case d.Get("sdk_auth").(string) != "":
		principal = azure.SDKAuthFile(d.Get("sdk_auth").(string), tenantDomain)
	default:
		appID, err := uuid.Parse(d.Get("app_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}
		tenantID, err := uuid.Parse(d.Get("tenant_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}

		principal = azure.ServicePrincipal(appID, d.Get("app_secret").(string), tenantID, tenantDomain)
	}

	id, err := azure.Wrap(client).SetServicePrincipal(ctx, principal)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())

	azureReadServicePrincipal(ctx, d, m)
	return nil
}

// azureReadServicePrincipal run the Read operation for the Azure service
// principal resource. This reads the state of the Azure service principal in
// Polaris.
func azureReadServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadServicePrincipal")

	return nil
}

// azureUpdateServiceAccount run the Update operation for the Azure service
// principal resource. This updates the Azure service principal in Polaris.
func azureUpdateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateServicePrincipal")

	client := m.(*polaris.Client)

	if d.HasChange("permissions_hash") {
		err := azure.Wrap(client).PermissionsUpdatedForTenantDomain(ctx, d.Get("tenant_domain").(string), nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	azureReadServicePrincipal(ctx, d, m)
	return nil
}

// azureDeleteServicePrincipal run the Delete operation for the Azure service
// principal resource. This only removes the local state of the GCP service
// account since the service account cannot be removed using the Polaris API.
func azureDeleteServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteServicePrincipal")

	d.SetId("")
	return nil
}
