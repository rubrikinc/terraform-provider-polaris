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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceGcpProjectV1 defines the schema for version 1 of the GCP project
// resource.
func resourceGcpProjectV1() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{"credentials", "project"},
				ValidateFunc: validateFileExist,
			},
			"delete_snapshots_on_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"organization_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_number": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"credentials"},
				RequiredWith:  []string{"organization_name", "project", "project_number"},
				ValidateFunc:  validateStringIsNumber,
			},
		},
	}
}

// resourceAwsAccountStateUpgradeV1 introduces a cloud native protection
// feature block.
func resourceGcpProjectStateUpgradeV1(ctx context.Context, state map[string]interface{}, m interface{}) (map[string]interface{}, error) {
	tflog.Trace(ctx, "resourceGcpProjectStateUpgradeV1")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(state["id"].(string))
	if err != nil {
		return nil, err
	}

	account, err := gcp.Wrap(client).ProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Add the new cloud native protection feature block.
	cnpFeature, ok := account.Feature(core.FeatureExocompute)
	if !ok {
		return nil, errors.New("aws account missing cloud native protection")
	}

	state["cloud_native_protection"] = []interface{}{
		map[string]interface{}{
			"status": cnpFeature.Status,
		},
	}

	return state, nil
}
