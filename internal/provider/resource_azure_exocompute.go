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
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/exocompute"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlexocompute "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/exocompute"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
)

const resourceAzureExocomputeDescription = `
The ´polaris_azure_exocompute´ resource creates an RSC Exocompute configuration
for Azure workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When a host configuration is created, RSC will
    automatically deploy the necessary resources in the specified Azure region
    to run the Exocompute service. A host configuration can be used by both the
    host cloud account and application cloud accounts mapped to the host
    account.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the Azure
    AKS cluster attached by the customer, using the
    ´polaris_azure_exocompute_cluster_attachment´ resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Item 1 and 2 above requires that the Azure subscription has been onboarded with
the ´exocompute´ feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a ´polaris_azure_exocompute´ resource:
 1. Using the ´cloud_account_id´, ´region´, ´subnet´ and
   ´pod_overlay_network_cidr´ fields creates an RSC managed host configuration.
 2. Using the ´cloud_account_id´ and ´region´ fields creates a customer managed
    host configuration. Note, the ´polaris_azure_exocompute_cluster_attachment´
    resource must be used to attach an Azure AKS cluster to the Exocompute
    configuration.
 3. Using the ´cloud_account_id´ and ´host_cloud_account_id´ fields creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   ´pod_overlay_network_cidr´ field, this is discouraged and should only be done
   for backwards compatibility reasons.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.
`

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceAzureExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateExocompute,
		ReadContext:   azureReadExocompute,
		DeleteContext: azureDeleteExocompute,

		Description: description(resourceAzureExocomputeDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Exocompute configuration ID (UUID).",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keySubscriptionID},
				Description: "RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for " +
					"which the Exocompute service runs. Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyHostCloudAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyRegion},
				Description: "RSC cloud account ID of the shared exocompute host account. Changing this forces a new " +
					"resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyOptionalConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyAllowlistAdditionalIPs: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsIPAddress,
							},
							Optional: true,
							ForceNew: true,
							MinItems: 1,
							Description: "Allowlist additional IP addresses for the API server on the Kubernetes " +
								"cluster. Requires that the `allowlist_rubrik_ips` field is set to `true`. Changing " +
								"this forces a new resource to be created.",
							RequiredWith: []string{keyOptionalConfig + ".0." + keyAllowlistRubrikIPs},
						},
						keyAllowlistRubrikIPs: {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Description: "Allowlist Rubrik IPs for the API server on the Kubernetes cluster. " +
								"Defaults to `false`. Changing this forces a new resource to be created.",
						},
						keyClusterAccess: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlexocompute.AzureClusterAccessPrivate),
							ForceNew: true,
							Description: "Azure cluster access type. Possible values are " +
								"`AKS_CLUSTER_ACCESS_TYPE_PUBLIC` and `AKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Defaults to " +
								"`AKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlexocompute.AzureClusterAccessPrivate),
								string(gqlexocompute.AzureClusterAccessPublic),
							}, false),
						},
						keyClusterTier: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlexocompute.AzureClusterTierFree),
							ForceNew: true,
							Description: "Azure cluster tier. Possible values are `AKS_CLUSTER_TIER_FREE` and " +
								"`AKS_CLUSTER_TIER_STANDARD`. Defaults to `AKS_CLUSTER_TIER_FREE`. Changing this " +
								"forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlexocompute.AzureClusterTierFree),
								string(gqlexocompute.AzureClusterTierStandard),
							}, false),
						},
						keyDiskEncryptionAtHost: {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Description: "Enable disk encryption at host. Defaults to `false`. Changing this forces " +
								"a new resource to be created.",
						},
						keyMaxNodeCount: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlexocompute.AzureClusterNodeCountMedium),
							ForceNew: true,
							Description: "The maximum number of nodes each cluster can use. Make sure you have " +
								"enough IP addresses in the subnet or a node pool large enough to handle the number " +
								"of nodes to avoid launch failure. Possible values are `AKS_NODE_COUNT_BUCKET_SMALL` " +
								"(32 nodes), `AKS_NODE_COUNT_BUCKET_MEDIUM` (64 nodes), `AKS_NODE_COUNT_BUCKET_LARGE` " +
								"(128 nodes) and `AKS_NODE_COUNT_BUCKET_XLARGE` (256 nodes). Defaults to " +
								"`AKS_NODE_COUNT_BUCKET_MEDIUM`. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlexocompute.AzureClusterNodeCountSmall),
								string(gqlexocompute.AzureClusterNodeCountMedium),
								string(gqlexocompute.AzureClusterNodeCountLarge),
								string(gqlexocompute.AzureClusterNodeCountXLarge),
							}, false),
						},
						keyPrivateExocomputeDNSZoneID: {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Description: "Azure resource ID of the private DNS zone which will resolve the API " +
								"server URL for a private cluster. If empty, Azure will automatically create a " +
								"private DNS zone in the node resource group, and will delete it when the AKS " +
								"cluster is deleted. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyResourceGroupPrefix: {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Description: "Prefix of resource groups associated with the cluster, such as cluster " +
								"nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySnapshotPrivateAccessDNSZoneID: {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Description: "Azure resource ID of the private DNS zone linked to the exocompute VNet, " +
								"which will resolve private endpoints linked to snapshots. If empty, a new private " +
								"DNS zone will be created in the Exocompute resource group. Changing this forces a " +
								"new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUserDefinedRouting: {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Description: "Enable user defined routing. This allows the route for the Exocompute " +
								"egress traffic to be configured. Defaults to `false`. Changing this forces a new " +
								"resource to be created.",
						},
					},
				},
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostCloudAccountID},
				MaxItems:      1,
			},
			keyPodOverlayNetworkCIDR: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{keyHostCloudAccountID},
				ForceNew:      true,
				Description: "The CIDR range assigned to pods when launching Exocompute with the CNI overlay network " +
					"plugin mode. Rubrik recommends a size of /18 or larger. The pod CIDR must not overlap with the " +
					"cluster subnet or any IP ranges used in on-premises networks and other peered VNets. The " +
					"default space assigned by Azure is 10.244.0.0/16. Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyRegion: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{keyHostCloudAccountID},
				Description: "Azure region to run the exocompute service in. Should be specified in the standard " +
					"Azure style, e.g. `eastus`. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
			},
			keySubnet: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{keyHostCloudAccountID},
				ForceNew:      true,
				Description: "Azure subnet ID of the cluster subnet corresponding to the Exocompute configuration. " +
					"This subnet will be used to allocate IP addresses to the nodes of the cluster. Changing this " +
					"forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keySubscriptionID: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for " +
					"which the Exocompute service runs. Changing this forces a new resource to be created. " +
					"**Deprecated:** use `cloud_account_id` instead.",
				Deprecated:   "use `cloud_account_id` instead.",
				ValidateFunc: validation.IsUUID,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceAzureExocomputeV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceAzureExocomputeStateUpgradeV0,
			Version: 0,
		}},
	}
}

func azureCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureCreateExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Get(keyCloudAccountID).(string)
	if id == "" {
		id = d.Get(keySubscriptionID).(string)
	}
	accountID, err := uuid.Parse(id)
	if err != nil {
		return diag.FromErr(err)
	}

	if hostCloudAccount, ok := d.GetOk(keyHostCloudAccountID); ok {
		hostCloudAccountID, err := uuid.Parse(hostCloudAccount.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		err = exocompute.Wrap(client).MapAzureCloudAccount(ctx, accountID, hostCloudAccountID)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(appCloudAccountPrefix + accountID.String())
	} else {
		var exoConfig exocompute.AzureConfigurationFunc
		region := gqlazure.RegionFromName(d.Get(keyRegion).(string))
		if podOverlayNetworkCIDR, ok := d.GetOk(keyPodOverlayNetworkCIDR); ok {
			// RSC managed host configuration with overlay network.
			exoConfig = func(ctx context.Context, cloudAccountID uuid.UUID) (gqlexocompute.CreateAzureConfigurationParams, error) {
				return gqlexocompute.CreateAzureConfigurationParams{
					CloudAccountID:        cloudAccountID,
					IsManagedByRubrik:     true,
					Region:                region.ToCloudAccountRegionEnum(),
					SubnetID:              d.Get(keySubnet).(string),
					PodOverlayNetworkCIDR: podOverlayNetworkCIDR.(string),
					OptionalConfig:        fromAzureOptionalConfig(d),
				}, nil
			}
		} else if subnet, ok := d.GetOk(keySubnet); ok {
			// RSC managed host configuration.
			exoConfig = func(ctx context.Context, cloudAccountID uuid.UUID) (gqlexocompute.CreateAzureConfigurationParams, error) {
				return gqlexocompute.CreateAzureConfigurationParams{
					CloudAccountID:    cloudAccountID,
					IsManagedByRubrik: true,
					Region:            region.ToCloudAccountRegionEnum(),
					SubnetID:          subnet.(string),
					OptionalConfig:    fromAzureOptionalConfig(d),
				}, nil
			}
		} else {
			exoConfig = exocompute.AzureBYOKCluster(region)
		}
		exoConfigID, err := exocompute.Wrap(client).AddAzureConfiguration(ctx, accountID, exoConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(exoConfigID.String())
	}

	azureReadExocompute(ctx, d, m)
	return nil
}

func azureReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if id := d.Id(); strings.HasPrefix(id, appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}

		hostID, err := exocompute.Wrap(client).AzureHostCloudAccount(ctx, appID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if _, ok := d.GetOk(keySubscriptionID); ok {
			if err := d.Set(keySubscriptionID, appID.String()); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := d.Set(keyCloudAccountID, appID.String()); err != nil {
				return diag.FromErr(err)
			}
		}
		if err := d.Set(keyHostCloudAccountID, hostID.String()); err != nil {
			return diag.FromErr(err)
		}
	} else {
		exoConfigID, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}

		exoConfig, err := exocompute.Wrap(client).AzureConfigurationByID(ctx, exoConfigID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}
		if _, ok := d.GetOk(keySubscriptionID); ok {
			if err := d.Set(keySubscriptionID, exoConfig.CloudAccountID.String()); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := d.Set(keyCloudAccountID, exoConfig.CloudAccountID.String()); err != nil {
				return diag.FromErr(err)
			}
		}
		if err := d.Set(keyRegion, exoConfig.Region.Name()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keySubnet, exoConfig.SubnetID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyPodOverlayNetworkCIDR, exoConfig.PodOverlayNetworkCIDR); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyOptionalConfig, toAzureOptionalConfig(exoConfig.OptionalConfig)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func azureDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	if id := d.Id(); strings.HasPrefix(id, appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}
		err = exocompute.Wrap(client).UnmapAzureCloudAccount(ctx, appID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		exoConfigID, err := uuid.Parse(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		err = exocompute.Wrap(client).RemoveAzureConfiguration(ctx, exoConfigID)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	return nil
}

func fromAzureOptionalConfig(d *schema.ResourceData) *gqlexocompute.AzureOptionalConfig {
	block, ok := d.GetOk(keyOptionalConfig)
	if !ok || len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil
	}
	config := block.([]any)[0].(map[string]any)

	var additionalIPs []string
	for _, ip := range config[keyAllowlistAdditionalIPs].(*schema.Set).List() {
		additionalIPs = append(additionalIPs, ip.(string))
	}

	return &gqlexocompute.AzureOptionalConfig{
		AdditionalWhitelistIPs:     additionalIPs,
		ClusterAccess:              gqlexocompute.AzureClusterAccess(config[keyClusterAccess].(string)),
		ClusterTier:                gqlexocompute.AzureClusterTier(config[keyClusterTier].(string)),
		DiskEncryptionAtHost:       config[keyDiskEncryptionAtHost].(bool),
		ExocomputePrivateDnsZoneID: config[keyPrivateExocomputeDNSZoneID].(string),
		NodeCount:                  gqlexocompute.AzureClusterNodeCount(config[keyMaxNodeCount].(string)),
		NodeRGPrefix:               config[keyResourceGroupPrefix].(string),
		SnapshotPrivateDnsZoneId:   config[keySnapshotPrivateAccessDNSZoneID].(string),
		UserDefinedRouting:         config[keyUserDefinedRouting].(bool),
		WhitelistRubrikIPs:         config[keyAllowlistRubrikIPs].(bool),
	}
}

func toAzureOptionalConfig(config *gqlexocompute.AzureOptionalConfig) []any {
	if config == nil {
		return nil
	}

	additionalIPs := &schema.Set{F: schema.HashString}
	for _, ip := range config.AdditionalWhitelistIPs {
		additionalIPs.Add(ip)
	}

	return []any{map[string]any{
		keyAllowlistAdditionalIPs:         additionalIPs,
		keyAllowlistRubrikIPs:             config.WhitelistRubrikIPs,
		keyClusterAccess:                  string(config.ClusterAccess),
		keyClusterTier:                    string(config.ClusterTier),
		keyDiskEncryptionAtHost:           config.DiskEncryptionAtHost,
		keyMaxNodeCount:                   string(config.NodeCount),
		keyPrivateExocomputeDNSZoneID:     config.ExocomputePrivateDnsZoneID,
		keyResourceGroupPrefix:            config.NodeRGPrefix,
		keySnapshotPrivateAccessDNSZoneID: config.SnapshotPrivateDnsZoneId,
		keyUserDefinedRouting:             config.UserDefinedRouting,
	}}
}
