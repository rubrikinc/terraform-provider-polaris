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
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	gqlarchival "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/archival"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core/secret"
)

const resourceDataCenterArchivalLocationAmazonS3Description = `
The ´polaris_data_center_archival_location_amazon_s3´ resource create a data
center archival location with the Amazon S3 storage type.

~> Before configuring the immutability settings, see
   [KB article](https://support.rubrik.com/s/article/000005468) or the Rubrik
   User Guide documentation to determine the proper immutability lock period.

-> More information about AWS Immutable Storage and time-based retention locks
   can be found in the Amazon AWS documentation.
`

func resourceDataCenterArchivalLocationAmazonS3() *schema.Resource {
	return &schema.Resource{
		CreateContext: dataCenterCreateArchivalLocationAmazonS3,
		ReadContext:   dataCenterReadArchivalLocationAmazonS3,
		UpdateContext: dataCenterUpdateArchivalLocationAmazonS3,
		DeleteContext: dataCenterDeleteArchivalLocationAmazonS3,

		Description: description(resourceDataCenterArchivalLocationAmazonS3Description),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Data center archival location ID (UUID).",
			},
			keyArchivalProxySettings: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyBypassProxy: {
							Type:         schema.TypeBool,
							Optional:     true,
							ExactlyOneOf: []string{keyArchivalProxySettings + ".0." + keyProxyServer},
							Description: "When true, the system proxy will not be used to route the archival " +
								"requests and data.",
						},
						keyPassword: {
							Type:          schema.TypeString,
							Optional:      true,
							Sensitive:     true,
							ConflictsWith: []string{keyArchivalProxySettings + ".0." + keyBypassProxy},
							RequiredWith:  []string{keyArchivalProxySettings + ".0." + keyUsername},
							Description:   "Proxy password.",
							ValidateFunc:  validation.StringIsNotWhiteSpace,
						},
						keyPortNumber: {
							Type:     schema.TypeInt,
							Optional: true,
							RequiredWith: []string{
								keyArchivalProxySettings + ".0." + keyProtocol,
								keyArchivalProxySettings + ".0." + keyProxyServer,
							},
							Description:  "Proxy port number.",
							ValidateFunc: validation.IsPortNumber,
						},
						keyProtocol: {
							Type:     schema.TypeString,
							Optional: true,
							RequiredWith: []string{
								keyArchivalProxySettings + ".0." + keyPortNumber,
								keyArchivalProxySettings + ".0." + keyProxyServer,
							},
							Description:  "Proxy protocol. Possible values are `HTTP`, `HTTPS` and `SOCKS5`.",
							ValidateFunc: validation.StringInSlice([]string{"HTTP", "HTTPS", "SOCKS5"}, false),
						},
						keyProxyServer: {
							Type:     schema.TypeString,
							Optional: true,
							ExactlyOneOf: []string{
								keyArchivalProxySettings + ".0." + keyBypassProxy,
							},
							RequiredWith: []string{
								keyArchivalProxySettings + ".0." + keyPortNumber,
								keyArchivalProxySettings + ".0." + keyProtocol,
							},
							Description:  "Proxy server IP address or FQDN.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUsername: {
							Type:          schema.TypeString,
							Optional:      true,
							Description:   "Proxy username.",
							ConflictsWith: []string{keyArchivalProxySettings + ".0." + keyBypassProxy},
							RequiredWith:  []string{keyArchivalProxySettings + ".0." + keyPassword},
							ValidateFunc:  validation.StringIsNotWhiteSpace,
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Archival proxy settings will be used to route the archival data and requests.",
			},
			keyBucketName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS bucket name.",
				ValidateFunc: validation.StringLenBetween(3, 63),
			},
			keyCloudAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "RSC data center cloud account ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyCloudComputeSettings: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalConsolidation: {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							Description: "When true, archival consolidation is enabled. Archival consolidation " +
								"frees up storage. Default value is `false`.",
						},
						keySecurityGroupID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "AWS security group ID.",
						},
						keySubnetID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "AWS subnet ID.",
						},
						keyVPCID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "AWS VPC ID.",
						},
					},
				},
				MaxItems:    1,
				Optional:    true,
				Description: "Cloud compute settings will be used to launch a temporary instance in the AWS account.",
			},
			keyClusterID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Rubrik cluster ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyComputeProxySettings: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyPassword: {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							RequiredWith: []string{keyComputeProxySettings + ".0." + keyUsername},
							Description:  "Proxy password.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyPortNumber: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Proxy port number.",
							ValidateFunc: validation.IsPortNumber,
						},
						keyProtocol: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Proxy protocol. Possible values are `HTTP`, `HTTPS` and `SOCKS5`.",
							ValidateFunc: validation.StringInSlice([]string{"HTTP", "HTTPS", "SOCKS5"}, false),
						},
						keyProxyServer: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Proxy server IP address or FQDN.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyUsername: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Proxy username.",
							RequiredWith: []string{keyComputeProxySettings + ".0." + keyPassword},
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				Description: "Compute proxy settings will be used to make API calls for instantiating virtual " +
					"machines.",
			},
			keyEncryptionPassword: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{keyKMSMasterKey, keyRSAKey},
				Description: "Encryption password. Password encryption is available only for immutable archival " +
					"locations.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyEndpointSettings: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyS3Endpoint: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "AWS S3 endpoint.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						keyKMSEndpoint: {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{keyEncryptionPassword, keyRSAKey},
							Description: "AWS KMS endpoint. A KMS endpoint can only be specified when a KMS key is " +
								"used for encryption.",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				Description: "Endpoint settings will be used to specify dedicated VPC endpoints when archiving to " +
					"AWS S3 to leverage the AWS PrivateLink feature. The default region-based endpoint will be used " +
					"if no endpoint is specified.",
			},
			keyImmutabilitySettings: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLockPeriod: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Immutability lock period (days).",
						},
					},
				},
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{keyKMSMasterKey, keyRSAKey},
				Description: "Enables immutable storage with a time-based retention lock using the AWS immutability " +
					"feature for your archival location. Once enabled, you cannot delete the snapshots in this " +
					"archival location before the specified immutability lock period expires. Requires an encryption " +
					"password policy.",
			},
			keyKMSMasterKey: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyEncryptionPassword, keyRSAKey},
				Description:  "AWS KMS master key ID. Cannot be used with immutable archival locations.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Data center archival location name.",
				ValidateFunc: validation.StringLenBetween(1, 255),
			},
			keyRegion: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS region.",
				ValidateFunc: validation.StringInSlice(aws.AllRegionNames(), false),
			},
			keyRetrievalTier: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD_TIER",
				Description: "AWS bucket retrieval tier. Determines the speed and cost of retrieving data from " +
					"the Glacier and Glacier Flexible Retrieval storage classes. Possible values are `BULK_TIER`, " +
					"`EXPEDITED_TIER` and `STANDARD_TIER`. Default value is `STANDARD_TIER`.",
				ValidateFunc: validation.StringInSlice([]string{"BULK_TIER", "EXPEDITED_TIER", "STANDARD_TIER"}, false),
			},
			keyRSAKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{keyEncryptionPassword, keyKMSMasterKey},
				Description:  "PEM encoded private RSA key. Cannot be used with immutable archival locations.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyStorageClass: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "STANDARD",
				Description:  "AWS bucket storage class. Possible values are `STANDARD`, `STANDARD_IA` and `ONEZONE_IA`. Default value is `STANDARD`.",
				ValidateFunc: validation.StringInSlice([]string{"STANDARD", "STANDARD_IA", "ONEZONE_IA"}, false),
			},
			keyStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of data center archival location.",
			},
			keySyncStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Synchronization status of AWS target.",
			},
		},
	}
}

