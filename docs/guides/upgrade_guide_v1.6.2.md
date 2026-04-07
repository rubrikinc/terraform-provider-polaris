---
page_title: "Upgrade Guide: v1.6.2"
---

# Upgrade Guide v1.6.2

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. If you have existing `polaris_azure_subscription` resources with the `sql_db_protection` block and
user-assigned managed identity fields, upgrading to v1.6.2 may show a diff on the first `terraform plan` as the
provider reads back the managed identity state from the API. This is expected and can be resolved by running
`terraform apply -refresh-only`.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.6.2 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.6.0"
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
If you see a diff on `user_assigned_managed_identity_name` or `user_assigned_managed_identity_principal_id` fields,
this is expected as the provider now reads these values from the API. Proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.6.2 version.

## Bug Fixes

### polaris_azure_subscription: managed identity upgrade fix

The `upgradeFeatureToUseManagedIdentity` function was not including permission groups from the Terraform configuration
when calling the Go SDK. This caused the SDK to select the legacy GraphQL query variant (using the deprecated `feature`
field), which silently dropped the `FeatureSpecificInfo` payload containing the user-assigned managed identity (UMI)
details. As a result, subscriptions upgraded to `BACKUP_V2` were missing the required UMI mapping.

This fix extracts permission groups from the configuration block before calling the SDK, matching the pattern already
used in `upgradeSQLDBFeatureToUseResourceGroup`. The SDK now selects the correct query variant (`featureToUpgrade`),
which carries both permission groups and the managed identity input through to backend validation.

### polaris_azure_subscription: managed identity state refresh

The `user_assigned_managed_identity_name` and `user_assigned_managed_identity_principal_id` fields in the
`sql_db_protection` block are now read back from the API during `terraform plan` and `terraform apply`. Previously
these fields were write-only and not refreshed from remote state, which could cause unnecessary diffs or mask
configuration drift.
