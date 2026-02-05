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
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"
)

const resourceAWSExocomputeDescription = `
The ´polaris_aws_exocompute´ resource creates an RSC Exocompute configuration
for AWS workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When an RSC managed host configuration is created, RSC
    will automatically deploy the necessary resources in the specified AWS
    region to run the Exocompute service. AWS security groups can be managed by
    RSC or by the customer.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the AWS EKS
    cluster attached by the customer, using the
    ´polaris_aws_exocompute_cluster_attachment´ resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Items 1 and 2 above requires that the AWS account has been onboarded with the
´EXOCOMPUTE´ feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a ´polaris_aws_exocompute´ resource:
 1. Using the ´account_id´, ´region´, ´vpc_id´ and ´subnets´ or ´subnet´ fields
    creates an RSC managed host configuration. Use the ´subnet´ block when pod
    subnets are needed. The ´cluster_security_group_id´ and
    ´node_security_group_id´ fields can be used to create an Exocompute
    configuration where the customer manage the security groups.
 2. Using the ´account_id´ and ´region´ fields creates a customer managed host
    configuration. Note, the ´polaris_aws_exocompute_cluster_attachment´
    resource must be used to attach an AWS EKS cluster to the Exocompute
    configuration.
 3. Using the ´account_id´ and ´host_cloud_account_id´ fields creates an
    application configuration.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.
`

// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceAwsExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateExocompute,
		ReadContext:   awsReadExocompute,
		DeleteContext: awsDeleteExocompute,

		Description: description(resourceAWSExocomputeDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Exocompute configuration ID (UUID).",
			},
			keyAccountID: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyClusterSecurityGroupID: {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID},
				RequiredWith:  []string{keyNodeSecurityGroupID},
				Description: "AWS security group ID for the cluster. Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyHostAccountID: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{keyHostAccountID, keyRegion},
				Description:  "Exocompute host cloud account ID. Changing this forces a new resource to be created.",
				ValidateFunc: validation.IsUUID,
			},
			keyNodeSecurityGroupID: {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID},
				RequiredWith:  []string{keyClusterSecurityGroupID},
				Description: "AWS security group ID for the nodes. Changing this forces a new resource to be " +
					"created.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPolarisManaged: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "If true the security groups are managed by RSC.",
			},
			keyClusterAccess: {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID, keyClusterSecurityGroupID, keyNodeSecurityGroupID},
				RequiredWith:  []string{keyVPCID},
				Description: "EKS cluster access type. Possible values are " +
					"`EKS_CLUSTER_ACCESS_TYPE_PUBLIC` and `EKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Can only be used with " +
					"RSC managed configurations.",
				ValidateFunc: validation.StringInSlice([]string{
					string(gqlexocompute.EKSClusterAccessPrivate),
					string(gqlexocompute.EKSClusterAccessPublic),
				}, false),
			},
			keyRegion: {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				AtLeastOneOf:  []string{keyHostAccountID, keyRegion},
				ConflictsWith: []string{"host_account_id"},
				Description: "AWS region to run the Exocompute instance in. Changing this forces a new resource " +
					"to be created.",
				ValidateFunc: validation.StringInSlice(gqlaws.AllRegionNames(), false),
			},
			keySubnet: {
				Type:          schema.TypeSet,
				Elem:          awsSubnetResource(),
				MinItems:      2,
				MaxItems:      2,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID, keySubnets},
				RequiredWith:  []string{keyVPCID},
				Description: "AWS subnet for the cluster. Each subnet block accepts a `subnet_id` and an optional " +
					"`pod_subnet_id`. Changing this forces a new resource to be created.",
			},
			keySubnets: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				MinItems:      2,
				MaxItems:      2,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID, keySubnet},
				RequiredWith:  []string{keyVPCID},
				Description: "AWS subnet IDs for the cluster subnets. Changing this forces a new resource to be " +
					"created.",
			},
			keyVPCID: {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{keyHostAccountID},
				Description:   "AWS VPC ID for the cluster network. Changing this forces a new resource to be created.",
				ValidateFunc:  validation.StringIsNotWhiteSpace,
			},
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, m any) error {
			if _, ok := diff.GetOk(keyVPCID); ok {
				subnet := diff.Get(keySubnet).(*schema.Set)
				subnets := diff.Get(keySubnets).(*schema.Set)
				if subnet.Len() == 0 && subnets.Len() == 0 {
					return errors.New("vpc_id requires either subnet or subnets to be specified")
				}
			}
			return nil
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func awsCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsCreateExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	accountID, err := uuid.Parse(d.Get(keyAccountID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if host, ok := d.GetOk(keyHostAccountID); ok {
		hostID, err := uuid.Parse(host.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		err = exocompute.Wrap(client).MapAWSCloudAccount(ctx, accountID, hostID)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId("app-" + accountID.String())
	} else {
		clusterSecurityGroupID := d.Get(keyClusterSecurityGroupID).(string)
		nodeSecurityGroupID := d.Get(keyNodeSecurityGroupID).(string)
		region := d.Get(keyRegion).(string)
		vpcID := d.Get(keyVPCID).(string)
		clusterAccess := d.Get(keyClusterAccess).(string)

		// Read subnets from whichever field is set.
		var subnets []gqlexocompute.AWSSubnet
		if v := d.Get(keySubnet).(*schema.Set); v.Len() > 0 {
			for _, s := range v.List() {
				subnets = append(subnets, fromAWSSubnetResource(s.(map[string]any)))
			}
		} else {
			for _, s := range d.Get(keySubnets).(*schema.Set).List() {
				subnets = append(subnets, gqlexocompute.AWSSubnet{ID: s.(string)})
			}
		}

		var config exocompute.AWSConfig
		switch {
		case region != "" && vpcID != "" && len(subnets) > 0:
			cfg := exocompute.AWSConfigParams{
				Region:                 gqlaws.RegionFromName(region),
				VPCID:                  vpcID,
				Subnets:                subnets,
				ClusterSecurityGroupID: clusterSecurityGroupID,
				NodeSecurityGroupID:    nodeSecurityGroupID,
			}
			if clusterAccess != "" {
				cfg.OptionalConfig = &gqlexocompute.AWSOptionalConfig{
					AWSClusterAccess: gqlexocompute.AWSClusterAccess(clusterAccess),
				}
			}
			config = cfg
		case region != "":
			config = exocompute.AWSConfigParams{Region: gqlaws.RegionFromName(region)}
		default:
			return diag.Errorf("invalid exocompute configuration")
		}

		id, err := exocompute.Wrap(client).AddAWSConfiguration(ctx, accountID, config)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(id.String())
	}

	awsReadExocompute(ctx, d, m)
	return nil
}

func awsReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsReadExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	if strings.HasPrefix(d.Id(), appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}
		hostID, err := exocompute.Wrap(client).AWSHostCloudAccount(ctx, appID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyAccountID, appID.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyHostAccountID, hostID.String()); err != nil {
			return diag.FromErr(err)
		}
	} else {
		configID, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		exoConfig, err := exocompute.Wrap(client).AWSConfigurationByID(ctx, configID)
		if errors.Is(err, graphql.ErrNotFound) {
			d.SetId("")
			return nil
		}
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyAccountID, exoConfig.CloudAccountID.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyRegion, exoConfig.Region.Name()); err != nil {
			return diag.FromErr(err)
		}

		// Rubrik managed cluster
		if err := d.Set(keyClusterSecurityGroupID, exoConfig.ClusterSecurityGroupID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyNodeSecurityGroupID, exoConfig.NodeSecurityGroupID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyPolarisManaged, exoConfig.IsManagedByRubrik); err != nil {
			return diag.FromErr(err)
		}
		// Populate keySubnets (string set of IDs).
		stringSubnets := schema.Set{F: schema.HashString}
		if exoConfig.Subnet1.ID != "" {
			stringSubnets.Add(exoConfig.Subnet1.ID)
		}
		if exoConfig.Subnet2.ID != "" {
			stringSubnets.Add(exoConfig.Subnet2.ID)
		}
		if err := d.Set(keySubnets, &stringSubnets); err != nil {
			return diag.FromErr(err)
		}

		// Populate keySubnet (block set with full details including
		// pod_subnet_id).
		subnetSet := schema.Set{F: schema.HashResource(awsSubnetResource())}
		if exoConfig.Subnet1.ID != "" {
			subnetSet.Add(toAWSSubnetResource(exoConfig.Subnet1))
		}
		if exoConfig.Subnet2.ID != "" {
			subnetSet.Add(toAWSSubnetResource(exoConfig.Subnet2))
		}
		if err := d.Set(keySubnet, &subnetSet); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(keyVPCID, exoConfig.VPCID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(keyClusterAccess, awsClusterAccess(exoConfig.OptionalConfig)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func awsDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Trace(ctx, "awsDeleteExocompute")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	if strings.HasPrefix(id, appCloudAccountPrefix) {
		appID, err := uuid.Parse(strings.TrimPrefix(id, appCloudAccountPrefix))
		if err != nil {
			return diag.FromErr(err)
		}
		if err = exocompute.Wrap(client).UnmapAWSCloudAccount(ctx, appID); err != nil {
			return diag.FromErr(err)
		}
	} else {
		configID, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		if err = exocompute.Wrap(client).RemoveAWSConfiguration(ctx, configID); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// awsClusterAccess returns the cluster access type from the optional config.
// Returns the default public access type if the config is nil.
func awsClusterAccess(config *gqlexocompute.AWSOptionalConfig) string {
	if config == nil {
		return string(gqlexocompute.EKSClusterAccessPublic)
	}
	return string(config.AWSClusterAccess)
}

func awsSubnetResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			keySubnetID: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "AWS subnet ID.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyPodSubnetID: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "AWS subnet ID for the pods.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func fromAWSSubnetResource(block map[string]any) gqlexocompute.AWSSubnet {
	return gqlexocompute.AWSSubnet{
		ID:          block[keySubnetID].(string),
		PodSubnetID: block[keyPodSubnetID].(string),
	}
}

func toAWSSubnetResource(subnet gqlexocompute.AWSSubnet) map[string]any {
	return map[string]any{
		keySubnetID:    subnet.ID,
		keyPodSubnetID: subnet.PodSubnetID,
	}
}
