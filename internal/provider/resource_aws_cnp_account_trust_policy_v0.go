// Copyright 2025 Rubrik, Inc.
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
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAwsCnpAccountTrustPolicyV0() *schema.Resource {
	return &schema.Resource{
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
			keyExternalID: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "External ID. Changing this forces a new resource to be created.",
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"CLOUD_NATIVE_ARCHIVAL", "CLOUD_NATIVE_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION",
						"EXOCOMPUTE", "RDS_PROTECTION", "SERVERS_AND_APPS", "KUBERNETES_PROTECTION",
					}, false),
				},
				MinItems: 1,
				Required: true,
				ForceNew: true,
				Description: "RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, " +
					"`CLOUD_NATIVE_S3_PROTECTION`, `KUBERNETES_PROTECTION`, `EXOCOMPUTE` and `RDS_PROTECTION`. Changing this forces a new " +
					"resource to be created.",
			},
			keyPolicy: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "AWS IAM trust policy.",
			},
			keyRoleKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "RSC artifact key for the AWS role.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func resourceAwsCnpAccountTrustPolicyStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	log.Print("[TRACE] resourceAwsCnpAccountTrustPolicyStateUpgradeV0")

	accountID, err := uuid.Parse(state[keyID].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid resource id: %s", err)
	}

	// Migrate the resource ID to include the role key.
	trustPolicyID, err := joinTrustPolicyID(state[keyRoleKey].(string), accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create trust policy ID: %s", err)
	}
	state[keyID] = trustPolicyID
	return state, nil
}
