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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceAwsExocompute defines the schema for the AWS exocompute resource.
func resourceAwsExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: awsCreateExocompute,
		ReadContext:   awsReadExocompute,
		DeleteContext: awsDeleteExocompute,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "RSC account id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"cluster_security_group_id": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				RequiredWith:     []string{"node_security_group_id"},
				Description:      "AWS security group id for the cluster.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"node_security_group_id": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				RequiredWith:     []string{"cluster_security_group_id"},
				Description:      "AWS security group id for the nodes.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"polaris_managed": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "If true the security groups are managed by Polaris.",
			},
			"region": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "AWS region to run the exocompute instance in.",
				ValidateDiagFunc: validateAwsRegion,
			},
			"subnets": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems:    2,
				MaxItems:    2,
				Required:    true,
				ForceNew:    true,
				Description: "AWS subnet ids for the cluster subnets.",
			},
			"vpc_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "AWS VPC id for the cluster network.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

// awsCreateExocompute run the Create operation for the AWS exocompute
// resource. This enables the exocompute feature and adds an exocompute config
// to the Polaris cloud account.
func awsCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsCreateExocompute")

	client := m.(*polaris.Client)

	accountID, err := uuid.Parse(d.Get("account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	account, err := aws.Wrap(client).Account(ctx, aws.CloudAccountID(accountID), core.FeatureExocompute)
	if err != nil {
		return diag.FromErr(err)
	}
	cnpFeature, ok := account.Feature(core.FeatureExocompute)
	if !ok {
		return diag.Errorf("exocompute not enabled on account")
	}

	region := d.Get("region").(string)
	if !cnpFeature.HasRegion(region) {
		return diag.Errorf("region %q not available with exocompute feature", region)
	}

	vpcID := d.Get("vpc_id").(string)

	var subnets []string
	for _, s := range d.Get("subnets").(*schema.Set).List() {
		subnets = append(subnets, s.(string))
	}

	clusterSecurityGroupID := d.Get("cluster_security_group_id").(string)
	nodeSecurityGroupID := d.Get("node_security_group_id").(string)

	var config aws.ExoConfigFunc
	if clusterSecurityGroupID == "" || nodeSecurityGroupID == "" {
		config = aws.Managed(region, vpcID, subnets)
	} else {
		config = aws.Unmanaged(region, vpcID, subnets, clusterSecurityGroupID, nodeSecurityGroupID)
	}

	id, err := aws.Wrap(client).AddExocomputeConfig(ctx, aws.CloudAccountID(accountID), config)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id.String())

	awsReadExocompute(ctx, d, m)
	return nil
}

// awsReadExocompute run the Read operation for the AWS exocompute resource.
// This reads the state of the exocompute config in Polaris.
func awsReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsReadExocompute")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	exoConfig, err := aws.Wrap(client).ExocomputeConfig(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("region", exoConfig.Region); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vpc_id", exoConfig.VPCID); err != nil {
		return diag.FromErr(err)
	}

	subnets := schema.Set{F: schema.HashString}
	for _, subnet := range exoConfig.Subnets {
		subnets.Add(subnet.ID)
	}
	if err := d.Set("subnets", &subnets); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("polaris_managed", exoConfig.ManagedByRubrik); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cluster_security_group_id", exoConfig.ClusterSecurityGroupID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("node_security_group_id", exoConfig.NodeSecurityGroupID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// awsDeleteExocompute run the Delete operation for the AWS exocompute
// resource. This removes the exocompute config from Polaris.
func awsDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] awsDeleteExocompute")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = aws.Wrap(client).RemoveExocomputeConfig(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
