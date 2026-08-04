# Configures RSC to back up an Azure SQL Managed Instance server.
#
# RSC connects to the managed instance using the credentials in the
# sql_credentials block and creates the user it uses to perform backups. The
# credentials are write-only: they are sent to RSC but never written to
# Terraform state, so they can come straight from a secret store.

# Look up the managed instance server by name to get its RSC object ID.
data "polaris_object" "sql_mi" {
  name        = "my-sql-managed-instance"
  object_type = "AzureSqlManagedInstanceServer"

  # Optional, set to the RSC cloud account ID of the parent subscription when
  # the same server name exists in more than one subscription.
  # subscription_id = "00000000-0000-0000-0000-000000000000"
}

variable "sql_username" {
  type = string
}

variable "sql_password" {
  type      = string
  sensitive = true
}

resource "polaris_azure_sql_managed_instance_backup_setup" "example" {
  server_id = data.polaris_object.sql_mi.id

  # Write-only, so these never reach Terraform state and can be sourced from a
  # secret store such as Vault.
  sql_credentials {
    sql_username = var.sql_username
    sql_password = var.sql_password
  }

  # Because sql_credentials is write-only, changing the username or password
  # produces no difference in the plan. Change sql_credential_version to send
  # them again, for example after rotating the password.
  sql_credential_version = "1"
}
