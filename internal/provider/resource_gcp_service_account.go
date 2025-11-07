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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
)

const resourceGCPServiceAccountDescription = `
The ´polaris_gcp_service_account´ resource adds the GCP service account to RSC
as the default service account. The default service account will be used by RSC
to authenticate to the GCP for projects added to RSC without a service account.

~> **Note:** Changing the name of the default service account can take a
   considerable time to propagate through the system. Use the ´ignore_changes´
   field of the ´lifecycle´ block if it becomes an issue.

~> **Note:** Destroying the ´polaris_gcp_service_account´ resource only updates
   the local state, it does not remove the service account from RSC. However,
   it's possible to overwrite the RSC global service account with new service
   accounts.

-> **Note:** There is no way to verify if an default GCP service account has
   been added to RSC using the UI.
`

func resourceGcpServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateServiceAccount,
		ReadContext:   gcpReadServiceAccount,
		UpdateContext: gcpUpdateServiceAccount,
		DeleteContext: gcpDeleteServiceAccount,

		Description: description(resourceGCPServiceAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the  service account name.",
			},
			keyCredentials: {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
				Description: "Base64 encoded GCP service account private key or path to GCP service account key " +
					"file.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Service account name in RSC. Defaults to `service-account-<timestamp>`.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermissionsHash: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Signals that the permissions has been updated. **Deprecated:** use the " +
					"`permissions` field of the `feature` block of the `polaris_gcp_project` resource instead.",
				Deprecated: "Use the `permissions` field of the `feature` block of `polaris_gcp_project` " +
					"instead.",
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceGcpServiceAccountV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpServiceAccountStateUpgradeV0,
			Version: 0,
		}},
	}
}

func gcpCreateServiceAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateServiceAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	credentials := d.Get(keyCredentials).(string)
	name := d.Get(keyName).(string)
	if name == "" {
		name = fmt.Sprintf("service-account-%d", time.Now().UnixMicro())
	}

	if err := gcp.Wrap(client).SetServiceAccount(ctx, gcp.Key(credentials), gcp.Name(name)); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	hash.Write([]byte(name))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))
	gcpReadServiceAccount(ctx, d, m)
	return nil
}

func gcpReadServiceAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadServiceAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	name, err := gcp.Wrap(client).ServiceAccount(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", name); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func gcpUpdateServiceAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateServiceAccount")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges(keyCredentials, keyName) {
		credentials := d.Get(keyCredentials).(string)
		name := d.Get(keyName).(string)

		if err := gcp.Wrap(client).SetServiceAccount(ctx, gcp.Key(credentials), gcp.Name(name)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(keyPermissionsHash) {
		err := gcp.Wrap(client).PermissionsUpdatedForDefault(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gcpReadServiceAccount(ctx, d, m)
	return nil
}

// This function only removes the local state of the RSC global GCP service
// account since the service account cannot be removed using the Polaris API.
func gcpDeleteServiceAccount(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteServiceAccount")

	d.SetId("")
	return nil
}
