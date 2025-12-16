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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
)

// resourceSLADomainAssignmentV0 returns the V0 schema for the
// polaris_sla_domain_assignment resource. This is used by the state upgrader.
func resourceSLADomainAssignmentV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SLA domain ID (UUID).",
			},
			keyObjectIDs: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
				MinItems:    1,
				Required:    true,
				Description: "Object IDs (UUID).",
			},
			keySLADomainID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "SLA domain ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
		},
	}
}

// resourceSLADomainAssignmentStateUpgradeV0 upgrades the state from V0 to V1.
// V0 state only had sla_domain_id and object_ids. V1 adds assignment_type,
// apply_changes_to_existing_snapshots, and apply_changes_to_non_policy_snapshots.
func resourceSLADomainAssignmentStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	tflog.Trace(ctx, "resourceSLADomainAssignmentStateUpgradeV0")

	// Add the new fields with their default values. The old state only
	// supported protectWithSlaId, so we set that as the assignment type.
	state[keyAssignmentType] = string(gqlsla.ProtectWithSLA)
	state[keyApplyChangesToExistingSnapshots] = true
	state[keyApplyChangesToNonPolicySnapshots] = false

	return state, nil
}
