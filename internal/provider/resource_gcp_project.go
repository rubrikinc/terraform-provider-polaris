package provider

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
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
				Description:      "Path to Google Cloud Platform service account file.",
				ValidateDiagFunc: credentialsFileExists,
			},
			"delete_snapshots_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "What should happen to snapshots when the project is removed from Polaris.",
			},
			"organization_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				Description:      "Google Cloud Platform organization name.",
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
				Description:      "Google Cloud Platform project name.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"project_number": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ConflictsWith:    []string{"credentials"},
				RequiredWith:     []string{"organization_name", "project", "project_number"},
				Description:      "Google Cloud Platform project number.",
				ValidateDiagFunc: stringIsInteger,
			},
		},
	}
}

// gcpCreateProject run the Create operation for the GCP project resource. This
// adds the GCP project to the Polaris platform.
func gcpCreateProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpCreateProject")

	client := m.(*polaris.Client)

	// Resource parameters.
	credentials := d.Get("credentials").(string)
	organizationName := d.Get("organization_name").(string)
	projectID := d.Get("project").(string)
	projectName := d.Get("project_name").(string)

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
	gcpConfig := polaris.FromGcpProject(projectID, projectName, projectNumber, organizationName)
	switch {
	case credentials != "" && projectID == "":
		gcpConfig = polaris.FromGcpKeyFile(credentials)
	case credentials != "" && projectID != "":
		gcpConfig = polaris.FromGcpKeyFileWithProjectID(credentials, projectID)
	}

	// Check if the project already exist in Polaris.
	project, err := client.GcpProject(ctx, gcpConfig)
	switch {
	case errors.Is(err, polaris.ErrNotFound):
	case err == nil:
		return diag.Errorf("project %q has already been added to polaris", project.ProjectID)
	case err != nil:
		return diag.FromErr(err)
	}

	// Add project to Polaris.
	if err := client.GcpProjectAdd(ctx, gcpConfig); err != nil {
		return diag.FromErr(err)
	}

	// Lookup the id and GCP project id of the newly added project. Note that
	// the resource id is created from both.
	project, err = client.GcpProject(ctx, gcpConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toResourceID(project.ID, strings.ToLower(project.ProjectID)))

	gcpReadProject(ctx, d, m)
	return nil
}

// gcpReadProject run the Read operation for the GCP project resource. This
// reads the state of the GCP project in Polaris.
func gcpReadProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpReadProject")

	client := m.(*polaris.Client)

	// Extract the GCP project id from the resource id.
	_, projectID, err := fromResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the GCP project in Polaris and update the local state.
	project, err := client.GcpProject(ctx, polaris.WithGcpProjectID(projectID))
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("organization_name", project.OrganizationName)
	d.Set("project_name", project.ProjectName)
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

	// Get the old resource arguments.
	oldProject, _ := d.GetChange("project")
	projectID := oldProject.(string)

	oldSnapshots, _ := d.GetChange("delete_snapshots_on_destroy")
	deleteSnapshots := oldSnapshots.(bool)

	// Remove the project from Polaris.
	if err := client.GcpProjectRemove(ctx, polaris.WithGcpProjectID(projectID), deleteSnapshots); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
