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
			"credentials": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				AtLeastOneOf:     []string{"credentials", "project"},
				Description:      "Path to GCP service account key file.",
				ValidateDiagFunc: credentialsFileExists,
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should snapshots be deleted when the resource is destroyed.",
			},
			"organization_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				Description:      "GCP organization name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				Description:      "GCP project id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				Description:      "GCP project name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_number": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				Description:      "GCP project number.",
				ValidateDiagFunc: stringIsInteger,
			},
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{{
			Type:    resourceGcpProjectV0().CoreConfigSchema().ImpliedType(),
			Upgrade: resourceGcpProjectStateUpgradeV0,
			Version: 0,
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
		project = gcp.KeyFileAndProject(credentials, projectID)
	case credentials == "" && projectID != "":
		project = gcp.Project(projectID, projectNumber)
	}

	// Check if the project already exist in Polaris.
	account, err := client.GCP().Project(ctx, gcp.ID(project), core.CloudNativeProtection)
	switch {
	case errors.Is(err, graphql.ErrNotFound):
	case err == nil:
		return diag.Errorf("project %q has already been added to polaris", account.NativeID)
	case err != nil:
		return diag.FromErr(err)
	}

	// Add project to Polaris.
	id, err := client.GCP().AddProject(ctx, project, opts...)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id.String())

	// Populate the local Terraform state.
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
	project, err := client.GCP().Project(ctx, gcp.CloudAccountID(id), core.CloudNativeProtection)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("organization_name", project.OrganizationName)
	d.Set("project_name", project.Name)
	d.Set("project_number", strconv.FormatInt(project.ProjectNumber, 10))

	return nil
}

// gcpUpdateProject run the Update operation for the GCP project resource. This
// only updates the local delete_snapshots_on_destroy parameter.
func gcpUpdateProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpDeleteProject")

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
	err = client.GCP().RemoveProject(ctx, gcp.CloudAccountID(id), deleteSnapshots)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
