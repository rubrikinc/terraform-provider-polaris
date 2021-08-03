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
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				AtLeastOneOf:     []string{"credentials", "app_id"},
				Description:      "Path to Azure service principal file.",
				ValidateDiagFunc: credentialsFileExists,
			},
			"app_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_name", "app_secret", "tenant_id"},
				Description:      "App registration application id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"app_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_secret", "tenant_id"},
				Description:      "App registration display name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"app_secret": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "tenant_id"},
				Description:      "App registration client secret.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
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
				Description:      "Tenant directory/domain name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "app_secret"},
				Description:      "Tenant/domain id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
		},
	}
}

// azureCreateServicePrincipal run the Create operation for the Azure service
// principal resource. This adds the Azure service principal to the Polaris
// platform.
func azureCreateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateServicePrincipal")

	client := m.(*polaris.Client)

	var principal azure.ServicePrincipalFunc
	if credentials := d.Get("credentials").(string); credentials != "" {
		principal = azure.SDKAuthFile(credentials, d.Get("tenant_domain").(string))
	} else {
		appID, err := uuid.Parse(d.Get("app_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}

		tenantID, err := uuid.Parse(d.Get("tenant_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}

		principal = azure.ServicePrincipal(appID, d.Get("app_secret").(string), tenantID, d.Get("tenant_domain").(string))
	}

	// Set service principal in Polaris.
	id, err := client.Azure().SetServicePrincipal(ctx, principal)
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
		err := client.Azure().PermissionsUpdated(ctx)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	var principal azure.ServicePrincipalFunc
	if d.HasChange("credentials") || d.HasChange("tenant_domain") {
		if credentials := d.Get("credentials").(string); credentials != "" {
			principal = azure.SDKAuthFile(credentials, d.Get("tenant_domain").(string))
		}
	}

	if d.HasChange("app_id") || d.HasChange("app_secret") || d.HasChange("tenant_id") || d.HasChange("tenant_domain") {
		if id := d.Get("app_id").(string); id != "" {
			appID, err := uuid.Parse(id)
			if err != nil {
				return diag.FromErr(err)
			}

			tenantID, err := uuid.Parse(d.Get("tenant_id").(string))
			if err != nil {
				return diag.FromErr(err)
			}

			principal = azure.ServicePrincipal(appID, d.Get("app_secret").(string), tenantID, d.Get("tenant_domain").(string))
		}
	}

	// Set service principal in Polaris.
	id, err := client.Azure().SetServicePrincipal(ctx, principal)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())

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
