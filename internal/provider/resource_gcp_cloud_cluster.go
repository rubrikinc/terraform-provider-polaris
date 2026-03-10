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
)

const resourceGCPCloudClusterDescription = `
The ´polaris_gcp_cloud_cluster´ resource creates a GCP cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in GCP using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual GCP infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The GCP project must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.
`

// This resource uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func resourceGcpCloudCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateCloudCluster,
		ReadContext:   gcpReadCloudCluster,
		UpdateContext: gcpUpdateCloudCluster,
		DeleteContext: gcpDeleteCloudCluster,
		Description:   description(resourceGCPCloudClusterDescription),
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
				Description:  "GCP region to deploy the cluster in. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyZone: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "GCP zone to deploy the cluster in. Changing this forces a new resource to be created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyClusterConfig: {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Configuration for the cloud cluster.",
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
							Description:  "Name of the GCS bucket to use for the cluster. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
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
						keyTimezone: {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "Timezone for the cluster using IANA standard format e.g. America/Los_Angeles, Europe/Paris, etc.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyLocation: {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "Location for the cluster. This is free text, RSC will map it to the closest possible location e.g. Palo Alto, CA.",
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
							Description: "GCP instance type for the cluster nodes. Changing this forces a new resource to be created. Supported values are `N2_STANDARD_16`, `N2_STANDARD_32`, `N2_STANDARD_64` and `N2_HIGHMEM_16`.",
							ValidateFunc: validation.StringInSlice([]string{
								"N2_STANDARD_16", "N2_STANDARD_32", "N2_STANDARD_64", "N2_HIGHMEM_16",
							}, false),
						},
						keyNetwork: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "GCP network name for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keySubnet: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "GCP subnet name for the cluster nodes. Changing this forces a new resource to be created.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyHostProject: {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "GCP host project for shared VPC. Changing this forces a new resource to be created.",
						},
						keyServiceAccounts: {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Required:    true,
							ForceNew:    true,
							Description: "GCP service account emails for the cluster nodes. Changing this forces a new resource to be created.",
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
						keyDeleteProtection: {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true,
							Default:     true,
							Description: "Whether to enable delete protection on the GCP instances. Changing this forces a new resource to be created.",
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

// gcpCreateCloudCluster creates the GCP cloud cluster resource.
func gcpCreateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpCreateCloudCluster")

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

	clusterConfigMap := d.Get(keyClusterConfig).([]any)[0].(map[string]any)

	region := d.Get(keyRegion).(string)
	zone := d.Get(keyZone).(string)
	numNodes := clusterConfigMap[keyNumNodes].(int)

	// Extract network configuration
	network := vmConfigMap[keyNetwork].(string)
	subnet := vmConfigMap[keySubnet].(string)
	hostProject := vmConfigMap[keyHostProject].(string)

	// Build networkConfig by replicating the single subnet config for numNodes times
	networkConfig := make([]gqlcloudcluster.GcpSubnetInput, numNodes)
	for i := 0; i < numNodes; i++ {
		networkConfig[i] = gqlcloudcluster.GcpSubnetInput{
			HostProject: hostProject,
			Name:        subnet,
			Network:     network,
			Region:      region,
		}
	}

	// Build serviceAccounts with cloud-platform scope
	serviceAccountsSet := vmConfigMap[keyServiceAccounts].(*schema.Set)
	serviceAccounts := make([]gqlcloudcluster.GcpServiceAccountInput, 0)
	for _, sa := range serviceAccountsSet.List() {
		serviceAccounts = append(serviceAccounts, gqlcloudcluster.GcpServiceAccountInput{
			Email:  sa.(string),
			Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
	}

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

	vmType := gqlcloudcluster.VmConfigType(vmConfigMap[keyVMType].(string))

	gcpEsConfig := gqlcloudcluster.GcpEsConfigInput{
		BucketName:         clusterConfigMap[keyBucketName].(string),
		Region:             region,
		ShouldCreateBucket: false,
	}

	clusterConfig := gqlcloudcluster.GcpClusterConfig{
		ClusterName:      clusterConfigMap[keyClusterName].(string),
		UserEmail:        clusterConfigMap[keyAdminEmail].(string),
		AdminPassword:    secret.String(clusterConfigMap[keyAdminPassword].(string)),
		DNSNameServers:   dnsNameServers,
		DNSSearchDomains: dnsSearchDomains,
		NTPServers:       ntpServers,
		NumNodes:         numNodes,
		GcpEsConfig:      gcpEsConfig,
	}

	vmConfig := gqlcloudcluster.GcpVmConfig{
		CDMVersion:       vmConfigMap[keyCDMVersion].(string),
		InstanceType:     vmConfigMap[keyInstanceType].(string),
		NetworkConfig:    networkConfig,
		ServiceAccounts:  serviceAccounts,
		VMType:           vmType,
		Zone:             zone,
		DeleteProtection: vmConfigMap[keyDeleteProtection].(bool),
	}

	input := gqlcloudcluster.CreateGcpClusterInput{
		CloudAccountID:       cloudAccountID,
		ClusterConfig:        clusterConfig,
		IsEsType:             true,
		KeepClusterOnFailure: clusterConfigMap[keyKeepClusterOnFailure].(bool),
		Validations:          validations,
		VMConfig:             vmConfig,
	}

	gcpCluster, err := cloudcluster.Wrap(client).CreateGcpCloudCluster(ctx, input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(gcpCluster.ID.String())

	// Read back the created resource to populate computed fields. A failed
	// readback must not be returned as an error: the resource was successfully
	// created and returning an error here would leave Terraform unable to
	// manage it. A plan diff on the next run is an acceptable outcome.
	if diags := gcpReadCloudCluster(ctx, d, m); diags.HasError() {
		for _, diagnostic := range diags {
			tflog.Warn(ctx, "failed to read back gcp cloud cluster after create", map[string]any{
				"summary": diagnostic.Summary,
				"detail":  diagnostic.Detail,
			})
		}
	}
	return nil
}

// gcpReadCloudCluster reads the GCP cloud cluster resource.
func gcpReadCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpReadCloudCluster")

	// For cloud clusters, the read operation is limited since the cluster
	// creation is a long-running operation and the cluster state is managed
	// by RSC. We mainly verify that the resource still exists in the state.

	// If the ID is empty, the resource doesn't exist
	if d.Id() == "" {
		return nil
	}

	// Create the gqlapi client
	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	// Get cloud cluster ID
	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Create filter for cloud cluster
	clusterFilter := gqlcluster.SearchFilter{
		ID: []string{id.String()},
	}

	// Use AllCloudClusters and filter for cluster
	cloudClusters, err := gqlcloudcluster.Wrap(client.GQL).AllCloudClusters(ctx, 1, "", clusterFilter, gqlcluster.SortByClusterName, core.SortOrderDesc)
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
	dnsServers, err := gqlcluster.Wrap(client.GQL).DNSServers(ctx, uuid.MustParse(d.Id()))
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

	ntpServers, err := gqlcluster.Wrap(client.GQL).NTPServers(ctx, uuid.MustParse(d.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	ntpServersSet := schema.Set{F: schema.HashString}
	for _, server := range ntpServers {
		ntpServersSet.Add(server.Server)
	}
	clusterConfigMap[keyNTPServers] = &ntpServersSet

	// Read cluster settings
	clusterID, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	clusterSettings, err := gqlcluster.Wrap(client.GQL).ClusterSettings(ctx, clusterID)
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

// gcpDeleteCloudCluster deletes the GCP cloud cluster resource.
func gcpDeleteCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpDeleteCloudCluster")

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

// gcpUpdateCloudCluster updates the resource in-place. The following actions
// are supported:
//   - Update Network DNS
//   - Update Network DNS Search Domains
//   - Update NTP
//   - Update Cluster Name
//   - Update Timezone
//   - Update Location
func gcpUpdateCloudCluster(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "gcpUpdateCloudCluster")

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
		if d.HasChanges(keyClusterConfig+".0."+keyClusterName, keyClusterConfig+".0."+keyTimezone, keyClusterConfig+".0."+keyLocation) {
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

	return gcpReadCloudCluster(ctx, d, m)
}
