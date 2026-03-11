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
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

type roleAssignmentModelV0 struct {
	ID        types.String `tfsdk:"id"`
	RoleID    types.String `tfsdk:"role_id"`
	UserEmail types.String `tfsdk:"user_email"`
}

// UpgradeState upgrades the state of the role assignment resource to the
// latest version. Note: each StateUpgrader must upgrade directly from its
// prior version to the current version. The framework does not chain through
// intermediate versions.
func (r *roleAssignmentResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	tflog.Trace(ctx, "roleAssignmentResource.UpgradeState")

	return map[int64]resource.StateUpgrader{
		0: r.upgradeStateV0(),
	}
}

// upgradeStateV0 upgrades v0 state (SDKv2 format with SHA-256 hash ID) to v1
// by looking up the user by email and replacing the hash ID with the user ID.
func (r *roleAssignmentResource) upgradeStateV0() resource.StateUpgrader {
	return resource.StateUpgrader{
		PriorSchema: &schema.Schema{
			Attributes: map[string]schema.Attribute{
				keyID: schema.StringAttribute{
					Computed:    true,
					Description: "SHA-256 hash of the user email and the role ID.",
				},
				keyRoleID: schema.StringAttribute{
					Required:    true,
					Description: "Role ID (UUID).",
				},
				keyUserEmail: schema.StringAttribute{
					Required:    true,
					Description: "User email address.",
				},
			},
		},
		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, res *resource.UpgradeStateResponse) {
			tflog.Trace(ctx, "roleAssignmentResource.upgradeStateV0")

			var prior roleAssignmentModelV0
			res.Diagnostics.Append(req.State.Get(ctx, &prior)...)
			if res.Diagnostics.HasError() {
				return
			}

			email := prior.UserEmail.ValueString()
			roleID := prior.RoleID.ValueString()
			expectedID := fmt.Sprintf("%x", sha256.Sum256([]byte(email+roleID)))
			if prior.ID.ValueString() != expectedID {
				res.Diagnostics.AddError("State upgrade failed",
					fmt.Sprintf("unexpected resource id: %s", prior.ID.ValueString()))
				return
			}

			polarisClient, err := r.client.polaris()
			if err != nil {
				res.Diagnostics.AddError("Client error", err.Error())
				return
			}

			user, err := access.Wrap(polarisClient).UserByEmail(ctx, email, gqlaccess.DomainLocal)
			if err != nil {
				res.Diagnostics.AddError("Failed to look up user by email", err.Error())
				return
			}

			var state roleAssignmentModel
			state.ID = types.StringValue(user.ID)
			state.RoleID = prior.RoleID
			state.RoleIDs = types.SetNull(types.StringType)
			state.SSOGroupID = types.StringNull()
			state.UserEmail = prior.UserEmail
			state.UserID = types.StringNull()
			res.Diagnostics.Append(res.State.Set(ctx, &state)...)
		},
	}
}
