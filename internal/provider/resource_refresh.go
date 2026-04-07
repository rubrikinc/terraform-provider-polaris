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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/hierarchy"
)

const resourceRefreshDescription = `
The ´polaris_refresh´ resource blocks until an account or subscription's
inventory refresh in RSC is newer than a user-specified timestamp. This is
useful for ensuring that leaf objects such as virtual machines or EC2 instances
are discoverable via ´polaris_object´ after a subscription or account is
onboarded.

The resource does not trigger a refresh — RSC handles that automatically. It
simply polls until the condition is met. All arguments are ´ForceNew´, so any
change destroys and recreates the resource (re-polls).

~> Automatic refresh requires the ´cloud_discovery´ feature to be enabled on
the account or subscription. Without it, the resource may time out waiting for
a refresh that never occurs.

The default timeout is 45 minutes and can be overridden with a ´timeouts´
block.
`

func resourceRefresh() *schema.Resource {
	return &schema.Resource{
		CreateContext: refreshCreate,
		ReadContext:   refreshRead,
		DeleteContext: refreshDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},

		Description: description(resourceRefreshDescription),
		Schema: map[string]*schema.Schema{
			keyObjectID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC object ID (UUID) to monitor. Typically the output of ´polaris_object´.",
				ValidateFunc: validation.IsUUID,
			},
			keyObjectType: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Object type to monitor. Supported types: ´AwsNativeAccount´, ´AzureNativeSubscription´.",
				ValidateFunc: validation.StringInSlice([]string{
					"AwsNativeAccount",
					"AzureNativeSubscription",
				}, false),
			},
			keyTimestamp: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RFC3339 timestamp. The resource blocks until all features have been refreshed after this time.",
				ValidateFunc: validation.IsRFC3339Time,
			},
		},
	}
}

func refreshCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "refreshCreate")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	objectID, err := uuid.Parse(d.Get(keyObjectID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	timestamp, err := time.Parse(time.RFC3339, d.Get(keyTimestamp).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	objectType := d.Get(keyObjectType).(string)

	var features func(ctx context.Context) ([]hierarchy.Feature, error)
	switch objectType {
	case "AwsNativeAccount":
		features = func(ctx context.Context) ([]hierarchy.Feature, error) {
			obj, err := hierarchy.ObjectByIDAndWorkload[hierarchy.AWSNativeAccount](
				ctx, client.GQL, objectID, hierarchy.WorkloadAllSubHierarchyType,
			)
			if err != nil {
				return nil, err
			}
			return obj.Features, nil
		}
	case "AzureNativeSubscription":
		features = func(ctx context.Context) ([]hierarchy.Feature, error) {
			obj, err := hierarchy.ObjectByIDAndWorkload[hierarchy.AzureNativeSubscription](
				ctx, client.GQL, objectID, hierarchy.WorkloadAllSubHierarchyType,
			)
			if err != nil {
				return nil, err
			}
			return obj.Features, nil
		}
	default: // Unreachable: ValidateFunc restricts objectType to known values.
		return diag.Errorf("unsupported object_type: %s", objectType)
	}

	if err := pollForRefresh(ctx, objectID, timestamp, features); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(objectID.String())
	return nil
}

// pollForRefresh polls until all features returned by the features function
// have been refreshed after the given timestamp.
func pollForRefresh(ctx context.Context, objectID uuid.UUID, timestamp time.Time, features func(ctx context.Context) ([]hierarchy.Feature, error)) error {
	for {
		feats, err := features(ctx)
		if err != nil {
			tflog.Warn(ctx, "failed to query features, will retry", map[string]any{
				"object_id": objectID.String(),
				"error":     err.Error(),
			})

			select {
			case <-ctx.Done():
				return fmt.Errorf("timed out waiting for %s to be refreshed after %s: last error: %s", objectID, timestamp.Format(time.RFC3339), err)
			case <-time.After(10 * time.Second):
			}
			continue
		}

		allRefreshed := len(feats) > 0
		for _, f := range feats {
			if !f.LastRefreshedAt.After(timestamp) {
				allRefreshed = false
				break
			}
		}

		if allRefreshed {
			return nil
		}

		tflog.Debug(ctx, "waiting for refresh", map[string]any{
			"object_id": objectID.String(),
			"timestamp": timestamp.Format(time.RFC3339),
		})

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for %s to be refreshed after %s", objectID, timestamp.Format(time.RFC3339))
		case <-time.After(10 * time.Second):
		}
	}
}

func refreshRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	return nil
}

func refreshDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	return nil
}
