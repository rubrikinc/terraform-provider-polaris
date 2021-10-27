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

// dataSourceGcpPermissions defines the schema for the GCP permissions data
// source.
func dataSourceGcpPermissions() *schema.Resource {
	return &schema.Resource{
		ReadContext: gcpPermissionsRead,

		Schema: map[string]*schema.Schema{
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
			"permissions": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Permissions required for the features enabled.",
			},
		},
	}
}

// gcpPermissionsRead run the Read operation for the GCP permissions data
// source. Reads the permissions required for the specified Polaris features.
func gcpPermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpPermissionsRead")

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

	perms, err := client.GCP().Permissions(ctx, features)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Strings(perms)

	// Format permissions according to the data source schema.
	var permissions []interface{}
	hash := sha256.New()
	for _, perm := range perms {
		permissions = append(permissions, perm)
		hash.Write([]byte(perm))
	}
	if err := d.Set("permissions", &permissions); err != nil {
		return diag.FromErr(err)
	}

	d.Set("hash", fmt.Sprintf("%x", hash.Sum(nil)))

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
