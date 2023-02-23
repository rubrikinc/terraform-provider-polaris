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
	"log"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// resourceAzureExocompute defines the schema for the Azure exocompute resource.
func resourceAzureExocompute() *schema.Resource {
	return &schema.Resource{
		CreateContext: azureCreateExocompute,
		ReadContext:   azureReadExocompute,
		DeleteContext: azureDeleteExocompute,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Polaris subscription id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"polaris_managed": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Deprecated:  "This argument has no effect on the exocompute configuration. Follow the upgrade guide for v0.6.0, when released, to remove.",
				Description: "If true the security groups are managed by Polaris.",
			},
			"region": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Azure region to run the exocompute instance in.",
				ValidateDiagFunc: validateAzureRegion,
			},
			"subnet": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Azure subnet id.",
			},
		},
	}
}

// azureCreateExocompute run the Create operation for the Azure exocompute
// resource. This enables the exocompute feature and adds an exocompute config
// to the Polaris cloud account.
func azureCreateExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureCreateExocompute")

	client := m.(*polaris.Client)

	accountID, err := uuid.Parse(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	account, err := azure.NewAPI(client.GQL).Subscription(ctx, azure.CloudAccountID(accountID), core.FeatureExocompute)
	if errors.Is(err, graphql.ErrNotFound) {
		return diag.Errorf("exocompute not enabled on account")
	}
	if err != nil {
		return diag.FromErr(err)
	}

	region := d.Get("region").(string)
	if !account.Features[0].HasRegion(region) {
		return diag.Errorf("region %q not available with exocompute feature", region)
	}

	var config azure.ExoConfigFunc
	if d.Get("polaris_managed").(bool) {
		config = azure.Managed(region, d.Get("subnet").(string))
	} else {
		//lint:ignore SA1019 unmanaged exocompute configuration will not be supported in the next release
		config = azure.Unmanaged(region, d.Get("subnet").(string))
	}

	id, err := azure.NewAPI(client.GQL).AddExocomputeConfig(ctx, azure.CloudAccountID(accountID), config)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id.String())

	awsReadExocompute(ctx, d, m)
	return nil
}

// azureReadExocompute run the Read operation for the Azure exocompute
// resource. This reads the state of the exocompute config in Polaris.
func azureReadExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureReadExocompute")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	exoConfig, err := azure.NewAPI(client.GQL).ExocomputeConfig(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("region", exoConfig.Region); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("subnet", exoConfig.SubnetID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("polaris_managed", exoConfig.ManagedByRubrik); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// azureDeleteExocompute run the Delete operation for the Azure exocompute
// resource. This removes the exocompute config from Polaris.
func azureDeleteExocompute(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azureDeleteExocompute")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = azure.NewAPI(client.GQL).RemoveExocomputeConfig(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
