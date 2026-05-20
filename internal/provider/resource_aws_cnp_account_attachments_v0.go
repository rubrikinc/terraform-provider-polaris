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

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceAwsCnpAccountAttachmentsV0 defines the schema for version 0 of the
// AWS CNP account attachments resource, where features were a flat set of
// feature name strings with no permission group association.
func resourceAwsCnpAccountAttachmentsV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:     schema.TypeString,
				Computed: true,
			},
			keyAccountID: {
				Type:     schema.TypeString,
				Required: true,
			},
			keyFeatures: {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			keyInstanceProfile: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyKey:  {Type: schema.TypeString, Required: true},
						keyName: {Type: schema.TypeString, Required: true},
					},
				},
			},
			keyRoleChainingAccountID: {
				Type:     schema.TypeString,
				Optional: true,
			},
			keyRole: {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyKey:         {Type: schema.TypeString, Required: true},
						keyARN:         {Type: schema.TypeString, Required: true},
						keyPermissions: {Type: schema.TypeString, Optional: true},
					},
				},
			},
		},
	}
}

// resourceAwsCnpAccountAttachmentsStateUpgradeV0 converts the flat `features`
// set into `feature` blocks carrying a permission_groups set. For each feature
// in v0 state, the live RSC account is consulted to populate the actual
// permission groups currently registered — this avoids guessing a single
// default (e.g. BASIC) that wouldn't be correct for features like EXOCOMPUTE
// (BASIC + RSC_MANAGED_CLUSTER), SERVERS_AND_APPS (CLOUD_CLUSTER_ES, no BASIC)
// or migrated CLOUD_NATIVE_PROTECTION (BASIC + RESTORE + EXPORT_POWER_ON +
// EXPORT_POWER_OFF + DOWNLOAD_FILE). When a feature in v0 state is not present
// on the account (drift), BASIC is used as a conservative fallback that the
// next `terraform plan` will reconcile.
func resourceAwsCnpAccountAttachmentsStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	tflog.Trace(ctx, "resourceAwsCnpAccountAttachmentsStateUpgradeV0")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(state[keyID].(string))
	if err != nil {
		return nil, err
	}

	account, err := aws.Wrap(client).AccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldFeatures, _ := state[keyFeatures].([]any)
	featureBlocks := make([]any, 0, len(oldFeatures))
	for _, raw := range oldFeatures {
		name := raw.(string)
		groups := permissionGroupsFromAccount(account, name)
		featureBlocks = append(featureBlocks, map[string]any{
			keyName:             name,
			keyPermissionGroups: groups,
		})
	}
	state[keyFeature] = featureBlocks
	delete(state, keyFeatures)

	return state, nil
}

// permissionGroupsFromAccount returns the permission group names currently
// registered for the named feature on the given account, or ["BASIC"] when the
// feature is not present (drift fallback).
func permissionGroupsFromAccount(account aws.CloudAccount, name string) []any {
	for _, f := range account.Features {
		if f.Feature.Name == name {
			groups := make([]any, 0, len(f.PermissionGroups))
			for _, g := range f.PermissionGroups {
				groups = append(groups, string(g))
			}
			if len(groups) == 0 {
				groups = append(groups, string(core.PermissionGroupBasic))
			}
			return groups
		}
	}
	return []any{string(core.PermissionGroupBasic)}
}
