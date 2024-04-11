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

func resourceCDMBootstrapCCESAWS() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCDMBootstrapCCESAWSCreate,
		ReadContext:   resourceCDMBootstrapCCESAWSRead,
		UpdateContext: resourceCDMBootstrapCCESAWSUpdate,
		DeleteContext: resourceCDMBootstrapCCESAWSDelete,

		Schema: map[string]*schema.Schema{
			"admin_email": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The Rubrik cluster sends messages for the admin account to this email address.",
				ValidateFunc: validateEmailAddress,
			},
			"admin_password": {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				Description:  "Password for the admin account.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"bucket_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS S3 bucket where CCES will store its data.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"cluster_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Unique name to assign to the Rubrik cluster.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"cluster_nodes": {
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsIPAddress,
				},
				Description: "The node name and IP formatted as a map.",
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
			"enable_immutability": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag to determine if versioning will be used on the S3 object storage to enable immutability.",
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
			"timeout": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "4m",
				Description:  "The time to wait to establish a connection the Rubrik cluster before returning an error (defaults to `4m`).",
				ValidateFunc: validateDuration,
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

func resourceCDMBootstrapCCESAWSCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAWSCreate")

	client, err := m.(*client).cdm()
	if err != nil {
		return diag.FromErr(err)
	}

	var timeout time.Duration
	if t, ok := d.GetOk("timeout"); ok {
		timeout, err = time.ParseDuration(t.(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	config := toClusterConfig(d)
	config.StorageConfig = cdm.AWSStorageConfig{
		BucketName:         d.Get("bucket_name").(string),
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
	return resourceCDMBootstrapCCESAWSRead(ctx, d, m)
}

func resourceCDMBootstrapCCESAWSRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAWSRead")

	client, err := m.(*client).cdm()
	if err != nil {
		return diag.FromErr(err)
	}

	var timeout time.Duration
	if t, ok := d.GetOk("timeout"); ok {
		timeout, err = time.ParseDuration(t.(string))
		if err != nil {
			return diag.FromErr(err)
		}
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
func resourceCDMBootstrapCCESAWSUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAWSUpdate")
	return resourceCDMBootstrapCCESAWSRead(ctx, d, m)
}

// Once a Cluster has been bootstrapped it can not be deleted.
func resourceCDMBootstrapCCESAWSDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] resourceCDMBootstrapCCESAWSDelete")
	return nil
}
