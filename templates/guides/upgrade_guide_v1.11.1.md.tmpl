---
page_title: "Upgrade Guide: v1.11.1"
---

# Upgrade Guide v1.11.1

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.11.1 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.11.1"
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
If you get an error or an unwanted diff, see the _Significant Changes_ section below for additional instructions.
Otherwise, refresh the state to the v1.11.1 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.11.1 version.

## Significant Changes

### Security group fields in the AWS Exocompute resource are no longer deprecated

The deprecation of the `cluster_security_group_id` and `node_security_group_id` fields in the `polaris_aws_exocompute`
resource, introduced in the [v1.10.0 upgrade guide](upgrade_guide_v1.10.0.md), has been removed. RSC will continue to
support security groups passed in by users, so the fields no longer produce a deprecation warning and do not need to be
removed from the configuration.

No configuration changes are required. Configurations that set the fields continue to work as before, and
configurations that removed the fields to resolve the deprecation warning continue to let RSC create and manage the
security groups.
