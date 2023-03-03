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
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// resourceAzureServicePrincipal defines the schema for the Azure service
// principal resource. Note that the delete function cannot remove the service
// principal since there is no delete operation in the Polaris API.
func resourceAzureServicePrincipalV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				AtLeastOneOf:     []string{"credentials", "app_id"},
				Description:      "Path to Azure service principal file.",
				ValidateDiagFunc: fileExists,
			},
			"app_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_name", "app_secret", "tenant_domain", "tenant_id"},
				Description:      "App registration application id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"app_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_secret", "tenant_domain", "tenant_id"},
				Description:      "App registration display name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"app_secret": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "tenant_domain", "tenant_id"},
				Description:      "App registration client secret.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_domain": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "app_secret", "tenant_id"},
				Description:      "Tenant directory/domain name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"tenant_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"app_id", "app_name", "app_secret", "tenant_domain"},
				Description:      "Tenant/domain id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
		},
	}
}

// resourceAzureServicePrincipalStateUpgradeV0 makes the tenant domain
// parameter required.
func resourceAzureServicePrincipalStateUpgradeV0(ctx context.Context, state map[string]interface{}, m interface{}) (map[string]interface{}, error) {
	log.Print("[TRACE] resourceAzureServicePrincipalStateUpgradeV0")

	// Tenant domain is only missing when the principal has been given as a
	// credentials file.
	credentials, ok := state["credentials"]
	if !ok {
		return state, nil
	}

	buf, err := os.ReadFile(credentials.(string))
	if err != nil {
		return nil, err
	}

	var tenantDomain struct {
		V0 string `json:"tenant_domain"`
		V1 string `json:"tenantDomain"`
	}
	if err := json.Unmarshal(buf, &tenantDomain); err != nil {
		return nil, err
	}

	switch {
	case tenantDomain.V0 != "":
		state["tenant_domain"] = tenantDomain.V0
	case tenantDomain.V1 != "":
		state["tenant_domain"] = tenantDomain.V1
	default:
		return nil, errors.New("credentials file does not contain tenant domain")
	}

	return state, nil
}
