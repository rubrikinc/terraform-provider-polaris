// Copyright 2024 Rubrik, Inc.
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
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqlgcp "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/gcp"
)

const dataSourceAccountDescription = `
The ´polaris_account´ data source is used to access information about the RSC account.

-> **Note:** The ´fqdn´ and ´name´ fields are read from the local RSC credentials and
   not from RSC.
`

func dataSourceAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: accountRead,

		Description: description(dataSourceAccountDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the features, the fully qualified domain name and the name.",
			},
			keyAws: {
				Type:        schema.TypeList,
				Elem:        cloudFeaturesResource(),
				Computed:    true,
				Description: "AWS cloud vendor information including supported features, and their permission groups.",
			},
			keyAzure: {
				Type:        schema.TypeList,
				Elem:        cloudFeaturesResource(),
				Computed:    true,
				Description: "Azure cloud vendor information including supported features, and their permission groups.",
			},
			keyFeatures: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Features enabled for the RSC account.",
			},
			keyFQDN: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified domain name of the RSC account.",
			},
			keyGcp: {
				Type:        schema.TypeList,
				Elem:        cloudFeaturesResource(),
				Computed:    true,
				Description: "GCP cloud vendor information including supported features, and their permission groups.",
			},
			keyName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RSC account name.",
			},
		},
	}
}

// cloudFeaturesResource returns the schema for a cloud vendor's features block.
func cloudFeaturesResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keyFeatures: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Features supported for this cloud vendor with their permission groups.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Feature name.",
						},
						keyPermissionGroups: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed:    true,
							Description: "Permission groups available for the feature.",
						},
					},
				},
			},
		},
	}
}

func accountRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "accountRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountFeatures, err := core.Wrap(client.GQL).EnabledFeaturesForAccount(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	accountFQDN := strings.ToLower(client.Account.AccountFQDN())
	accountName := strings.ToLower(client.Account.AccountName())

	accountFeaturesAttr := &schema.Set{F: schema.HashString}
	for _, accountFeature := range accountFeatures {
		accountFeaturesAttr.Add(accountFeature.Name)
	}
	if err := d.Set(keyFeatures, accountFeaturesAttr); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyFQDN, accountFQDN); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, accountName); err != nil {
		return diag.FromErr(err)
	}

	// Populate AWS features block with permission groups.
	awsPermGroups, err := gqlaws.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, aws.SupportedFeatures())
	if err != nil {
		return diag.FromErr(err)
	}
	awsFeaturesAttr := sortedAWSFeatures(awsPermGroups)
	if err := d.Set(keyAws, []map[string]any{{keyFeatures: awsFeaturesAttr}}); err != nil {
		return diag.FromErr(err)
	}

	// Populate Azure features block with permission groups.
	azurePermGroups, err := gqlazure.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, azure.SupportedFeatures())
	if err != nil {
		return diag.FromErr(err)
	}
	azureFeaturesAttr := sortedAzureFeatures(azurePermGroups)
	if err := d.Set(keyAzure, []map[string]any{{keyFeatures: azureFeaturesAttr}}); err != nil {
		return diag.FromErr(err)
	}

	// Populate GCP features block with permission groups.
	gcpPermGroups, err := gqlgcp.Wrap(client.GQL).AllPermissionsGroupsByFeature(ctx, gcp.SupportedFeatures())
	if err != nil {
		return diag.FromErr(err)
	}
	gcpFeaturesAttr := sortedGCPFeatures(gcpPermGroups)
	if err := d.Set(keyGcp, []map[string]any{{keyFeatures: gcpFeaturesAttr}}); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, accountFeature := range accountFeatures {
		hash.Write([]byte(accountFeature.Name))
	}
	hash.Write([]byte(accountFQDN))
	hash.Write([]byte(accountName))
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}

// sortedAWSFeatures sorts AWS permission groups by feature name and permission group name.
func sortedAWSFeatures(permGroups []gqlaws.FeaturePermissionGroups) []map[string]any {
	result := make([]map[string]any, 0, len(permGroups))
	for _, fp := range permGroups {
		var groups []string
		for _, pg := range fp.PermissionGroups {
			groups = append(groups, string(pg.PermissionGroup))
		}
		sort.Strings(groups)
		result = append(result, map[string]any{
			keyName:             fp.Feature,
			keyPermissionGroups: groups,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i][keyName].(string) < result[j][keyName].(string)
	})
	return result
}

// sortedAzureFeatures sorts Azure permission groups by feature name and permission group name.
func sortedAzureFeatures(permGroups []gqlazure.FeaturePermissionGroups) []map[string]any {
	result := make([]map[string]any, 0, len(permGroups))
	for _, fp := range permGroups {
		var groups []string
		for _, pg := range fp.PermissionGroups {
			groups = append(groups, string(pg.PermissionGroup))
		}
		sort.Strings(groups)
		result = append(result, map[string]any{
			keyName:             fp.Feature,
			keyPermissionGroups: groups,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i][keyName].(string) < result[j][keyName].(string)
	})
	return result
}

// sortedGCPFeatures sorts GCP permission groups by feature name and permission group name.
func sortedGCPFeatures(permGroups []gqlgcp.FeaturePermissionGroups) []map[string]any {
	result := make([]map[string]any, 0, len(permGroups))
	for _, fp := range permGroups {
		var groups []string
		for _, pg := range fp.PermissionGroups {
			groups = append(groups, string(pg.PermissionGroup))
		}
		sort.Strings(groups)
		result = append(result, map[string]any{
			keyName:             fp.Feature,
			keyPermissionGroups: groups,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i][keyName].(string) < result[j][keyName].(string)
	})
	return result
}
