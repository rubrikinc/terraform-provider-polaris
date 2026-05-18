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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceAwsCnpAccountAttachmentsV0 defines the schema for version 0 of the
// AWS CNP account attachments resource, where features were a flat set of
// feature name strings with no permission group association.
func resourceAwsCnpAccountAttachmentsV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"account_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"features": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"instance_profile": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key":  {Type: schema.TypeString, Required: true},
						"name": {Type: schema.TypeString, Required: true},
					},
				},
			},
			"role_chaining_account_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"role": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key":         {Type: schema.TypeString, Required: true},
						"arn":         {Type: schema.TypeString, Required: true},
						"permissions": {Type: schema.TypeString, Optional: true},
					},
				},
			},
		},
	}
}

// resourceAwsCnpAccountAttachmentsStateUpgradeV0 converts the flat `features`
// set into `feature` blocks carrying a permission_groups set. Existing v0
// states only ever onboarded features at BASIC, so backfill BASIC for every
// migrated feature.
func resourceAwsCnpAccountAttachmentsStateUpgradeV0(ctx context.Context, state map[string]any, m any) (map[string]any, error) {
	tflog.Trace(ctx, "resourceAwsCnpAccountAttachmentsStateUpgradeV0")

	oldFeatures, _ := state["features"].([]any)
	featureBlocks := make([]any, 0, len(oldFeatures))
	for _, name := range oldFeatures {
		featureBlocks = append(featureBlocks, map[string]any{
			"name":              name,
			"permission_groups": []any{"BASIC"},
		})
	}
	state["feature"] = featureBlocks
	delete(state, "features")

	return state, nil
}