func dataCenterCreateArchivalLocationAmazonS3(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterCreateArchivalLocationAmazonS3")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := uuid.Parse(d.Get(keyClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	cloudComputeSettings, archiveConsolidation := fromCloudComputeSettings(d)
	archivalProxySettings, bypassProxy := fromArchivalProxySettings(d)
	s3Endpoint, kmsEndpoint := fromEndpointSettings(d)
	id, err := archival.Wrap(client).CreateAWSTarget(ctx, gqlarchival.CreateAWSTargetParams{
		Name:                   d.Get(keyName).(string),
		ClusterID:              clusterID,
		CloudAccountID:         cloudAccountID,
		BucketName:             d.Get(keyBucketName).(string),
		Region:                 aws.RegionFromName(d.Get(keyRegion).(string)).ToRegionEnum(),
		StorageClass:           d.Get(keyStorageClass).(string),
		RetrievalTier:          d.Get(keyRetrievalTier).(string),
		KMSMasterKeyID:         d.Get(keyKMSMasterKey).(string),
		RSAKey:                 secret.String(d.Get(keyRSAKey).(string)),
		EncryptionPassword:     secret.String(d.Get(keyEncryptionPassword).(string)),
		CloudComputeSettings:   cloudComputeSettings,
		IsConsolidationEnabled: archiveConsolidation,
		ProxySettings:          archivalProxySettings,
		BypassProxy:            bypassProxy,
		ComputeProxySettings:   fromComputeProxySettings(d),
		ImmutabilitySettings:   fromImmutabilitySettings(d),
		S3Endpoint:             s3Endpoint,
		KMSEndpoint:            kmsEndpoint,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())
	dataCenterReadArchivalLocationAmazonS3(ctx, d, m)
	return nil
}

func dataCenterReadArchivalLocationAmazonS3(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterReadArchivalLocationAmazonS3")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	target, err := archival.Wrap(client).AWSTargetByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyName, target.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyClusterID, target.Cluster.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStatus, target.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySyncStatus, target.SyncStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudAccountID, target.CloudAccount.ID.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBucketName, target.Bucket); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRegion, target.Region.Name()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyStorageClass, target.StorageClass); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRetrievalTier, target.RetrivalTier); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyKMSMasterKey, target.KMSMasterKeyID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyArchivalProxySettings, toArchivalProxySettings(target)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyCloudComputeSettings, toCloudComputeSettings(target)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyComputeProxySettings, toComputeProxySettings(target)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyEndpointSettings, toEndpointSettings(target)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyImmutabilitySettings, toImmutabilitySettings(target)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterUpdateArchivalLocationAmazonS3(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterUpdateArchivalLocationAmazonS3")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cloudAccountID, err := uuid.Parse(d.Get(keyCloudAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	cloudComputeSettings, archiveConsolidation := fromCloudComputeSettings(d)
	archivalProxySettings, bypassProxy := fromArchivalProxySettings(d)
	s3Endpoint, kmsEndpoint := fromEndpointSettings(d)
	err = archival.Wrap(client).UpdateAWSTarget(ctx, id, gqlarchival.UpdateAWSTargetParams{
		Name:                   d.Get(keyName).(string),
		CloudAccountID:         cloudAccountID,
		StorageClass:           d.Get(keyStorageClass).(string),
		RetrievalTier:          d.Get(keyRetrievalTier).(string),
		CloudComputeSettings:   cloudComputeSettings,
		IsConsolidationEnabled: archiveConsolidation,
		ProxySettings:          archivalProxySettings,
		BypassProxy:            bypassProxy,
		ComputeProxySettings:   fromComputeProxySettings(d),
		ImmutabilitySettings:   fromImmutabilitySettings(d),
		S3Endpoint:             s3Endpoint,
		KMSEndpoint:            kmsEndpoint,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func dataCenterDeleteArchivalLocationAmazonS3(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "dataCenterDeleteArchivalLocationAmazonS3")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := archival.Wrap(client).DeleteTarget(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// fromArchivalProxySettings extracts the archival proxy settings data from
// the resource configuration.
func fromArchivalProxySettings(d *schema.ResourceData) (*gqlarchival.AWSTargetProxySettings, bool) {
	data, ok := d.GetOk(keyArchivalProxySettings)
	if !ok {
		return nil, false
	}

	settings := data.([]any)[0].(map[string]any)
	return &gqlarchival.AWSTargetProxySettings{
		Username:    settings[keyUsername].(string),
		Password:    secret.String(settings[keyPassword].(string)),
		ProxyServer: settings[keyProxyServer].(string),
		Protocol:    settings[keyProtocol].(string),
		PortNumber:  settings[keyPortNumber].(int),
	}, settings[keyBypassProxy].(bool)
}

// toArchivalProxySettings extracts the archival proxy settings data from the
// target.
func toArchivalProxySettings(target gqlarchival.AWSTarget) []any {
	if target.ProxySettings == nil {
		return nil
	}

	return []any{
		map[string]any{
			keyPortNumber:  target.ProxySettings.PortNumber,
			keyProtocol:    target.ProxySettings.Protocol,
			keyProxyServer: target.ProxySettings.ProxyServer,
			keyUsername:    target.ProxySettings.Username,
			keyBypassProxy: target.BypassProxy,
		},
	}
}

// fromCloudComputeSettings extracts the cloud compute settings data from the
// resource configuration.
func fromCloudComputeSettings(d *schema.ResourceData) (*gqlarchival.AWSTargetCloudComputeSettings, bool) {
	data, ok := d.GetOk(keyArchivalProxySettings)
	if !ok {
		return nil, false
	}

	settings := data.([]any)[0].(map[string]any)
	return &gqlarchival.AWSTargetCloudComputeSettings{
		VPCID:           settings[keyVPCID].(string),
		SubnetID:        settings[keySubnetID].(string),
		SecurityGroupID: settings[keySecurityGroupID].(string),
	}, settings[keyArchivalConsolidation].(bool)
}

// toCloudComputeSettings extracts the cloud compute settings data from the
// target.
func toCloudComputeSettings(target gqlarchival.AWSTarget) []any {
	if target.ComputeSettings == nil {
		return nil
	}

	return []any{
		map[string]any{
			keySecurityGroupID:       target.ComputeSettings.SecurityGroupID,
			keySubnetID:              target.ComputeSettings.SubnetID,
			keyVPCID:                 target.ComputeSettings.VPCID,
			keyArchivalConsolidation: target.IsConsolidationEnabled,
		},
	}
}

// fromComputeProxySettings extracts the compute proxy settings data from the
// resource configuration.
func fromComputeProxySettings(d *schema.ResourceData) *gqlarchival.AWSTargetProxySettings {
	data, ok := d.GetOk(keyArchivalProxySettings)
	if !ok {
		return nil
	}

	settings := data.([]any)[0].(map[string]any)
	return &gqlarchival.AWSTargetProxySettings{
		Username:    settings[keyUsername].(string),
		Password:    secret.String(settings[keyPassword].(string)),
		ProxyServer: settings[keyProxyServer].(string),
		Protocol:    settings[keyProtocol].(string),
		PortNumber:  settings[keyPortNumber].(int),
	}
}

// toComputeProxySettings extracts the compute proxy settings data from the
// target.
func toComputeProxySettings(target gqlarchival.AWSTarget) []any {
	if target.ComputeSettings == nil || target.ComputeSettings.ProxySettings == nil {
		return nil
	}

	return []any{
		map[string]any{
			keyPortNumber:  target.ComputeSettings.ProxySettings.PortNumber,
			keyProtocol:    target.ComputeSettings.ProxySettings.Protocol,
			keyProxyServer: target.ComputeSettings.ProxySettings.ProxyServer,
			keyUsername:    target.ComputeSettings.ProxySettings.Username,
		},
	}
}

// fromEndpointSettings extracts the endpoint settings data from the resource
// configuration.
func fromEndpointSettings(d *schema.ResourceData) (string, string) {
	data, ok := d.GetOk(keyImmutabilitySettings)
	if !ok {
		return "", ""
	}

	settings := data.([]any)[0].(map[string]any)
	return settings[keyS3Endpoint].(string), settings[keyKMSEndpoint].(string)
}

// toEndpointSettings extracts the endpoint settings data from the target.
func toEndpointSettings(target gqlarchival.AWSTarget) []any {
	if target.S3Endpoint == "" && target.KMSEndpoint == "" {
		return nil
	}

	return []any{
		map[string]any{
			keyS3Endpoint:  target.S3Endpoint,
			keyKMSEndpoint: target.KMSEndpoint,
		},
	}
}

// fromImmutabilitySettings extracts the immutability settings data from the
// resource configuration.
func fromImmutabilitySettings(d *schema.ResourceData) *gqlarchival.AWSTargetImmutabilitySettings {
	data, ok := d.GetOk(keyImmutabilitySettings)
	if !ok {
		return nil
	}

	settings := data.([]any)[0].(map[string]any)
	return &gqlarchival.AWSTargetImmutabilitySettings{
		LockDurationDays: settings[keyLockPeriod].(int),
	}
}

// toImmutabilitySettings extracts the immutability settings data from the
// target.
func toImmutabilitySettings(target gqlarchival.AWSTarget) []any {
	if target.ImmutabilitySettings == nil {
		return nil
	}

	return []any{
		map[string]any{
			keyLockPeriod: target.ImmutabilitySettings.LockDurationDays,
		},
	}
}
