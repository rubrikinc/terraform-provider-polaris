---
page_title: "Upgrade Guide: v1.9.0"
---

# Upgrade Guide v1.9.0

The v1.9.0 release adds a feature-gated V1/V2 model for Azure SQL Database and Managed Instance SLAs in the
`polaris_sla_domain` resource. See the [changelog](changelog.md) for the full list of changes.

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

Starting with v1.7.0, each release is also published as the renamed `rubrikinc/rubrik` provider. The
`rubrikinc/polaris` provider will continue to be released and supported for some time, so there is no need to switch
right now. The `rubrikinc/polaris` provider will eventually be retired, however, and you will need to switch to the
`rubrikinc/rubrik` provider before then. The migration paths will improve over time as more resources gain support for
Terraform's `moved {}` block, making the switch progressively simpler. See the
[latest upgrade guide for the rubrikinc/rubrik provider](https://registry.terraform.io/providers/rubrikinc/rubrik/latest/docs/guides)
for the currently available migration paths.

~> **Note:** If you are upgrading across multiple minor versions, review the upgrade guide for each intermediate
version as well. Each guide documents breaking changes and migration steps specific to that release.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.9.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.9.0"
    }
  }
}
```
Next, upgrade the provider to the new version by running:
```shell
% terraform init -upgrade
```
After the provider has been updated, validate the correctness of the Terraform configuration files by running:
```shell
% terraform plan
```
If you get an error or an unwanted diff, please see the _Significant Changes_ section below for additional instructions.
Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.9.0 version.

## Significant Changes

### Azure SQL Database and Managed Instance SLAs (feature-gated)

When the `CNP_AZURE_SQL_SLA_REVAMP` feature is enabled for your account, Azure SQL Database and Managed Instance SLAs
in the `polaris_sla_domain` resource follow a new V1/V2 model:

* A **V1** (Azure-managed, long-term retention) SLA carries a new `ltr_config` block (weekly, monthly, and yearly
  retention) and takes no Rubrik snapshot schedule or backup location.
* A **V2** (Rubrik-managed) SLA omits `ltr_config` and specifies a Rubrik snapshot schedule together with a
  `backup_location` block.

~> **Note:** This behavior is controlled by the `CNP_AZURE_SQL_SLA_REVAMP` account-level feature flag, not by the
provider version — enabling it affects any provider version managing Azure SQL SLAs for that account. If the feature
is not enabled for your account, Azure SQL SLAs are unaffected and **no configuration changes are required**.

With the feature enabled, the way an Azure SQL SLA specifies its backup location changes:

* **Before:** an Azure SQL Database SLA required exactly one top-level `archival` block with instant archival enabled,
  and an Azure SQL Managed Instance SLA could not specify an archival location.
* **After:** a V2 Azure SQL SLA specifies its location with a top-level `backup_location` block (the same block used by
  AWS S3 multiple backup locations) and must not use the `archival` block.

If the feature is enabled and you have an existing Azure SQL Database SLA that uses the `archival` block, replace it
with a `backup_location` block:
```terraform
# Before
resource "polaris_sla_domain" "azure_sql" {
  name         = "azure-sql"
  object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

  hourly_schedule {
    frequency      = 1
    retention      = 1
    retention_unit = "DAYS"
  }

  azure_sql_database_config {
    log_retention = 7
  }

  archival {
    archival_location_id = data.polaris_azure_archival_location.example.id
    threshold            = 0
  }
}

# After
resource "polaris_sla_domain" "azure_sql" {
  name         = "azure-sql"
  object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

  hourly_schedule {
    frequency      = 1
    retention      = 1
    retention_unit = "DAYS"
  }

  azure_sql_database_config {
    log_retention = 7
  }

  backup_location {
    archival_group_id = data.polaris_azure_archival_location.example.id
  }
}
```

To manage Azure native long-term retention, configure a V1 SLA with `ltr_config` and no schedule or backup location:
```terraform
resource "polaris_sla_domain" "azure_sql_v1" {
  name         = "azure-sql-v1"
  object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

  azure_sql_database_config {
    log_retention = 7
    ltr_config {
      weekly_retention {
        retention      = 4
        retention_unit = "WEEKS"
      }
      monthly_retention {
        retention      = 12
        retention_unit = "MONTHS"
      }
      yearly_retention {
        retention      = 7
        retention_unit = "YEARS"
        week_of_year   = 1
      }
    }
  }
}
```

~> **Note:** An existing SLA cannot be switched between V1 (Azure-managed) and V2 (Rubrik-managed) in place — the
provider rejects a change that adds or removes `ltr_config` on an existing `polaris_sla_domain`. To change the backup
type, create a new SLA Domain and reassign the affected databases to it. This matches the RSC UI, which disables the
backup-service selector when editing an existing SLA.

The release also adds a computed `backup_type` attribute (`NATIVE` for V1, `RUBRIK` for V2) and allows combining the
Azure SQL Database and Managed Instance object types in a single SLA.
