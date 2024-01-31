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

func resourceAwsExocomputeClusterAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateClusterAttachment,
		ReadContext:   awsReadClusterAttachment,
		DeleteContext: awsDeleteClusterAttachment,

		Schema: map[string]*schema.Schema{
			"cluster_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "AWS EKS cluster name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"connection_command": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster connection command.",
			},
			"exocompute_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "RSC exocompute id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

func awsCreateClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	configID, err := uuid.Parse(d.Get("exocompute_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName := d.Get("cluster_name").(string)

	// Request cluster attachment.
	clusterID, cmd, err := aws.Wrap(client).AddClusterToExocomputeConfig(ctx, configID, clusterName)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set read-only attributes.
	d.SetId(clusterID.String())
	if err := d.Set("connection_command", cmd); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsReadClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadClusterAttachment")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get attributes.
	configID, err := uuid.Parse(d.Get("exocompute_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName := d.Get("cluster_name").(string)

	// Request cluster attachment. The AddClusterToExocomputeConfig function is
	// idempotent.
	clusterID, cmd, err := aws.Wrap(client).AddClusterToExocomputeConfig(ctx, configID, clusterName)
	if err != nil {
		return diag.FromErr(err)
	}

	if clusterID.String() != d.Id() {
		return diag.Errorf("cluster id mismatch: %s != %s", clusterID.String(), d.Id())
	}

	// Set read-only attributes.
	if err := d.Set("connection_command", cmd); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsDeleteClusterAttachment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteClusterAttachment")

	// There is no way to detach a cluster from an exocompute config at this
	// time.
	d.SetId("")

	return nil
}
