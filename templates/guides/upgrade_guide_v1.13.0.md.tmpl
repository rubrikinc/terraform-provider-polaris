---
page_title: "Upgrade Guide: v1.13.0"
---

# Upgrade Guide v1.13.0

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.13.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.13.0"
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
If you get an error or an unwanted diff, see the _New Features_ section below for additional instructions. Otherwise,
refresh the state to the v1.13.0 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.13.0 version.

## New Features

### AWS accounts using multiple roles

RSC can onboard an AWS account using the IAM roles workflow with one IAM role per feature, instead of a single
`CROSSACCOUNT` role carrying the permissions of every feature. Which role mode an account uses is decided by RSC, and
fixed when the roles of the account are registered with the `polaris_aws_cnp_account_attachments` resource. The
`trust_policies` field of the `polaris_aws_cnp_account` resource always holds the trust policies of the role mode of the
account.

The `polaris_aws_cnp_artifacts` and `polaris_aws_cnp_permissions` data sources accept the AWS account ID in the new
`native_id` field. If omitted, RSC returns the legacy single-role artifacts and policies. If provided, RSC returns
artifacts and policies matching the account's trust policies: one role per feature in multi-role mode or a single
`CROSSACCOUNT` role in single-role mode. If the account has not been onboarded yet, RSC uses the role mode configured
for new accounts.

Pass the AWS account ID directly to both data sources as a value known at plan time, such as an input variable. This
lets Terraform read the role keys during planning and onboard the account in a single `terraform apply`:
```terraform
variable "account_id" {
  description = "AWS account ID."
  type        = string
}

locals {
  features = {
    CLOUD_DISCOVERY         = ["BASIC"]
    CLOUD_NATIVE_PROTECTION = ["BASIC"]
  }
}

data "polaris_aws_cnp_artifacts" "artifacts" {
  native_id = var.account_id

  dynamic "feature" {
    for_each = local.features
    content {
      name              = feature.key
      permission_groups = feature.value
    }
  }
}

data "polaris_aws_cnp_permissions" "permissions" {
  for_each  = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  native_id = var.account_id
  role_key  = each.key

  dynamic "feature" {
    for_each = local.features
    content {
      name              = feature.key
      permission_groups = feature.value
    }
  }
}

resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = var.account_id
  regions   = ["us-east-2"]

  dynamic "feature" {
    for_each = local.features
    content {
      name              = feature.key
      permission_groups = feature.value
    }
  }
}
```
Referencing `polaris_aws_cnp_account.account.native_id` instead creates a dependency on the account resource. If that
resource has pending changes, Terraform can defer the data reads until apply, leaving the role keys needed by
`for_each` unknown at plan time.

An account using multiple roles has a role for every feature, so role key names are longer than with a single role. A
role name prefix built from the full role key, e.g. `"rubrik-${lower(each.key)}-"`, is longer than the 38 characters
AWS allows for `CLOUD_NATIVE_DYNAMODB_PROTECTION`, and for `CLOUD_NATIVE_UEM_KEY_MANAGEMENT` when RSC has enabled that
feature on the account. Shorten the prefix, e.g.
`"${substr("rubrik-${lower(each.key)}", 0, 37)}-"`, which only truncates the role keys exceeding the limit. Role keys
within the limit keep the same prefix, so existing roles are not replaced.

Without `native_id`, the data sources behave as before. An account using multiple roles then gets trust policies for
one role per feature, but roles and policies for a single `CROSSACCOUNT` role, so the roles built from them do not
match.
