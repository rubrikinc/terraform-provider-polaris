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

const awsPrivateContainerRegistryDescription = `
The ´polaris_aws_private_container_registry´ resource enables the private container
registry (PCR) feature for the RSC customer account. This disables the standard
Rubrik container registry. Once PCR has been enabled, it can only be disabled by
Rubrik customer support.

!> **Note:** Creating a ´polaris_aws_private_container_registry´ resource enables
   the PCR feature for the RSC customer account. Destroying the resource will not
   disabled PCR, it can only be disabled by contacting Rubrik customer support.

~> **Note:** Even though the ´polaris_aws_private_container_registry´ resource ID
   is an RSC cloud account ID, there can only be a single PCR per RSC customer
   account.

## Exocompute Image Bundles
The following GraphQL query can be used to retrieve information about the image
bundles used by RSC for exocompute:
´´´graphql
query ExotaskImageBundle($input: GetExotaskImageBundleInput) {
  exotaskImageBundle(input: $input) {
    bundleImages {
      name
      sha
      tag
    }
    bundleVersion
    eksVersion
    repoUrl
  }
}
´´´
The ´repoUrl´ field holds the URL to the RSC container registry from where the RSC
images can be pulled.

The input is an object with the following structure:
´´´json
{
  "input": {
    "eksVersion": "1.29"
  }
}
´´´
Where ´eksVersion´ is the version of the customer's' EKS cluster. ´eksVersion´ is
optional, if it's not specified it defaults to the latest EKS version supported by
RSC.

The following GraphQL mutation can be used to set the approved bundle version for
the RSC customer account:
´´´graphql
mutation SetBundleApprovalStatus($input: SetBundleApprovalStatusInput!) {
  setBundleApprovalStatus(input: $input)
}
´´´
The input is an object with the following structure:
´´´json
{
  "input": {
    "approvalStatus": "APPROVED",
    "bundleVersion": "1.164"
    "bundleMetadata": {
      "eksVersion": "1.29"
    },
  }
}
´´´
Where ´approvalStatus´ can be either ´APPROVED´ or ´REJECTED´. ´bundleVersion´ is
the the bundle version being approved or rejected. ´bundleMetadata´ is optional.
`

func resourceAwsPrivateContainerRegistry() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreatePrivateContainerRegistry,
		ReadContext:   awsReadPrivateContainerRegistry,
		UpdateContext: awsUpdatePrivateContainerRegistry,
		DeleteContext: awsDeletePrivateContainerRegistry,

		Description: description(awsPrivateContainerRegistryDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyNativeID: {
				Type:     schema.TypeString,
				Required: true,
				Description: "AWS account ID of the AWS account that will pull images from the RSC container " +
					"registry.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyURL: {
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

	id, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	nativeID := d.Get(keyNativeID).(string)
	url := d.Get(keyURL).(string)
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

	if err := d.Set(keyNativeID, nativeID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyURL, url); err != nil {
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
	nativeID := d.Get(keyNativeID).(string)
	url := d.Get(keyURL).(string)
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
