package provider

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
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
				RequiredWith:     []string{"app_name", "app_secret", "tenant_domain", "tenant_id"},
				Description:      "App registration application id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"app_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_secret", "tenant_domain", "tenant_id"},
				Description:      "App registration display name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"app_secret": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "tenant_domain", "tenant_id"},
				Description:      "App registration client secret.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_domain": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "app_secret", "tenant_id"},
				Description:      "Tenant directory/domain name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "app_secret", "tenant_domain"},
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

	return azureUpdateServicePrincipal(ctx, d, m)
}

// azureReadServicePrincipal run the Read operation for the Azure service
// principal resource. This reads the state of the Azure service principal in
// Polaris.
func azureReadServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadServicePrincipal")
	return nil
}

// gcpUpdateServiceAccount run the Update operation for the GCP service account
// resource. This updates the Azure service principal in Polaris.
func azureUpdateServicePrincipal(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdateServicePrincipal")

	client := m.(*polaris.Client)

	var principal polaris.AzureServicePrincipal
	if credentials := d.Get("credentials").(string); credentials != "" {
		var err error
		principal, err = polaris.AzureServicePrincipalFromFile(credentials)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		appID, err := uuid.Parse(d.Get("app_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}

		tenantID, err := uuid.Parse(d.Get("tenant_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}

		principal = polaris.AzureServicePrincipal{
			Cloud:        graphql.AzurePublic,
			AppID:        appID,
			AppName:      d.Get("app_name").(string),
			AppSecret:    d.Get("app_secret").(string),
			TenantID:     tenantID,
			TenantDomain: d.Get("tenant_domain").(string),
		}
	}

	// Set service principal in Polaris.
	if err := client.AzureServicePrincipalSet(ctx, principal); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(principal.AppID.String())

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
