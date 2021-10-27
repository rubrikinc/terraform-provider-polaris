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
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceGcpProjectV0 defines the schema for version 0 of the GCP project
// resource.
func resourceGcpProjectV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				AtLeastOneOf:     []string{"credentials", "project"},
				ValidateDiagFunc: fileExists,
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
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				ValidateDiagFunc: stringIsInteger,
			},
		},
	}
}

// resourceGcpProjectStateUpgradeV0 simplifies the resource id to consist of
// only the Polaris cloud account id.
func resourceGcpProjectStateUpgradeV0(ctx context.Context, state map[string]interface{}, m interface{}) (map[string]interface{}, error) {
	log.Print("[TRACE] resourceGcpProjectStateUpgradeV0")

	client := m.(*polaris.Client)

	// Split the id into Polaris cloud account id and GCP project id.
	parts := strings.Split(state["id"].(string), ":")
	if len(parts) != 2 {
		return state, errors.New("invalid id format for v0 resource state")
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		return state, err
	}

	// Retrieve the account using the Polaris cloud account id.
	account1, err := client.GCP().Project(ctx, gcp.CloudAccountID(id), core.FeatureCloudNativeProtection)
	if err != nil {
		return nil, err
	}

	// Retrieve the account using the GCP project id.
	account2, err := client.GCP().Project(ctx, gcp.ProjectID(parts[1]), core.FeatureCloudNativeProtection)
	if err != nil {
		return nil, err
	}

	// Make sure the two ids refer to the same Polaris cloud account.
	if account1.ID != account2.ID {
		return state, errors.New("v0 id refers to two different accounts")
	}

	// Update the id to consist of only the Polaris cloud account id.
	state["id"] = account1.ID.String()
	return state, nil
}
