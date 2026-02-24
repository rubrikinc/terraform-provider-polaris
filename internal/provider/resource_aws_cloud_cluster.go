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
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/cluster"
	gqlcloudcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cloudcluster"
	gqlcluster "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cluster"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core/secret"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"
)

const resourceAWSCloudClusterDescription = `
The ´polaris_aws_cloud_cluster´ resource creates an AWS cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in AWS using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual AWS infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The AWS account must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.

~> **Note:** Cloud Cluster deletion is now supported. When destroying this resource,
   the cluster will be removed from RSC. If the cluster has blocking conditions
   (active SLAs, global SLAs, or RCV locations), the deletion will fail and you must
   resolve these conditions first. Use the 'force_cluster_delete_on_destroy' option
   to force removal when eligible.
`

// This resource uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func resourceAwsCloudCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCloudCluster,
		ReadContext:   awsReadCloudCluster,
		UpdateContext: awsUpdateCloudCluster,
		DeleteContext: awsDeleteCloudCluster,
		Description:   description(resourceAWSCloudClusterDescription),
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
			keyRegion: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "AWS region to deploy the cluster in. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringInSlice(gqlaws.AllRegionNames(), false),
			},
			keyUsePlacementGroups: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     false,
				Description: "Whether to use placement groups for the cluster. Changing this forces a new resource to be created.",
			},
			keyClusterConfig: {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Configuration for the cloud cluster. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyClusterName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Unique name to assign to the cloud cluster.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyAdminEmail: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Email address for the cluster admin user. Changing this value will have no effect on the cluster.",
							ValidateFunc: validateEmailAddress,
						},
						keyAdminPassword: {
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							ForceNew:     true,
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
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							MinItems:    1,
							Description: "DNS name servers for the cluster.",
						},
						keyDNSSearchDomains: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							MinItems:    1,
							Description: "DNS search domains for the cluster.",
						},
						keyNTPServers: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							MinItems:    1,
							Description: "NTP servers for the cluster.",
						},
						keyBucketName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Name of the S3 bucket to use for the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyEnableImmutability: {
							Type:        schema.TypeBool,
							Required:    true,
							ForceNew:    true,
							Description: "Whether to enable immutability and object lock for the S3 bucket. Changing this forces a new resource to be created.",
						},
						keyKeepClusterOnFailure: {
							Type:        schema.TypeBool,
							Required:    true,
							ForceNew:    true,
							Description: "Whether to keep the cluster on failure (can be useful for troubleshooting). Changing this forces a new resource to be created.",
						},
						keyForceClusterDeleteOnDestroy: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether to force delete the cluster on destroy.",
						},
						keyDynamicScalingEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether to enable dynamic scaling for the cluster. Requires CDM Version 9.5+. Changing this forces a new resource to be created.",
							ForceNew:    true,
						},
						keyTimezone: {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "Timezone for the cluster.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyLocation: {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "Location for the cluster.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
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
							Description: "AWS instance type for the cluster nodes. Changing this forces a new resource to be created. Supported values are `M5_4XLARGE`, `M6I_2XLARGE`, `M6I_4XLARGE`, `M6I_8XLARGE`, `R6I_4XLARGE`, `M6A_2XLARGE`, `M6A_4XLARGE`, `M6A_8XLARGE` and `R6A_4XLARGE`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlcloudcluster.AwsInstanceTypeM5_4XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6I_2XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6I_4XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6I_8XLarge),
								string(gqlcloudcluster.AwsInstanceTypeR6I_4XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6A_2XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6A_4XLarge),
								string(gqlcloudcluster.AwsInstanceTypeM6A_8XLarge),
								string(gqlcloudcluster.AwsInstanceTypeR6A_4XLarge),
							}, false),
						},
						keyInstanceProfileName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS instance profile name for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyVPCID: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS VPC ID where the cluster will be deployed. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySubnetID: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS subnet ID where the cluster nodes will be deployed. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySecurityGroupIDs: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							ForceNew:    true,
							Description: "AWS security group IDs for the cluster nodes. Changing this forces a new resource to be created.",
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

