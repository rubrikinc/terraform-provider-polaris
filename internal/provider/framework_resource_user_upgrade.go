// Copyright 2026 Rubrik, Inc.
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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

type userResourceModelV0 struct {
	ID             types.String `tfsdk:"id"`
	Email          types.String `tfsdk:"email"`
	IsAccountOwner types.Bool   `tfsdk:"is_account_owner"`
	RoleIDs        types.Set    `tfsdk:"role_ids"`
	Status         types.String `tfsdk:"status"`
}

func (r *userResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	tflog.Trace(ctx, "userResource.UpgradeState")

	return map[int64]resource.StateUpgrader{
		0: r.upgradeStateV0(),
	}
}

// upgradeStateV0 upgrades v0 state where the resource ID was the user email
// address to v1 where the resource ID is the user UUID.
func (r *userResource) upgradeStateV0() resource.StateUpgrader {
	return resource.StateUpgrader{
		PriorSchema: &schema.Schema{
			Attributes: map[string]schema.Attribute{
				keyID: schema.StringAttribute{
					Computed:    true,
					Description: "User email address.",
				},
				keyEmail: schema.StringAttribute{
					Required:    true,
					Description: "User email address. Changing this forces a new resource to be created.",
				},
				keyIsAccountOwner: schema.BoolAttribute{
					Computed:    true,
					Description: "True if the user is the account owner.",
				},
				keyRoleIDs: schema.SetAttribute{
					ElementType: types.StringType,
					Required:    true,
					Description: "Roles assigned to the user (UUIDs).",
				},
				keyStatus: schema.StringAttribute{
					Computed:    true,
					Description: "User status.",
				},
			},
		},
		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, res *resource.UpgradeStateResponse) {
			tflog.Trace(ctx, "userResource.upgradeStateV0")

			var prior userResourceModelV0
			res.Diagnostics.Append(req.State.Get(ctx, &prior)...)
			if res.Diagnostics.HasError() {
				return
			}

			email := prior.Email.ValueString()
			if id := prior.ID.ValueString(); id != email {
				res.Diagnostics.AddError("State upgrade failed",
					fmt.Sprintf("unexpected mismatch between user ID and email address: %s != %s", id, email))
				return
			}

			polarisClient, err := r.client.polaris()
			if err != nil {
				res.Diagnostics.AddError("RSC client error", err.Error())
				return
			}

			user, err := access.Wrap(polarisClient).UserByEmail(ctx, email, gqlaccess.DomainLocal)
			if err != nil {
				res.Diagnostics.AddError("Failed to look up user by email", err.Error())
				return
			}

			var state userResourceModel
			state.ID = types.StringValue(user.ID)
			state.Domain = types.StringValue(string(user.Domain))
			state.Email = prior.Email
			state.IsAccountOwner = prior.IsAccountOwner
			state.RoleIDs = prior.RoleIDs
			state.Status = prior.Status
			res.Diagnostics.Append(res.State.Set(ctx, &state)...)
		},
	}
}
