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
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
)

func resourceGcpServiceAccountV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyCredentials: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				//ValidateFunc: validateFileExist,
				Description: "Path to GCP service account key file.",
			},
			keyName: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Service account name in Polaris. If not given the name of the service account key file is used.",
				//ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPermissionsHash: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Signals that the permissions has been updated.",
				//ValidateDiagFunc: validateHash,
			},
		},
	}
}

// resourceGcpServiceAccountStateUpgradeV0 changes the resource ID to be the
// SHA-256 sum of the service account name.
func resourceGcpServiceAccountStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	tflog.Trace(ctx, "resourceGcpServiceAccountStateUpgradeV0")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	name, err := gcp.Wrap(client).ServiceAccount(ctx)
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	hash.Write([]byte(name))
	state["id"] = fmt.Sprintf("%x", hash.Sum(nil))

	return state, nil
}
