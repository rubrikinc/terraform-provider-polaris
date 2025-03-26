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
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/cdm"
)

// This resource uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func resourceCDMBootstrapCCESAzure() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCDMBootstrapCCESAzureCreate,
		ReadContext:   resourceCDMBootstrapCCESAzureRead,
		UpdateContext: resourceCDMBootstrapCCESAzureUpdate,
		DeleteContext: resourceCDMBootstrapCCESAzureDelete,

		Schema: map[string]*schema.Schema{
			keyAdminEmail: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The Rubrik cluster sends messages for the admin account to this email address.",
				ValidateFunc: validateEmailAddress,
			},
			keyAdminPassword: {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				Description:  "Password for the admin account.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"cluster_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Unique name to assign to the Rubrik cluster.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyClusterNodes: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsIPAddress,
				},
				ExactlyOneOf: []string{keyNodeConfig},
				Description:  "The node name and IP formatted as a map.",
			},
			"connection_string": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The connection string for the Azure storage account where CCES will store its data.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"container_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the container in the Azure storage account where CCES will store its data.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"dns_name_servers": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsIPv4Address,
				},
				MinItems:    1,
				Description: "IPv4 addresses of DNS servers.",
			},
			"dns_search_domain": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:    1,
				Description: "The search domain that the DNS Service will use to resolve hostnames that are not fully qualified.",
			},
			keyEnableEncryption: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When bootstrapping a Cloud Cluster this value must be `false`. Only kept for backwards compatibility. ",
				Deprecated:  "Not used. Only kept for backwards compatibility.",
			},
			"enable_immutability": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag to determine if versioning will be used on the Azure Blob storage to enable immutability.",
			},
			"management_gateway": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "IP address assigned to the management network gateway",
				ValidateFunc: validation.IsIPAddress,
			},
			"management_subnet_mask": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Subnet mask assigned to the management network.",
				ValidateFunc: validation.IsIPAddress,
			},
			keyNodeConfig: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsIPAddress,
				},
				Description: "The node name and IP formatted as a map.",
				Deprecated:  "Use `cluster_nodes` instead. Only kept for backwards compatibility.",
			},
			"ntp_server1_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "IP address for NTP server #1.",
				ValidateFunc: validation.IsIPAddress,
			},
			"ntp_server1_key": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ntp_server1_key_id", "ntp_server1_key_type"},
				Description:  "Symmetric key material for NTP server #1.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"ntp_server1_key_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				RequiredWith: []string{"ntp_server1_key", "ntp_server1_key_type"},
				Description:  "Key id number for NTP server #1 (typically this is 0).",
			},
			"ntp_server1_key_type": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ntp_server1_key", "ntp_server1_key_id"},
				Description:  "Symmetric key type for NTP server #1.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"ntp_server2_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "IP address for NTP server #2.",
				ValidateFunc: validation.IsIPAddress,
			},
			"ntp_server2_key": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ntp_server2_key_id", "ntp_server2_key_type"},
				Description:  "Symmetric key material for NTP server #2.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"ntp_server2_key_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				RequiredWith: []string{"ntp_server2_key", "ntp_server2_key_type"},
				Description:  "Key id number for NTP server #2 (typically this is 1).",
			},
			"ntp_server2_key_type": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ntp_server2_key", "ntp_server2_key_id"},
				Description:  "Symmetric key type for NTP server #2.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyTimeout: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The time to wait to establish a connection the Rubrik cluster before returning an error (defaults to `4m`).",
				ValidateFunc: validateBackwardsCompatibleTimeout,
			},
			"wait_for_completion": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Flag to determine if Terraform should wait for the bootstrap process to complete.",
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(60 * time.Minute),
			Read:    schema.DefaultTimeout(20 * time.Minute),
			Default: schema.DefaultTimeout(20 * time.Minute),
		},
	}
}

func resourceCDMBootstrapCCESAzureCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAzureCreate")

	client := cdm.NewBootstrapClientWithLogger(true, m.(*client).logger)

	timeout, err := toBackwardsCompatibleTimeout(d)
	if err != nil {
		return diag.FromErr(err)
	}

	config := toClusterConfig(d)
	config.StorageConfig = cdm.AzureStorageConfig{
		ConnectionString:   d.Get("connection_string").(string),
		ContainerName:      d.Get("container_name").(string),
		EnableImmutability: d.Get("enable_immutability").(bool),
	}
	if len(config.ClusterNodes) == 0 {
		return diag.Errorf("At least one cluster node is required")
	}
	nodeIP := config.ClusterNodes[0].ManagementIP
	requestID, err := client.BootstrapCluster(ctx, nodeIP, config, timeout)
	if err != nil {
		return diag.FromErr(err)
	}
	if d.Get("wait_for_completion").(bool) {
		if err := client.WaitForBootstrap(ctx, nodeIP, requestID, timeout); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(d.Get("cluster_name").(string))
	return resourceCDMBootstrapCCESAzureRead(ctx, d, m)
}

func resourceCDMBootstrapCCESAzureRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAzureRead")

	client := cdm.NewBootstrapClientWithLogger(true, m.(*client).logger)

	timeout, err := toBackwardsCompatibleTimeout(d)
	if err != nil {
		return diag.FromErr(err)
	}

	config := toClusterConfig(d)
	if len(config.ClusterNodes) == 0 {
		return diag.Errorf("At least one cluster node is required")
	}
	nodeIP := config.ClusterNodes[0].ManagementIP
	isBootstrapped, err := client.IsBootstrapped(ctx, nodeIP, timeout)
	if err != nil {
		return diag.FromErr(err)
	}
	if !isBootstrapped {
		d.SetId("")
	}

	return nil
}

// Once a Cluster has been bootstrapped it can not be updated through the
// bootstrap resource
func resourceCDMBootstrapCCESAzureUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAzureUpdate")
	return resourceCDMBootstrapCCESAzureRead(ctx, d, m)
}

// Once a Cluster has been bootstrapped it cannot be un-bootstrapped, delete
// simply removes the resource from the local state.
func resourceCDMBootstrapCCESAzureDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAzureDelete")
	d.SetId("")
	return nil
}
