// Copyright 2024 Rubrik, Inc.
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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/exocompute"
)

const azureExocomputeClusterAttachmentDescription = `
The ´polaris_azure_exocompute_cluster_attachment´ resource attaches an Azure AKS
cluster to a customer managed host Exocompute configuration, allowing RSC to use
the cluster for Exocompute operations.

The cluster name must be specified as ´<resource-group>/<cluster-name>´, e.g.
´my-resource-group/my-cluster´.
`

func resourceAzureExocomputeClusterAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateAwsExocomputeClusterAttachment,
		ReadContext:   azureReadAwsExocomputeClusterAttachment,
		UpdateContext: azureUpdateAwsExocomputeClusterAttachment,
		DeleteContext: azureDeleteAwsExocomputeClusterAttachment,

		Description: description(azureExocomputeClusterAttachmentDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cluster ID (UUID).",
			},
			keyClusterName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Azure AKS cluster name. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyExocomputeID: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "RSC exocompute configuration ID (UUID). Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: validation.IsUUID,
			},
			keyManifest: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Kubernetes manifest which can be passed to `kubectl apply` to create a connection " +
					"between the cluster and RSC. See `" + keyConnectionCommand + "` for an alternative connection " +
					"method.",
			},
			keyTokenRefresh: {
				Type:     schema.TypeInt,
				Optional: true,
				Description: "To force a refresh of the authentication token, part of the connection command and " +
					"manifest, increase the value of this field. The token is valid for 24 hours.",
			},
		},
	}
}

func azureCreateAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureCreateAwsExocomputeClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	configID, err := uuid.Parse(d.Get(keyExocomputeID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName := d.Get(keyClusterName).(string)

	clusterID, info, err := exocompute.Wrap(client).ConnectAzureCluster(ctx, clusterName, configID)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyManifest, info.Manifest); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterID.String())
	return nil
}

func azureReadAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadAwsExocomputeClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	configID, err := uuid.Parse(d.Get(keyExocomputeID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName := d.Get(keyClusterName).(string)

	info, err := exocompute.Wrap(client).AzureClusterConnection(ctx, clusterName, configID)
	if err != nil {
		tflog.Warn(ctx, "failed to read cluster attachment", map[string]any{"err": err.Error()})
		return nil
	}
	if err := d.Set(keyManifest, info.Manifest); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureUpdateAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureUpdateAwsExocomputeClusterAttachment")

	if d.HasChange(keyTokenRefresh) {
		return awsCreateAwsExocomputeClusterAttachment(ctx, d, m)
	}

	return nil
}

func azureDeleteAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteAwsExocomputeClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := exocompute.Wrap(client).DisconnectAzureCluster(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
