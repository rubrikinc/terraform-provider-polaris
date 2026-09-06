---
page_title: "Upgrade Guide: v1.7.1"
subcategory: "Upgrade Guides"
---

# Upgrade Guide v1.7.1

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

The v1.7.1 release is also published as the renamed `rubrikinc/rubrik` provider. The `rubrikinc/polaris` provider will
continue to be released and supported for some time, so there is no need to switch right now. The
`rubrikinc/polaris` provider will eventually be retired, however, and you will need to switch to the `rubrikinc/rubrik`
provider before then. The migration paths will improve over time as more resources gain support for Terraform's
`moved {}` block, making the switch progressively simpler. See the
[v1.7.1 upgrade guide for the rubrikinc/rubrik provider](https://registry.terraform.io/providers/rubrikinc/rubrik/latest/docs/guides/upgrade_guide_v1.7.1)
for the currently available migration paths.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.7.1 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.7.1"
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
If you get an error or an unwanted diff, please see the _New Features_ and _Deprecations_ sections below for additional
instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.7.1 version.

## New Features

### RECOVERY permission group for RDS and DynamoDB

The `RDS_PROTECTION` and `CLOUD_NATIVE_DYNAMODB_PROTECTION` features now support a separate `RECOVERY` permission group
alongside `BASIC`. `BASIC` covers backup; `RECOVERY` grants the elevated AWS permissions required to perform recovery
operations. This split lets you keep the day-to-day footprint minimal and grant elevated privileges only when needed.

The new group is available on the `polaris_aws_account`, `polaris_aws_cnp_account` and `polaris_aws_cnp_account_attachments`
resources, and on the `polaris_aws_cnp_permissions` data source. To opt in, add `RECOVERY` to the `permission_groups`
set on the matching feature block in `polaris_aws_cnp_account` (or, for `polaris_aws_account`, enable the relevant
sub-fields in the `cloud_native_dynamodb_protection` or `rds_protection` block):

```terraform
resource "polaris_aws_cnp_account" "account" {
  # ...
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

If your RSC tenant is not yet running the backend split, configuring `RECOVERY` is a no-op until the backend rolls it
out; no `terraform apply` is required at that point — the next refresh will reconcile state.

## Deprecations

### polaris_aws_cnp_account_attachments: `features` field

The `features` field on the `polaris_aws_cnp_account_attachments` resource is now deprecated. Permission groups for
each feature are read directly from the cloud account managed by `polaris_aws_cnp_account` when artifacts are
registered, so the attachments resource no longer needs to track them. This means new permission groups like
`RECOVERY` flow through automatically once they are configured on `polaris_aws_cnp_account`, with no schema change to
the attachments resource.

No action is required for existing configurations — the field is retained for backwards compatibility. You will see
a deprecation warning during `terraform plan`. The field will be removed in a future major release; at that point you
will be able to drop it entirely.
