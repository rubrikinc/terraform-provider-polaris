package provider

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/gcp"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// stringIsInteger assumes m is a string holding an integer and returns nil if
// the string can be converted to an integer, otherwise an diagnostic message
// is returned.
func stringIsInteger(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := strconv.ParseInt(m.(string), 10, 64); err != nil {
		return diag.Errorf("expected an integer: %s", err)
	}

	return nil
}

// resourceGcpProject defines the schema for the GCP project resource. Note
// that the update function only changes the local state.
func resourceGcpProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateProject,
		ReadContext:   gcpReadProject,
		UpdateContext: gcpUpdateProject,
		DeleteContext: gcpDeleteProject,

		Schema: map[string]*schema.Schema{
			"cloud_native_protection": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the Cloud Native Protection feature.",
						},
					},
				},
				MaxItems:    1,
				Required:    true,
				Description: "Enable the Cloud Native Protection feature for the GCP project.",
			},
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"credentials", "project_number"},
				Description:      "Path to GCP service account key file.",
				ValidateDiagFunc: fileExists,
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"organization_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Organization name.",
			},
			"permissions_hash": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Signals that the permissions has been updated.",
				ValidateDiagFunc: validateHash,
			},
			"project": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				Description:      "Project id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_name": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Project name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_number": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				RequiredWith:     []string{"organization_name", "project", "project_name"},
				Description:      "Project number.",
				ValidateDiagFunc: stringIsInteger,
			},
		},

		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceGcpProjectV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpProjectStateUpgradeV0,
			Version: 0,
		}, {
			Type:    resourceGcpProjectV1().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpProjectStateUpgradeV1,
			Version: 1,
		}},
	}
}

// gcpCreateProject run the Create operation for the GCP project resource. This
// adds the GCP project to the Polaris platform.
func gcpCreateProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpCreateProject")

	client := m.(*polaris.Client)
	credentials := d.Get("credentials").(string)
	projectID := d.Get("project").(string)

	var opts []gcp.OptionFunc
	if name, ok := d.GetOk("project_name"); ok {
		opts = append(opts, gcp.Name(name.(string)))
	}
	if orgName, ok := d.GetOk("organization_name"); ok {
		opts = append(opts, gcp.Organization(orgName.(string)))
	}

	// Terraform schema integers are restricted to int and hence cannot handle
	// a GCP project number when running on a 32-bit platform.
	var projectNumber int64
	if pn, ok := d.GetOk("project_number"); ok {
		var err error
		projectNumber, err = strconv.ParseInt(pn.(string), 10, 64)
		if err != nil {
			return diag.Errorf("project_number should be an integer: %s", err)
		}
	}

	// Determine how the project details should be passed on to Polaris.
	var project gcp.ProjectFunc
	switch {
	case credentials != "" && projectID == "":
		project = gcp.KeyFile(credentials)
	case credentials != "" && projectID != "":
		project = gcp.KeyFileWithProject(credentials, projectID)
	default:
		project = gcp.Project(projectID, projectNumber)
	}

	account, err := client.GCP().Project(ctx, gcp.ID(project), core.FeatureAll)
	if err == nil {
		return diag.Errorf("project %q already added to polaris", account.NativeID)
	}
	if !errors.Is(err, graphql.ErrNotFound) {
		return diag.FromErr(err)
	}

	// At this time GCP only supports the CNP feature.
	id, err := client.GCP().AddProject(ctx, project, core.FeatureCloudNativeProtection, opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.String())

	gcpReadProject(ctx, d, m)
	return nil
}

// gcpReadProject run the Read operation for the GCP project resource. This
// reads the state of the GCP project in Polaris.
func gcpReadProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpReadProject")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the GCP project in Polaris and update the local state.
	account, err := client.GCP().Project(ctx, gcp.CloudAccountID(id), core.FeatureAll)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("organization_name", account.OrganizationName)
	d.Set("project", account.NativeID)
	d.Set("project_name", account.Name)
	d.Set("project_number", strconv.FormatInt(account.ProjectNumber, 10))

	if feature, ok := account.Feature(core.FeatureCloudNativeProtection); ok {
		status := core.FormatStatus(feature.Status)
		err := d.Set("cloud_native_protection", []interface{}{
			map[string]interface{}{
				"status": &status,
			},
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// gcpUpdateProject run the Update operation for the GCP project resource. This
// only updates the local delete_snapshots_on_destroy parameter.
func gcpUpdateProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpUpdateProject")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("permissions_hash") {
		err = client.GCP().PermissionsUpdated(ctx, gcp.CloudAccountID(id), nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gcpReadProject(ctx, d, m)
	return nil
}

// gcpDeleteProject run the Delete operation for the GCP project resource. This
// removes the GCP project from Polaris.
func gcpDeleteProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpDeleteProject")

	client := m.(*polaris.Client)

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the old resource arguments.
	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	// Remove the project from Polaris.
	err = client.GCP().RemoveProject(ctx, gcp.CloudAccountID(id), core.FeatureCloudNativeProtection, deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
