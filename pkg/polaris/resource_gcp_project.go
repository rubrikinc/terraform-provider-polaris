package polaris

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/trinity-team/rubrik-polaris-sdk-for-go/pkg/polaris"
)

// resourceGcpProject defines the schema for the GCP project resource. There is
// no update function since all parameters are forced to new.
func resourceGcpProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: gcpCreateProject,
		ReadContext:   gcpReadProject,
		DeleteContext: gcpDeleteProject,

		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateDiagFunc: func(m interface{}, p cty.Path) diag.Diagnostics {
					if _, err := os.Stat(m.(string)); err != nil {
						details := "unknown error"

						var pathErr *fs.PathError
						if errors.As(err, &pathErr) {
							details = pathErr.Err.Error()
						}

						return diag.Errorf("failed to access the credentials file: %s", details)
					}

					return nil
				},
				Description: "Path to a service account key file in JSON format.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "GCP project.",
			},
		},
	}
}

// gcpCreateProject run the Create operation for the GCP schema resource. This
// adds the GCP project to the Polaris platform.
func gcpCreateProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpCreateProject")

	client := m.(*polaris.Client)
	keyFile := d.Get("credentials").(string)
	projectID := d.Get("project").(string)

	gcpConfig := polaris.FromGcpKeyFile(keyFile)
	if projectID != "" {
		gcpConfig = polaris.FromGcpKeyFileWithProjectID(keyFile, projectID)
	}

	// Check if the project already exist in Polaris.
	project, err := client.GcpProject(ctx, gcpConfig)
	switch {
	case errors.Is(err, polaris.ErrNotFound):
	case err == nil:
		return diag.Errorf("project %q already added to polaris", project.ProjectID)
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
	d.SetId(toResourceID(project.ID, project.ProjectID))
	d.Set("project", project.ProjectID)

	return nil
}

// gcpReadProject run the Read operation for the GCP schema resource. This
// reads the state of the GCP project in Polaris.
func gcpReadProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpReadProject")

	client := m.(*polaris.Client)

	// Extract the GCP project id from the resource id.
	_, projectID, err := fromResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Lookup the GCP project in Polaris.
	project, err := client.GcpProject(ctx, polaris.WithGcpProjectID(projectID))
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("project", project.ProjectID)

	return nil
}

// gcpDeleteProject run the Delete operation for the AWS schema resource. This
// removes the AWS account from Polaris.
func gcpDeleteProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Print("[TRACE] gcpDeleteProject")

	client := m.(*polaris.Client)

	// Get the old resource arguments.
	oldProject, _ := d.GetChange("project")
	projectID := oldProject.(string)

	// Remove the project from Polaris.
	if err := client.GcpProjectRemove(ctx, polaris.WithGcpProjectID(projectID), false); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return nil
}
