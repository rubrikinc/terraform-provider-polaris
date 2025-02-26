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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/pcr"
)

const azurePrivateContainerRegistryDescription = `
The ´polaris_azure_private_container_registry´ resource enables the private
container registry (PCR) feature for the RSC customer account. This disables the
standard Rubrik container registry.

~> **Note:** Even though the ´polaris_azure_private_container_registry´ resource
   ID is an RSC cloud account ID, there can only be a single PCR per RSC
   customer account.

## Exocompute Image Bundles
The following GraphQL query can be used to retrieve information about the image
bundles used by RSC for exocompute:
´´´graphql
query ExotaskImageBundle {
  exotaskImageBundle {
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
The ´repoUrl´ field holds the URL to the RSC container registry from where the
RSC images can be pulled.

The following GraphQL mutation can be used to set the approved bundle version
for the RSC customer account:
´´´graphql
mutation SetBundleApprovalStatus($input: SetBundleApprovalStatusInput!) {
  setBundleApprovalStatus(input: $input)
}
´´´
The input is an object with the following structure:
´´´json
{
  "input": {
    "approvalStatus": "ACCEPTED",
    "bundleVersion": "1.164",
	"bundleMetadata": {
      "eksVersion": "1.29"
    }
  }
}
´´´
Where ´approvalStatus´ can be either ´ACCEPTED´ or ´REJECTED´. ´bundleVersion´
is the the bundle version being approved or rejected. ´eksVersion´ is the
version of the customer's EKS cluster.
`

func resourceAzurePrivateContainerRegistry() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreatePrivateContainerRegistry,
		ReadContext:   azureReadPrivateContainerRegistry,
		UpdateContext: azureUpdatePrivateContainerRegistry,
		DeleteContext: azureDeletePrivateContainerRegistry,

		Description: description(azurePrivateContainerRegistryDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC cloud account ID (UUID).",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyAppID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Azure app registration application ID. Also known as the client ID.",
				ValidateFunc: validation.IsUUID,
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

func azureCreatePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreatePrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	appID, err := uuid.Parse(d.Get(keyAppID).(string))
	if err != nil {
		return diag.FromErr(err)
	}
	url := d.Get(keyURL).(string)
	if err := pcr.Wrap(client).SetAzureRegistry(ctx, id, appID, url); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	awsReadPrivateContainerRegistry(ctx, d, m)
	return nil
}

func azureReadPrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadPrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	pcrInfo, err := pcr.Wrap(client).AzureRegistry(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyAppID, pcrInfo.PCRDetails.ImagePullDetails.CustomerAppId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyURL, pcrInfo.PCRDetails.RegistryURL); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureUpdatePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureUpdatePrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	appID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	url := d.Get(keyURL).(string)
	if err := pcr.Wrap(client).SetAzureRegistry(ctx, id, appID, url); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func azureDeletePrivateContainerRegistry(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeletePrivateContainerRegistry")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := pcr.Wrap(client).RemoveRegistry(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
