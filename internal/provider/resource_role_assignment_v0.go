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
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

func resourceRoleAssignmentV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the user email and the role ID.",
			},
			keyRoleID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Role ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyUserEmail: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "User email address. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

// resourceRoleAssignmentStateUpgradeV0 changes the resource ID to be the user
// ID and not the hash of the user email address and role ID.
func resourceRoleAssignmentStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	log.Print("[TRACE] resourceRoleAssignmentStateUpgradeV0")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	email := state[keyUserEmail].(string)
	roleID := state[keyRoleID].(string)
	if id := state[keyID].(string); id != fmt.Sprintf("%x", sha256.Sum256([]byte(email+roleID))) {
		return nil, fmt.Errorf("failed to upgrade role assignment state, unexpected resource id: %s", id)
	}

	user, err := access.Wrap(client).UserByEmail(ctx, email, gqlaccess.DomainLocal)
	if err != nil {
		return nil, err
	}

	state[keyID] = user.ID
	return state, nil
}
