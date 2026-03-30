---
page_title: "Upgrade Guide: v1.6.0"
---

# Upgrade Guide v1.6.0

## New Features

### polaris_sso_group resource

The new `polaris_sso_group` resource creates and manages SSO groups in RSC. It supports assigning roles to SSO groups
and importing existing groups using the `<group_name>:<identity_provider_id>` format.

```terraform
data "polaris_identity_provider" "example" {
  name = "My IdP"
}

resource "polaris_sso_group" "example" {
  group_name     = "mygroup"
  auth_domain_id = data.polaris_identity_provider.example.identity_provider_id

  role_ids = [
    polaris_custom_role.viewer.id,
  ]
}
```

For more details, see the [polaris_sso_group documentation](../resources/sso_group.md).

### polaris_identity_provider data source

The new `polaris_identity_provider` data source looks up identity providers configured in RSC by ID or name. This is
useful for referencing identity providers when configuring SSO group resources.

For more details, see the [polaris_identity_provider documentation](../data-sources/identity_provider.md).

### polaris_refresh resource

The new `polaris_refresh` resource polls until an account or subscription's inventory refresh in RSC is newer than a
given timestamp. This ensures leaf objects like VMs and EC2 instances are discoverable via `polaris_object` after
onboarding.

For more details, see the [polaris_refresh documentation](../resources/refresh.md).

### polaris_aws_account: role chaining support

The `polaris_aws_account` resource now supports the `role_chaining` feature block for cross-account role chaining. The
feature is mutually exclusive with all other features. A new `role_chaining_account_id` field allows referencing the RSC
cloud account ID of an account with the Role Chaining feature enabled.

For more details, see the [polaris_aws_account documentation](../resources/aws_account.md).

### polaris_object: additional workload types

The `polaris_object` data source now supports `AwsNativeEbsVolume`, `AwsNativeEc2Instance`, `AwsNativeRdsInstance` and
`AzureNativeVirtualMachine` workload types. These workload-level types use server-side filters to exclude inactive
objects.

For more details, see the [polaris_object documentation](../data-sources/object.md).

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note that deprecated resources and fields will be removed in a future release. Please migrate your
configurations to use the recommended replacements as soon as possible.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.6.0 release. One
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
If you get an error or an unwanted diff, please see the _Significant Changes_ section below for additional instructions.
Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.6.0 version.

## Significant Changes

### polaris_aws_account: permission_groups required in all feature blocks

The `permission_groups` field is now required in the `cloud_native_protection` and `exocompute` blocks of the
`polaris_aws_account` resource. Previously, `permission_groups` was optional for these two blocks to remain backwards
compatible with older configurations that predated the field. This inconsistency has now been addressed.

If your configuration omits `permission_groups` from a `cloud_native_protection` or `exocompute` block, an error similar
to this will be produced:
```
╷
│ Error: Missing required argument
│
│   on main.tf line 43, in resource "polaris_aws_account" "account":
│   43: resource "polaris_aws_account" "account" {
│
│ The argument "permission_groups" is required, but no definition was found.
╵
```

To fix this, add `permission_groups = ["BASIC"]` to the affected block(s). For example:
```terraform
resource "polaris_aws_account" "example" {
  profile = "example"

  cloud_native_protection {
    permission_groups = ["BASIC"]

    regions = [
      "us-east-2",
    ]
  }
}
```
