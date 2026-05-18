---
page_title: "Upgrade Guide: v1.8.0"
---

# Upgrade Guide v1.8.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

~> **Note:** If you are upgrading across multiple minor versions (e.g. v1.6.x to v1.8.0), review the upgrade guide for
each intermediate version as well. Each guide documents breaking changes and migration steps specific to that release.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.8.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.8.0"
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
If you get an error or an unwanted diff, please see the _Breaking Changes_ and _New Features_ sections below for
additional instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.8.0 version.

## Breaking Changes

### polaris_aws_cnp_account_attachments: feature schema reshape

The `features` attribute on the `polaris_aws_cnp_account_attachments` resource has been replaced with a `feature`
block that carries both the feature name and its permission groups. This lets the resource pass permission groups
through to the RSC `registerAwsFeatureArtifacts` mutation, which is required for the new `RECOVERY` permission group on
`RDS_PROTECTION` and `CLOUD_NATIVE_DYNAMODB_PROTECTION`.

State is migrated automatically by a schema upgrader; the upgrader backfills `permission_groups = ["BASIC"]` for every
existing feature, which matches the prior behavior. **You must update your configuration before running
`terraform apply`** — otherwise Terraform will report the `features` attribute as removed.

Before:
```terraform
resource "polaris_aws_cnp_account_attachments" "attachments" {
  account_id = polaris_aws_cnp_account.account.id
  features   = polaris_aws_cnp_account.account.feature.*.name

  # ... instance_profile, role blocks ...
}
```

After:
```terraform
resource "polaris_aws_cnp_account_attachments" "attachments" {
  account_id = polaris_aws_cnp_account.account.id

  dynamic "feature" {
    for_each = polaris_aws_cnp_account.account.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }

  # ... instance_profile, role blocks ...
}
```

The `dynamic` block mirrors the feature/permission_group set already defined on the linked `polaris_aws_cnp_account`
resource, so adding `RECOVERY` (or any new permission group) on the account propagates to the attachments resource
without further changes.

## New Features

### AWS RDS and DynamoDB privilege elevation

RSC has introduced a `RECOVERY` permission group for the `RDS_PROTECTION` and `CLOUD_NATIVE_DYNAMODB_PROTECTION`
features. The `RECOVERY` group carries the elevated AWS privileges required to perform recovery operations on RDS
instances and DynamoDB tables. Customers can now opt in to the elevated permissions only when recovery is needed,
keeping the day-to-day footprint at `BASIC`.

The new permission group is accepted on:
- `polaris_aws_account` — `rds_protection.permission_groups` and `cloud_native_dynamodb_protection.permission_groups`
- `polaris_aws_cnp_account` — `feature.permission_groups` when `feature.name` is `RDS_PROTECTION` or
  `CLOUD_NATIVE_DYNAMODB_PROTECTION`
- `polaris_aws_cnp_account_attachments` — `feature.permission_groups` for the same features
- `polaris_aws_cnp_permissions` data source — request the recovery policy by setting
  `feature.permission_groups = ["RECOVERY"]`

`RECOVERY` requires the `REL_ENABLE_AWS_PAAS_DB_PRIVILEGE_ELEVATION` feature flag to be enabled on the RSC account.

#### Example

```terraform
resource "polaris_aws_cnp_account" "account" {
  # ... cloud, native_id, name, regions ...

  feature {
    name              = "RDS_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
  }

  feature {
    name              = "CLOUD_NATIVE_DYNAMODB_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
  }
}
```

### Expected Terraform drift after RSC migrates existing accounts

When the RSC backend migrates existing AWS accounts to the split permission groups, accounts that historically had
`CLOUD_NATIVE_PROTECTION` `BASIC` will gain `RESTORE`, `EXPORT_POWER_ON`, `EXPORT_POWER_OFF` and `DOWNLOAD_FILE`. Other
features (`RDS_PROTECTION`, `CLOUD_NATIVE_DYNAMODB_PROTECTION`) are not modified by the RSC migration — they remain at
`BASIC` until the customer opts in to `RECOVERY`.

After the RSC migration, a `terraform plan` against a configuration that lists only `BASIC` will show the new groups as
being removed from the account. This is expected and not harmful — applying the plan removes the elevated permissions,
returning the account to a `BASIC`-only footprint. To retain the elevated permissions, add them explicitly to the
config:

```terraform
feature {
  name              = "CLOUD_NATIVE_PROTECTION"
  permission_groups = ["BASIC", "RESTORE", "EXPORT_POWER_ON", "EXPORT_POWER_OFF", "DOWNLOAD_FILE"]
}
```
