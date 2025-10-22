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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/cloudcluster"
	gqlcloudcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cloudcluster"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core/secret"
	azureRegion "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
)

const resourceAzureCloudClusterDescription = `
The ´polaris_azure_cloud_cluster´ resource creates an Azure cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in Azure using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual Azure infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The Azure subscription must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.

~> **Note:** Cloud Cluster Removal is not supported via terraform yet. The cluster
   will be removed from state and you must remove the cluster through the RSC UI.
`

func resourceAzureCloudCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateCloudCluster,
		ReadContext:   azureReadCloudCluster,
		DeleteContext: azureDeleteCloudCluster,
		Description:   description(resourceAzureCloudClusterDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud cluster ID (UUID).",
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyClusterConfig: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "Configuration for the cloud cluster. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyClusterName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Unique name to assign to the cloud cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyAdminEmail: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Email address for the cluster admin user. Changing this value will have no effect on the cluster.",
							ValidateFunc: validateEmailAddress,
						},
						keyAdminPassword: {
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							Description:  "Password for the cluster admin user. Changing this value will have no effect on the cluster.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyNumNodes: {
							Type:         schema.TypeInt,
							Required:     true,
							ForceNew:     true,
							Description:  "Number of nodes in the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validateNumNodes,
						},
						keyDNSNameServers: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							Description: "DNS name servers for the cluster. Changing this forces a new resource to be created.",
						},
						keyDNSSearchDomains: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							ForceNew:    true,
							MinItems:    1,
							Description: "DNS search domains for the cluster. Changing this forces a new resource to be created.",
						},
						keyNTPServers: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							Description: "NTP servers for the cluster. Changing this forces a new resource to be created.",
						},
						keyKeepClusterOnFailure: {
							Type:        schema.TypeBool,
							Required:    true,
							ForceNew:    true,
							Description: "Whether to keep the cluster on failure (can be useful for troubleshooting). Changing this forces a new resource to be created.",
						},
					},
				},
			},
			keyVMConfig: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "VM configuration for the cluster nodes. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyCDMVersion: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "CDM version to use. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyCDMProduct: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "CDM Product Code. This is a read-only field and computed based on the CDM version.",
						},
						keyInstanceType: {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Azure instance type for the cluster nodes. Allowed values are `STANDARD_DS5_V2`, `STANDARD_D16S_V5`, `STANDARD_D8S_V5`, `STANDARD_D32S_V5`, `STANDARD_E16S_V5`, `STANDARD_D8AS_V5`, `STANDARD_D16AS_V5`, `STANDARD_D32AS_V5` and `STANDARD_E16AS_V5`. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlcloudcluster.AzureInstanceTypeStandardDS5V2),
								string(gqlcloudcluster.AzureInstanceTypeStandardD16SV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardD8SV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardD32SV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardE16SV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardD8ASV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardD16ASV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardD32ASV5),
								string(gqlcloudcluster.AzureInstanceTypeStandardE16ASV5),
							}, false),
						},
						keyResourceGroupName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure resource group name where the cluster will be deployed. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyStorageAccountName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure storage account name for the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyContainerName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure storage container name for the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyEnableImmutability: {
							Type:        schema.TypeBool,
							Required:    true,
							ForceNew:    true,
							Description: "Whether to enable immutability for the storage account. Changing this forces a new resource to be created.",
						},
						keyUserAssignedManagedIdentityName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Name of the user-assigned managed identity. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyRegion: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure region to deploy the cluster in. The format should be the native Azure format, e.g. `eastus`, `westus`, etc. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice(azureRegion.AllRegionNames(), false),
						},
						keyNetworkResourceGroup: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure resource group name for network resources. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyVnetResourceGroup: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure resource group name for the virtual network. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySubnet: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure subnet name for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyVnet: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure virtual network name. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyNetworkSecurityGroup: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure network security group name. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyNetworkSecurityResourceGroup: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Azure resource group name for the network security group. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyVMType: {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Default:     "DENSE",
							Description: "VM type for the cluster. Changing this forces a new resource to be created. Possible values are `STANDARD`, `DENSE` and `EXTRA_DENSE`. `DENSE` is recommended for CCES.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlcloudcluster.CCVmConfigStandard),
								string(gqlcloudcluster.CCVmConfigDense),
								string(gqlcloudcluster.CCVmConfigExtraDense),
							}, false),
						},
					},
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(60 * time.Minute),
			Read:    schema.DefaultTimeout(20 * time.Minute),
			Default: schema.DefaultTimeout(20 * time.Minute),
		},
	}
}

func azureCreateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureCreateCloudCluster")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	vmConfigList := d.Get(keyVMConfig).([]any)
	if len(vmConfigList) == 0 {
		return diag.Errorf("%s is required", keyVMConfig)
	}
	vmConfigMap := vmConfigList[0].(map[string]any)

	instanceTypeStr := vmConfigMap[keyInstanceType].(string)
	vmTypeStr := vmConfigMap[keyVMType].(string)
	vmType := gqlcloudcluster.VmConfigType(vmTypeStr)

	clusterConfigMap := d.Get(keyClusterConfig).([]any)[0].(map[string]any)

	dnsNameServers := make([]string, 0)
	if dnsNameServersSet, ok := clusterConfigMap[keyDNSNameServers].(*schema.Set); ok {
		for _, dns := range dnsNameServersSet.List() {
			dnsNameServers = append(dnsNameServers, dns.(string))
		}
	}

	dnsSearchDomains := make([]string, 0)
	if dnsSearchDomainsSet, ok := clusterConfigMap[keyDNSSearchDomains].(*schema.Set); ok {
		for _, domain := range dnsSearchDomainsSet.List() {
			dnsSearchDomains = append(dnsSearchDomains, domain.(string))
		}
	}

	ntpServers := make([]string, 0)
	if ntpServersSet, ok := clusterConfigMap[keyNTPServers].(*schema.Set); ok {
		for _, ntp := range ntpServersSet.List() {
			ntpServers = append(ntpServers, ntp.(string))
		}
	}

	validations := []gqlcloudcluster.ClusterCreateValidations{
		gqlcloudcluster.AllChecks,
	}

	region := azureRegion.RegionFromName(vmConfigMap[keyRegion].(string))

	vmConfig := gqlcloudcluster.AzureVMConfig{
		CDMVersion:                   vmConfigMap[keyCDMVersion].(string),
		InstanceType:                 gqlcloudcluster.AzureCCESSupportedInstanceType(instanceTypeStr),
		Location:                     region,
		ResourceGroup:                vmConfigMap[keyResourceGroupName].(string),
		NetworkResourceGroup:         vmConfigMap[keyNetworkResourceGroup].(string),
		VnetResourceGroup:            vmConfigMap[keyVnetResourceGroup].(string),
		Subnet:                       vmConfigMap[keySubnet].(string),
		Vnet:                         vmConfigMap[keyVnet].(string),
		NetworkSecurityGroup:         vmConfigMap[keyNetworkSecurityGroup].(string),
		NetworkSecurityResourceGroup: vmConfigMap[keyNetworkSecurityResourceGroup].(string),
		VMType:                       vmType,
	}

	azureEsConfig := gqlcloudcluster.AzureEsConfigInput{
		ResourceGroup:         vmConfigMap[keyResourceGroupName].(string),
		StorageAccount:        vmConfigMap[keyStorageAccountName].(string),
		ContainerName:         vmConfigMap[keyContainerName].(string),
		ShouldCreateContainer: false,
		EnableImmutability:    vmConfigMap[keyEnableImmutability].(bool),
		ManagedIdentity: gqlcloudcluster.AzureManagedIdentityName{
			Name: vmConfigMap[keyUserAssignedManagedIdentityName].(string),
		},
	}

	clusterConfig := gqlcloudcluster.AzureClusterConfig{
		ClusterName:      clusterConfigMap[keyClusterName].(string),
		UserEmail:        clusterConfigMap[keyAdminEmail].(string),
		AdminPassword:    secret.String(clusterConfigMap[keyAdminPassword].(string)),
		DNSNameServers:   dnsNameServers,
		DNSSearchDomains: dnsSearchDomains,
		NTPServers:       ntpServers,
		NumNodes:         clusterConfigMap[keyNumNodes].(int),
		AzureESConfig:    azureEsConfig,
	}

	input := gqlcloudcluster.CreateAzureClusterInput{
		CloudAccountID:       cloudAccountID,
		ClusterConfig:        clusterConfig,
		IsESType:             true,
		KeepClusterOnFailure: clusterConfigMap[keyKeepClusterOnFailure].(bool),
		Validations:          validations,
		VMConfig:             vmConfig,
	}

	azureCluster, err := cloudcluster.Wrap(client).CreateAzureCloudCluster(ctx, input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(azureCluster.ID.String())

	vmConfigList = d.Get(keyVMConfig).([]any)
	if len(vmConfigList) > 0 {
		vmConfigMap := vmConfigList[0].(map[string]any)
		vmConfigMap[keyCDMProduct] = azureCluster.CdmProduct
		d.Set(keyVMConfig, []any{vmConfigMap})
	}
	d.Set(keyCloudAccountID, azureCluster.CloudAccountID)

	return azureReadCloudCluster(ctx, d, m)
}

func azureReadCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureReadCloudCluster")

	// For cloud clusters, the read operation is limited since the cluster
	// creation is a long-running operation and the cluster state is managed
	// by RSC. We mainly verify that the resource still exists in the state.

	// If the ID is empty, the resource doesn't exist
	if d.Id() == "" {
		return nil
	}

	// create gqlapi client
	client := m.(*client).polarisClient.GQL
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	clusterFilter := gqlcloudcluster.ClusterFilter{
		ID: []string{id.String()},
	}

	// Use AllCloudClusters and filter for cluster
	cloudClusters, err := gqlcloudcluster.Wrap(client).AllCloudClusters(ctx, 1, "", clusterFilter, gqlcloudcluster.SortByClusterName, core.SortOrderDesc)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(cloudClusters) == 0 {
		d.SetId("")
		return nil
	}

	cloudCluster := cloudClusters[0]
	// validate the cloud cluster ID
	if cloudCluster.ID != id {
		return diag.Errorf("Cloud cluster ID mismatch. Expected %q, got %q", id, cloudCluster.ID)
	}

	// Get and update cluster_config block
	clusterConfigList := d.Get(keyClusterConfig).([]any)
	clusterConfigMap := clusterConfigList[0].(map[string]any)
	clusterConfigMap[keyClusterName] = cloudCluster.Name

	// Check if the CDM version changed
	vmConfigList := d.Get(keyVMConfig).([]any)
	vmConfigMap := vmConfigList[0].(map[string]any)
	vmConfigMap[keyCDMVersion] = cloudCluster.Version

	d.Set(keyClusterConfig, []any{clusterConfigMap})
	d.Set(keyVMConfig, []any{vmConfigMap})

	return nil
}

func azureDeleteCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "azureDeleteCloudCluster")

	// Cluster Removal is not supported via terraform yet. The user must remove the
	// cluster through the RSC UI. This will be implemented in the future.

	tflog.Warn(ctx, "Cloud cluster deletion should be handled through RSC directly. Removing from Terraform state only.")

	d.SetId("")
	return nil
}
