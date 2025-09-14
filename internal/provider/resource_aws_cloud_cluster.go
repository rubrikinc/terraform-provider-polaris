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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/cloudcluster"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const resourceAWSCloudClusterDescription = `
The ´polaris_aws_cloud_cluster´ resource creates an AWS cloud cluster in RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster in AWS using
the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual AWS infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required in some cases.

~> **Note:** The AWS account must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.

~> **Note:** Cloud Cluster Removal is not supported via terraform yet. The cluster
   will be removed from state and you must remove the cluster through the RSC UI.
`

func resourceAwsCloudCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateCloudCluster,
		ReadContext:   awsReadCloudCluster,
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
			keyIsEsType: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     false,
				Description: "Whether this is an ES type cluster. Changing this forces a new resource to be created.",
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
				ForceNew:    true,
				MaxItems:    1,
				Description: "VM configuration for the cluster nodes. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyClusterName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "Unique name to assign to the cloud cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUserEmail: {
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
							Optional:    true,
							ForceNew:    true,
							Description: "DNS name servers for the cluster. Changing this forces a new resource to be created.",
						},
						keyDNSSearchDomain: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							ForceNew:    true,
							Description: "DNS search domains for the cluster. Changing this forces a new resource to be created.",
						},
						keyNtpServers: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							ForceNew:    true,
							Description: "NTP servers for the cluster. Changing this forces a new resource to be created.",
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
							Optional:    true,
							ForceNew:    true,
							Default:     false,
							Description: "Whether to enable immutability for the S3 bucket. Changing this forces a new resource to be created.",
						},
						keyShouldCreateBucket: {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     false,
							Description: "Whether to create the S3 bucket if it does not exist. Changing this forces a new resource to be created.",
						},
						keyEnableObjectLock: {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     false,
							Description: "Whether to enable object lock for the S3 bucket. Changing this forces a new resource to be created.",
						},
						keyKeepClusterOnFailure: {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     false,
							Description: "Whether to keep the cluster on failure. Changing this forces a new resource to be created.",
						},
					},
				},
			},
			keyVmConfig: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "VM configuration for the cluster nodes. Changing this forces a new resource to be created.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyCdmVersion: {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							Description:  "CDM version to use. If not specified, the latest version will be used.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyCdmProduct: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "CDM product to use. If not specified, the latest version will be used.",
						},
						keyUseLatestCdmVersion: {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     false,
							Description: "Whether to use the latest CDM version. Changing this forces a new resource to be created.",
						},
						keyInstanceType: {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "AWS instance type for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlaws.AwsInstanceTypeM5_4XLarge),
								string(gqlaws.AwsInstanceTypeM6I_2XLarge),
								string(gqlaws.AwsInstanceTypeM6I_4XLarge),
								string(gqlaws.AwsInstanceTypeM6I_8XLarge),
								string(gqlaws.AwsInstanceTypeR6I_4XLarge),
								string(gqlaws.AwsInstanceTypeM6A_2XLarge),
								string(gqlaws.AwsInstanceTypeM6A_4XLarge),
								string(gqlaws.AwsInstanceTypeM6A_8XLarge),
								string(gqlaws.AwsInstanceTypeR6A_4XLarge),
							}, false),
						},
						keyInstanceProfileName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS instance profile name for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyVpc: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS VPC ID where the cluster will be deployed. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySubnet: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "AWS subnet ID where the cluster nodes will be deployed. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySecurityGroups: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							ForceNew:    true,
							Description: "AWS security group IDs for the cluster nodes. Changing this forces a new resource to be created.",
						},
						keyVmType: {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Default:     "DENSE",
							Description: "VM type for the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringInSlice([]string{
								string(cloudcluster.CCVmConfigStandard),
								string(cloudcluster.CCVmConfigDense),
								string(cloudcluster.CCVmConfigExtraDense),
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

	vmConfigList := d.Get(keyVmConfig).([]any)
	if len(vmConfigList) == 0 {
		return diag.Errorf("%s is required", keyVmConfig)
	}
	vmConfigMap := vmConfigList[0].(map[string]any)

	securityGroupsSet := vmConfigMap[keySecurityGroups].(*schema.Set)
	securityGroups := make([]string, 0, securityGroupsSet.Len())
	for _, sg := range securityGroupsSet.List() {
		securityGroups = append(securityGroups, sg.(string))
	}

	instanceTypeStr := vmConfigMap[keyInstanceType].(string)
	instanceType := gqlaws.AwsCCInstanceType(instanceTypeStr)
	vmTypeStr := vmConfigMap[keyVmType].(string)
	vmType := cloudcluster.VmConfigType(vmTypeStr)

	clusterConfigMap := d.Get(keyClusterConfig).([]any)[0].(map[string]any)

	dnsNameServers := make([]string, 0)
	if dnsNameServersList, ok := clusterConfigMap[keyDNSNameServers].([]any); ok {
		for _, dns := range dnsNameServersList {
			dnsNameServers = append(dnsNameServers, dns.(string))
		}
	}

	dnsSearchDomains := make([]string, 0)
	if dnsSearchDomainsList, ok := clusterConfigMap[keyDNSSearchDomain].([]any); ok {
		for _, domain := range dnsSearchDomainsList {
			dnsSearchDomains = append(dnsSearchDomains, domain.(string))
		}
	}

	ntpServers := make([]string, 0)
	if ntpServersList, ok := clusterConfigMap[keyNtpServers].([]any); ok {
		for _, ntp := range ntpServersList {
			ntpServers = append(ntpServers, ntp.(string))
		}
	}

	validations := []cloudcluster.ClusterCreateValidations{
		cloudcluster.AllChecks,
	}

	vmConfig := gqlaws.AwsVmConfig{
		CdmVersion:          vmConfigMap[keyCdmVersion].(string),
		InstanceProfileName: vmConfigMap[keyInstanceProfileName].(string),
		InstanceType:        instanceType,
		SecurityGroups:      securityGroups,
		Subnet:              vmConfigMap[keySubnet].(string),
		VmType:              vmType,
		Vpc:                 vmConfigMap[keyVpc].(string),
	}

	awsEsConfig := cloudcluster.AwsEsConfigInput{
		BucketName:         clusterConfigMap[keyBucketName].(string),
		EnableImmutability: clusterConfigMap[keyEnableImmutability].(bool),
		ShouldCreateBucket: clusterConfigMap[keyShouldCreateBucket].(bool),
		EnableObjectLock:   clusterConfigMap[keyEnableObjectLock].(bool),
	}

	clusterConfig := gqlaws.AwsClusterConfig{
		ClusterName:      clusterConfigMap[keyClusterName].(string),
		UserEmail:        clusterConfigMap[keyUserEmail].(string),
		AdminPassword:    clusterConfigMap[keyAdminPassword].(string),
		DnsNameServers:   dnsNameServers,
		DnsSearchDomains: dnsSearchDomains,
		NtpServers:       ntpServers,
		NumNodes:         clusterConfigMap[keyNumNodes].(int),
		AwsEsConfig:      awsEsConfig,
	}

	input := gqlaws.CreateAwsClusterInput{
		CloudAccountID:       cloudAccountID,
		ClusterConfig:        clusterConfig,
		IsEsType:             d.Get(keyIsEsType).(bool),
		KeepClusterOnFailure: clusterConfigMap[keyKeepClusterOnFailure].(bool),
		Region:               d.Get(keyRegion).(string),
		UsePlacementGroups:   d.Get(keyUsePlacementGroups).(bool),
		Validations:          validations,
		VmConfig:             vmConfig,
	}

	useLatestCdmVersion := vmConfig.CdmVersion == "" || d.Get(keyUseLatestCdmVersion).(bool)

	cloudcluster, err := aws.Wrap(client).CreateCloudCluster(ctx, input, useLatestCdmVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cloudcluster.ID.String())
	d.Set(keyCdmProduct, cloudcluster.CdmProduct)
	d.Set(keyInstanceType, cloudcluster.InstanceType)

	return awsReadCloudCluster(ctx, d, m)
}

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

	clusterFilter := cloudcluster.ClusterFilterInput{
		ID: []string{id.String()},
	}

	// Use AllCloudClusters and filter for cluster
	cloudClusters, err := cloudcluster.Wrap(client).AllCloudClusters(ctx, 1, "", clusterFilter, cloudcluster.SortByClusterName, core.SortOrderDesc)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(cloudClusters) == 0 {
		d.SetId("")
		return nil
	}

	cloudCluster := cloudClusters[0]
	//check we have the correct cloud cluster
	if cloudCluster.ID != id {
		return diag.Errorf("Cloud cluster ID mismatch. Expected %q, got %q", id, cloudCluster.ID)
	}

	d.Set(keyClusterName, cloudCluster.Name)
	d.Set(keyCloudAccountID, cloudCluster.CloudInfo.NativeCloudAccountID)
	d.Set(keyRegion, cloudCluster.CloudInfo.Region)
	d.Set(keyCdmVersion, cloudCluster.Version)
	d.Set(keyBucketName, cloudCluster.CloudInfo.StorageConfig.LocationName)
	d.Set(keyEnableImmutability, cloudCluster.CloudInfo.StorageConfig.IsImmutable)

	return nil
}

func awsUpdateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsUpdateCloudCluster")

	// Update is not supported for cloud clusters. This will be implemented in the future.
	return diag.Errorf("Cloud cluster update is not supported. Please remove the resource and create a new one.")
}

func awsDeleteCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteCloudCluster")

	// Cluster Removal is not supported via terraform yet. The user must remove the
	// cluster through the RSC UI. This will be implemented in the future.

	tflog.Warn(ctx, "Cloud cluster deletion should be handled through RSC directly. Removing from Terraform state only.")

	d.SetId("")
	return nil
}
