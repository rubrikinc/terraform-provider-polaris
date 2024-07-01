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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
)

const awsExocomputeClusterAttachmentDescription = `
The ´polaris_aws_exocompute_cluster_attachment´ resource attaches an AWS EKS cluster
to a customer managed host Exocompute configuration, allowing RSC to use the cluster
for Exocompute operations.
`

func resourceAwsExocomputeClusterAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateAwsExocomputeClusterAttachment,
		ReadContext:   awsReadAwsExocomputeClusterAttachment,
		UpdateContext: awsUpdateAwsExocomputeClusterAttachment,
		DeleteContext: awsDeleteAwsExocomputeClusterAttachment,

		Description: description(awsExocomputeClusterAttachmentDescription),
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
				Description:  "AWS EKS cluster name. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyConnectionCommand: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "`kubectl` command which can be executed inside the EKS cluster to create a connection " +
					"between the cluster and RSC. See " + keySetupYAML + " for an alternative connection method.",
			},
			keyExocomputeID: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "RSC exocompute configuration ID (UUID). Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: validation.IsUUID,
			},
			keySetupYAML: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "K8s spec which can be passed to `kubectl apply -f` inside the EKS cluster to create a " +
					"connection between the cluster and RSC. See " + keyConnectionCommand + " for an alternative " +
					"connection method.",
			},
			keyTokenRefresh: {
				Type:     schema.TypeInt,
				Optional: true,
				Description: "To force a refresh of the token, part of the connection command, increase the value " +
					"of this field. The token is valid for 24 hours.",
			},
		},
	}
}

func awsCreateAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateAwsExocomputeClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	configID, err := uuid.Parse(d.Get(keyExocomputeID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName := d.Get(keyClusterName).(string)

	clusterID, kubectlCmd, setupYAML, err := aws.Wrap(client).AddClusterToExocomputeConfig(ctx, configID, clusterName)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyConnectionCommand, kubectlCmd); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySetupYAML, setupYAML); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterID.String())
	return nil
}

func awsReadAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadAwsExocomputeClusterAttachment")

	// There is no way to read the state of the cluster attachment without
	// updating the token.
	return nil
}

func awsUpdateAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdateAwsExocomputeClusterAttachment")

	if d.HasChange(keyTokenRefresh) {
		return awsCreateAwsExocomputeClusterAttachment(ctx, d, m)
	}

	return nil
}

func awsDeleteAwsExocomputeClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteAwsExocomputeClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := aws.Wrap(client).RemoveExocomputeCluster(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