// awsCreateCloudCluster creates the cloud cluster resource.
func awsCreateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateCloudCluster")

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

	securityGroupsSet := vmConfigMap[keySecurityGroupIDs].(*schema.Set)
	securityGroups := make([]string, 0, securityGroupsSet.Len())
	for _, sg := range securityGroupsSet.List() {
		securityGroups = append(securityGroups, sg.(string))
	}

	instanceTypeStr := vmConfigMap[keyInstanceType].(string)
	instanceType := gqlcloudcluster.AwsCCInstanceType(instanceTypeStr)
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

	vmConfig := gqlcloudcluster.AwsVmConfig{
		CDMVersion:          vmConfigMap[keyCDMVersion].(string),
		InstanceProfileName: vmConfigMap[keyInstanceProfileName].(string),
		InstanceType:        instanceType,
		SecurityGroups:      securityGroups,
		Subnet:              vmConfigMap[keySubnetID].(string),
		VMType:              vmType,
		VPC:                 vmConfigMap[keyVPCID].(string),
	}

	awsEsConfig := gqlcloudcluster.AwsEsConfigInput{
		BucketName:         clusterConfigMap[keyBucketName].(string),
		EnableImmutability: clusterConfigMap[keyEnableImmutability].(bool),
		ShouldCreateBucket: false,
		EnableObjectLock:   clusterConfigMap[keyEnableImmutability].(bool),
	}

	clusterConfig := gqlcloudcluster.AwsClusterConfig{
		ClusterName:           clusterConfigMap[keyClusterName].(string),
		UserEmail:             clusterConfigMap[keyAdminEmail].(string),
		AdminPassword:         secret.String(clusterConfigMap[keyAdminPassword].(string)),
		DNSNameServers:        dnsNameServers,
		DNSSearchDomains:      dnsSearchDomains,
		NTPServers:            ntpServers,
		NumNodes:              clusterConfigMap[keyNumNodes].(int),
		AwsEsConfig:           awsEsConfig,
		DynamicScalingEnabled: clusterConfigMap[keyDynamicScalingEnabled].(bool),
	}

	input := gqlcloudcluster.CreateAwsClusterInput{
		CloudAccountID:       cloudAccountID,
		ClusterConfig:        clusterConfig,
		IsEsType:             true,
		KeepClusterOnFailure: clusterConfigMap[keyKeepClusterOnFailure].(bool),
		Region:               d.Get(keyRegion).(string),
		UsePlacementGroups:   d.Get(keyUsePlacementGroups).(bool),
		Validations:          validations,
		VMConfig:             vmConfig,
	}

	cloudcluster, err := cloudcluster.Wrap(client).CreateCloudCluster(ctx, input, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cloudcluster.ID.String())

	vmConfigList = d.Get(keyVMConfig).([]any)
	if len(vmConfigList) > 0 {
		vmConfigMap := vmConfigList[0].(map[string]any)
		vmConfigMap[keyCDMProduct] = cloudcluster.CdmProduct
		d.Set(keyVMConfig, []any{vmConfigMap})
	}
	d.Set(keyCloudAccountID, cloudcluster.CloudAccountID)

	awsReadCloudCluster(ctx, d, m)
	return nil
}

