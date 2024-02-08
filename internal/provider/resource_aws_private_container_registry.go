// Copyright 2023 Rubrik, Inc.
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

func resourceAwsPrivateContainerRegistry() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreatePrivateContainerRegistry,
		ReadContext:   awsReadPrivateContainerRegistry,
		UpdateContext: awsUpdatePrivateContainerRegistry,
		DeleteContext: awsDeletePrivateContainerRegistry,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC account id",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"native_id": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS account ID of the AWS account that will pull images from the RSC container registry.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"url": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "URL for customer provided private container registry.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func awsCreatePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreatePrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Get("account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	nativeID := d.Get("native_id").(string)
	url := d.Get("url").(string)
	if err := aws.Wrap(client).SetPrivateContainerRegistry(ctx, aws.CloudAccountID(id), url, nativeID); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id.String())

	awsReadPrivateContainerRegistry(ctx, d, m)
	return nil
}

// There is no API endpoint to read the state of the private container registry.
func awsReadPrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadPrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	nativeID, url, err := aws.Wrap(client).PrivateContainerRegistry(ctx, aws.CloudAccountID(id))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("native_id", nativeID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("url", url); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func awsUpdatePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsUpdatePrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	nativeID := d.Get("native_id").(string)
	url := d.Get("url").(string)
	if err := aws.Wrap(client).SetPrivateContainerRegistry(ctx, aws.CloudAccountID(id), url, nativeID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// There is no API endpoint to remove the private container registry from the
// account.
func awsDeletePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeletePrivateContainerRegistry")
	return nil
}
