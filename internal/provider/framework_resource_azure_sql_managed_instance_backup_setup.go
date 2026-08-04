// Copyright 2026 Rubrik, Inc.
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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/azure"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core/secret"
)

// defaultSQLManagedInstanceBackupSetupTimeout is how long to wait for the
// backup setup job to finish before giving up.
const defaultSQLManagedInstanceBackupSetupTimeout = 30 * time.Minute

const resourceAzureSQLManagedInstanceBackupSetupDescription = `
The ´polaris_azure_sql_managed_instance_backup_setup´ resource configures RSC to
back up an Azure SQL Managed Instance server.

RSC connects to the managed instance using the SQL Server credentials in the
´sql_credentials´ block and creates the user it uses to perform backups. The
credentials are only used for this setup and are not stored by RSC, which is why
they are write-only arguments: they are sent to RSC but never written to
Terraform state, so they can be sourced from a secret store such as Vault
without leaking into state.

Use the ´polaris_object´ data source with an object type of
´AzureSqlManagedInstanceServer´ to look up the ´server_id´ by name.

~> **Note:** Because the ´sql_credentials´ arguments are write-only, changing
them produces no difference in the plan. Change ´sql_credential_version´ to make
Terraform send the credentials again.

~> **Note:** The credentials are validated by the managed instance only once the
setup job runs, so invalid credentials surface as a failed job rather than as an
immediate error.
`

var (
	_ resource.Resource              = &azureSQLManagedInstanceBackupSetupResource{}
	_ resource.ResourceWithConfigure = &azureSQLManagedInstanceBackupSetupResource{}
)

type azureSQLManagedInstanceBackupSetupResource struct {
	client *client
}

type azureSQLManagedInstanceBackupSetupResourceModel struct {
	ID                   types.String         `tfsdk:"id"`
	ServerID             types.String         `tfsdk:"server_id"`
	SQLCredentials       *sqlCredentialsModel `tfsdk:"sql_credentials"`
	SQLCredentialVersion types.String         `tfsdk:"sql_credential_version"`
	Timeouts             timeouts.Value       `tfsdk:"timeouts"`
}

type sqlCredentialsModel struct {
	SQLUsername types.String `tfsdk:"sql_username"`
	SQLPassword types.String `tfsdk:"sql_password"`
}

func newAzureSQLManagedInstanceBackupSetupResource() resource.Resource {
	return &azureSQLManagedInstanceBackupSetupResource{}
}

func (r *azureSQLManagedInstanceBackupSetupResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Metadata")

	res.TypeName = req.ProviderTypeName + "_" + keyAzureSQLManagedInstanceBackupSetup
}

func (r *azureSQLManagedInstanceBackupSetupResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Schema")

	res.Schema = schema.Schema{
		Description: description(resourceAzureSQLManagedInstanceBackupSetupDescription),
		Attributes: map[string]schema.Attribute{
			keyID: schema.StringAttribute{
				Computed:    true,
				Description: "RSC object ID of the SQL Managed Instance server (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			keyServerID: schema.StringAttribute{
				Required: true,
				Description: "RSC object ID of the SQL Managed Instance server (UUID). Changing this forces a " +
					"new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					isUUID(),
				},
			},
			keySQLCredentialVersion: schema.StringAttribute{
				Required: true,
				Description: "Arbitrary value identifying the version of the credentials. Change it to make " +
					"Terraform send the `sql_credentials` block again, e.g. after rotating the password. " +
					"Write-only arguments produce no difference in the plan on their own.",
				Validators: []validator.String{
					isNotWhiteSpace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			keySQLCredentials: schema.SingleNestedBlock{
				Description: "Credentials of a SQL Server user with permission to create the user RSC uses to " +
					"perform backups. Write-only, change `sql_credential_version` to send them again.",
				Validators: []validator.Object{
					objectvalidator.IsRequired(),
				},
				Attributes: map[string]schema.Attribute{
					keySQLUsername: schema.StringAttribute{
						Required:    true,
						WriteOnly:   true,
						Description: "SQL Server login.",
						Validators: []validator.String{
							isNotWhiteSpace(),
						},
					},
					keySQLPassword: schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "Password for `sql_username`.",
						Validators: []validator.String{
							isNotWhiteSpace(),
						},
					},
				},
			},
			keyTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *azureSQLManagedInstanceBackupSetupResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Configure")

	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client)
}

func (r *azureSQLManagedInstanceBackupSetupResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Create")

	var plan azureSQLManagedInstanceBackupSetupResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultSQLManagedInstanceBackupSetupTimeout)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(r.setupBackup(ctx, req.Config, plan, timeout)...)
	if res.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.ServerID
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

// Read is intentionally a no-op. RSC does not expose the backup credentials of
// a SQL Managed Instance server, so there is nothing to read back and no way to
// detect drift. Removing the resource from state here would make every plan
// recreate it.
func (r *azureSQLManagedInstanceBackupSetupResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Read")

	var state azureSQLManagedInstanceBackupSetupResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}
	res.Diagnostics.Append(res.State.Set(ctx, &state)...)
}

// Update re-runs the backup setup. Since server_id requires replacement, the
// only change which can reach this point is a new credentials_version, with or
// without new credentials.
func (r *azureSQLManagedInstanceBackupSetupResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Update")

	var plan azureSQLManagedInstanceBackupSetupResourceModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if res.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, defaultSQLManagedInstanceBackupSetupTimeout)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(r.setupBackup(ctx, req.Config, plan, timeout)...)
	if res.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.ServerID
	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *azureSQLManagedInstanceBackupSetupResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "azureSQLManagedInstanceBackupSetupResource.Delete")

	var state azureSQLManagedInstanceBackupSetupResourceModel
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	serverID, err := uuid.Parse(state.ServerID.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Invalid server ID", err.Error())
		return
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		res.Diagnostics.AddError("RSC client error", err.Error())
		return
	}

	if err := azure.Wrap(polarisClient).ClearSQLManagedInstanceBackupCredentials(ctx, []uuid.UUID{serverID}); err != nil {
		res.Diagnostics.AddError("Failed to clear SQL Managed Instance backup credentials", err.Error())
		return
	}
}

// setupBackup reads the write-only credentials from the configuration and runs
// the backup setup, blocking until the setup job finishes. Write-only arguments
// are null in the plan and never stored in state, so they must be read from the
// configuration.
func (r *azureSQLManagedInstanceBackupSetupResource) setupBackup(ctx context.Context, cfg tfsdk.Config, plan azureSQLManagedInstanceBackupSetupResourceModel, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics

	var config azureSQLManagedInstanceBackupSetupResourceModel
	diags.Append(cfg.Get(ctx, &config)...)
	if diags.HasError() {
		return diags
	}

	if config.SQLCredentials == nil {
		diags.AddError("Missing SQL credentials", "the sql_credentials block is required")
		return diags
	}

	serverID, err := uuid.Parse(plan.ServerID.ValueString())
	if err != nil {
		diags.AddError("Invalid server ID", err.Error())
		return diags
	}

	polarisClient, err := r.client.polaris()
	if err != nil {
		diags.AddError("RSC client error", err.Error())
		return diags
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = azure.Wrap(polarisClient).SetupSQLManagedInstanceBackup(ctx, []uuid.UUID{serverID},
		gqlazure.LoginCredentials{
			Login:    config.SQLCredentials.SQLUsername.ValueString(),
			Password: secret.String(config.SQLCredentials.SQLPassword.ValueString()),
		}, 0)
	if err != nil {
		diags.AddError("Failed to set up SQL Managed Instance backup", err.Error())
		return diags
	}

	return diags
}
