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
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceAwsAccountV1 defines the schema for version 1 of the AWS account
// resource.
func resourceAwsAccountV1() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Account name in Polaris. If not given the name is taken from AWS Organizations or, if the required permissions are missing, is derived from the AWS account ID and the named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"exocompute": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"regions": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateAwsRegion,
							},
							Required:    true,
							Description: "Regions to enable the exocompute feature in.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the exocompute feature for the account.",
			},
			"profile": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "AWS named profile.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"regions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateAwsRegion,
				},
				Required:    true,
				Description: "Regions that Polaris will monitor for instances to automatically protect.",
			},
		},
	}
}

// resourceAwsAccountStateUpgradeV1 introduces a cloud native protection
// feature block and adds status to both feature blocks.
func resourceAwsAccountStateUpgradeV1(ctx context.Context, state map[string]interface{}, m interface{}) (map[string]interface{}, error) {
	tflog.Trace(ctx, "resourceAwsAccountStateUpgradeV1")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(state["id"].(string))
	if err != nil {
		return nil, err
	}

	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(id), core.FeatureAll)
	if err != nil {
		return nil, err
	}

	// Add status to exocompute feature block.
	feature, ok := state["exocompute"]
	if ok {
		if exoFeature, ok := account.Feature(core.FeatureExocompute); ok {
			exo := feature.([]interface{})[0].(map[string]interface{})
			exo["status"] = exoFeature.Status
		}
	}

	// Add the new cloud native protection feature block. Takes ownership
	// of the resource's regions.
	cnpFeature, ok := account.Feature(core.FeatureExocompute)
	if !ok {
		return nil, errors.New("aws account missing cloud native protection")
	}

	state["cloud_native_protection"] = []interface{}{
		map[string]interface{}{
			"regions": state["regions"],
			"status":  cnpFeature.Status,
		},
	}

	// Remove regions from the resource.
	delete(state, "regions")

	return state, nil
}
