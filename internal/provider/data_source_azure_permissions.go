package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// dataSourceAzurePermissions defines the schema for the Azure permissions data
// source.
func dataSourceAzurePermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: azurePermissionsRead,

		Schema: map[string]*schema.Schema{
			"actions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Allowed actions.",
			},
			"data_actions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Allowed data actions.",
			},
			"features": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateFeature,
				},
				Required:    true,
				Description: "Enabled features.",
			},
			"hash": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the permissions, can be used to detect changes to the permissions.",
			},
			"not_actions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Disallowed actions.",
			},
			"not_data_actions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Disallowed data actions.",
			},
		},
	}
}

// azurePermissionsRead run the Read operation for the Azure permissions data
// source. Reads the permissions required for the specified Polaris features.
func azurePermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] azurePermissionsRead")

	client := m.(*polaris.Client)

	// Read permissions required for the specified features.
	var features []core.Feature
	for _, f := range d.Get("features").(*schema.Set).List() {
		feature, err := core.ParseFeature(f.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		features = append(features, feature)
	}

	perms, err := client.Azure().Permissions(ctx, features)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Strings(perms.Actions)
	sort.Strings(perms.DataActions)
	sort.Strings(perms.NotActions)
	sort.Strings(perms.NotDataActions)

	// Format permissions according to the data source schema.
	hash := sha256.New()

	var actions []interface{}
	for _, perm := range perms.Actions {
		actions = append(actions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set("actions", actions); err != nil {
		return diag.FromErr(err)
	}

	var dataActions []interface{}
	for _, perm := range perms.DataActions {
		dataActions = append(dataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set("data_actions", dataActions); err != nil {
		return diag.FromErr(err)
	}

	var notActions []interface{}
	for _, perm := range perms.NotActions {
		notActions = append(notActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set("not_actions", notActions); err != nil {
		return diag.FromErr(err)
	}

	var notDataActions []interface{}
	for _, perm := range perms.NotDataActions {
		notDataActions = append(notDataActions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set("not_data_actions", notDataActions); err != nil {
		return diag.FromErr(err)
	}

	d.Set("hash", fmt.Sprintf("%x", hash.Sum(nil)))

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