// awsReadCloudCluster reads the cloud cluster resource.
func awsReadCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadCloudCluster")

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

	clusterFilter := gqlcluster.SearchFilter{
		ID: []string{id.String()},
	}

	// Use AllCloudClusters and filter for cluster
	cloudClusters, err := gqlcloudcluster.Wrap(client).AllCloudClusters(ctx, 1, "", clusterFilter, gqlcluster.SortByClusterName, core.SortOrderDesc)
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

	// Check if the CDM version changed
	vmConfigList := d.Get(keyVMConfig).([]any)
	vmConfigMap := vmConfigList[0].(map[string]any)
	vmConfigMap[keyCDMVersion] = cloudCluster.Version

	// Read DNS, NTP, and DNS Search Domains from API and check if they match the Terraform state
	dnsServers, err := gqlcluster.Wrap(client).DNSServers(ctx, uuid.MustParse(d.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	dnsNameServersSet := schema.Set{F: schema.HashString}
	for _, server := range dnsServers.Servers {
		dnsNameServersSet.Add(server)
	}
	clusterConfigMap[keyDNSNameServers] = &dnsNameServersSet

	dnsSearchDomainsSet := schema.Set{F: schema.HashString}
	for _, domain := range dnsServers.Domains {
		dnsSearchDomainsSet.Add(domain)
	}
	clusterConfigMap[keyDNSSearchDomains] = &dnsSearchDomainsSet

	ntpServers, err := gqlcluster.Wrap(client).NTPServers(ctx, uuid.MustParse(d.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	ntpServersSet := schema.Set{F: schema.HashString}
	for _, server := range ntpServers {
		ntpServersSet.Add(server.Server)
	}
	clusterConfigMap[keyNTPServers] = &ntpServersSet

	// Read cluster settings
	clusterSettings, err := gqlcluster.Wrap(client).ClusterSettings(ctx, uuid.MustParse(d.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	clusterConfigMap[keyClusterName] = clusterSettings.Name
	clusterConfigMap[keyTimezone] = clusterSettings.Timezone
	clusterConfigMap[keyLocation] = clusterSettings.RawAddress

	d.Set(keyClusterConfig, []any{clusterConfigMap})
	d.Set(keyVMConfig, []any{vmConfigMap})

	return nil
}

// awsDeleteCloudCluster deletes the cloud cluster resource.
func awsDeleteCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCloudCluster")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the force delete flag from the Terraform configuration
	clusterConfigList := d.Get(keyClusterConfig).([]any)
	clusterConfigMap := clusterConfigList[0].(map[string]any)
	forceRemoval := clusterConfigMap[keyForceClusterDeleteOnDestroy].(bool)

	// Attempt cluster removal
	// The RemoveCluster function will handle all prechecks and validations
	info, err := cluster.Wrap(client).RemoveCluster(ctx, clusterID, forceRemoval, 0)
	if err != nil {
		tflog.Error(ctx, "Failed to remove cloud cluster", map[string]any{
			"cluster_id":             clusterID.String(),
			"error":                  err.Error(),
			"blocking_conditions":    info.BlockingConditions,
			"force_removal_eligible": info.ForceRemovalEligible,
		})
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cloud cluster removal initiated successfully", map[string]any{
		"cluster_id": clusterID.String(),
	})

	d.SetId("")
	return nil
}

// awsUpdateCloudCluster updates the resource in-place. The following actions
// are supported:
//   - Update Network DNS
//   - Update Network DNS Search Domains
//   - Update NTP
//   - Update Cluster Name
//   - Update Timezone
//   - Update Location
func awsUpdateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCloudCluster")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gqlCluster := gqlcluster.Wrap(client.GQL)

	// Check if cluster_config block has changes
	if d.HasChange(keyClusterConfig) {
		clusterConfigList := d.Get(keyClusterConfig).([]any)
		if len(clusterConfigList) == 0 {
			return diag.Errorf("%s is required", keyClusterConfig)
		}
		clusterConfigMap := clusterConfigList[0].(map[string]any)

		// Check for DNS name servers or DNS search domains change
		if d.HasChange(keyClusterConfig+".0."+keyDNSNameServers) || d.HasChange(keyClusterConfig+".0."+keyDNSSearchDomains) {
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

			tflog.Debug(ctx, "Updating DNS servers and search domains", map[string]any{
				"cluster_id":     clusterID.String(),
				"dns_servers":    dnsNameServers,
				"search_domains": dnsSearchDomains,
			})

			input := gqlcluster.UpdateDNSServersAndSearchDomainsInput{
				ClusterID:     clusterID,
				DNSServers:    dnsNameServers,
				SearchDomains: dnsSearchDomains,
			}

			if err := gqlCluster.UpdateDNSServersAndSearchDomains(ctx, input); err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "DNS name servers and search domains updated", map[string]any{
				"cluster_id": clusterID.String(),
			})
		}

		// Check for NTP servers change
		if d.HasChange(keyClusterConfig + ".0." + keyNTPServers) {
			input := gqlcluster.UpdateClusterNTPServersInput{
				ClusterID: clusterID,
			}

			if ntpServersSet, ok := clusterConfigMap[keyNTPServers].(*schema.Set); ok {
				for _, ntp := range ntpServersSet.List() {
					input.Servers = append(input.Servers, struct {
						Server       string                      `json:"server"`
						SymmetricKey *gqlcluster.NTPSymmetricKey `json:"symmetricKey,omitempty"`
					}{
						Server: ntp.(string),
						// SymmetricKey is nil, so it will be omitted from JSON
					})
				}
			}

			tflog.Debug(ctx, "Updating NTP servers", map[string]any{
				"cluster_id":  clusterID.String(),
				"ntp_servers": input.Servers,
			})

			if err := gqlCluster.UpdateNTPServers(ctx, input); err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "NTP servers updated", map[string]any{
				"cluster_id": clusterID.String(),
			})

		}

		// Check for cluster name change, timezone change or location change
		// since these use the same API we need to update them together
		if d.HasChange(keyClusterConfig+".0."+keyClusterName) || d.HasChange(keyClusterConfig+".0."+keyTimezone) || d.HasChange(keyClusterConfig+".0."+keyLocation) {

			clusterName := clusterConfigMap[keyClusterName].(string)
			timezone := clusterConfigMap[keyTimezone].(string)
			location := clusterConfigMap[keyLocation].(string)

			var parsedTimezone gqlcluster.Timezone
			if timezone != "" {
				parsedTimezone, err = gqlcluster.ParseTimeZone(timezone)
				if err != nil {
					return diag.FromErr(err)
				}
			}

			input := gqlcluster.UpdatedSettings{
				ClusterID: clusterID,
				Name:      clusterName,
				Timezone:  parsedTimezone,
				Address:   location,
			}
			if _, err := gqlCluster.UpdateSettings(ctx, input); err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "Cluster settings updated", map[string]any{
				"cluster_id": clusterID.String(),
				"name":       clusterName,
				"timezone":   parsedTimezone,
				"address":    location,
			})
		}
	}

	return awsReadCloudCluster(ctx, d, m)
}
