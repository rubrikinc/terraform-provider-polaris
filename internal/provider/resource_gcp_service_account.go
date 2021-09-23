package provider

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
)

// resourceGcpServiceAccount defines the schema for the GCP service account
// resource. Note that the delete function cannot remove the service account
// since there is no delete operation in the Polaris API.
func resourceGcpServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateServiceAccount,
		ReadContext:   gcpReadServiceAccount,
		UpdateContext: gcpUpdateServiceAccount,
		DeleteContext: gcpDeleteServiceAccount,

		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: fileExists,
				Description:      "Path to GCP service account key file.",
			},
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Service account name in Polaris. If not given the name of the service account key file is used.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"permissions_hash": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Signals that the permissions has been updated.",
				ValidateDiagFunc: validateHash,
			},
		},
	}
}

// gcpCreateServiceAccount run the Create operation for the GCP service account
// resource. This adds the GCP service account to the Polaris platform.
func gcpCreateServiceAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpCreateServiceAccount")

	client := m.(*polaris.Client)

	credentials := d.Get("credentials").(string)

	// Derive name from credentials filename if missing.
	name := d.Get("name").(string)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(credentials), filepath.Ext(credentials))
	}

	err := client.GCP().SetServiceAccount(ctx, gcp.KeyFile(credentials), gcp.Name(name))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)

	gcpReadServiceAccount(ctx, d, m)
	return nil
}

// gcpReadServiceAccount run the Read operation for the GCP service account
// resource. This reads the state of the GCP service account in Polaris.
func gcpReadServiceAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpReadServiceAccount")

	client := m.(*polaris.Client)

	name, err := client.GCP().ServiceAccount(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("name", name)

	return nil
}

// gcpUpdateServiceAccount run the Update operation for the GCP service account
// resource. This updates the service account in Polaris.
func gcpUpdateServiceAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpUpdateServiceAccount")

	client := m.(*polaris.Client)

	credentials := d.Get("credentials").(string)

	if d.HasChange("name") || d.HasChange("credentials") {
		name := d.Get("name").(string)
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(credentials), filepath.Ext(credentials))
		}

		err := client.GCP().SetServiceAccount(ctx, gcp.KeyFile(credentials), gcp.Name(name))
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("name", name)
		d.SetId(name)
	}

	if d.HasChange("permissions_hash") {
		err := client.GCP().PermissionsUpdated(ctx, gcp.KeyFile(credentials), nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// gcpDeleteServiceAccount run the Delete operation for the GCP service account
// resource. This only removes the local state of the GCP service account since
// the service account cannot be removed using the Polaris API.
func gcpDeleteServiceAccount(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpDeleteServiceAccount")

	d.SetId("")
	return nil
}
