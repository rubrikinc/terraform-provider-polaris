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
				ForceNew:         true,
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

	err := gcp.NewAPI(client.GQL).SetServiceAccount(ctx, gcp.KeyFile(credentials), gcp.Name(name))
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

	name, err := gcp.NewAPI(client.GQL).ServiceAccount(ctx)
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

	if d.HasChange("name") {
		d.Set("name", d.Get("name").(string))
	}

	if d.HasChange("permissions_hash") {
		err := gcp.NewAPI(client.GQL).PermissionsUpdatedForDefault(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gcpReadServiceAccount(ctx, d, m)
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
