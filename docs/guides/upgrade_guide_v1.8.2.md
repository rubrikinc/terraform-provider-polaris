---
page_title: "Upgrade Guide: v1.8.2"
---

# Upgrade Guide v1.8.2

The v1.8.2 release adds a plan-time validation to the `polaris_custom_role` resource: a role that grants the
`VIEW_CLUSTER` operation must now also grant `VIEW_CLUSTER_REFERENCE`. See the [changelog](changelog.md) for the full
list of changes.

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

~> **Note:** If you are upgrading across multiple minor versions (e.g. v1.7.x to v1.8.2), review the upgrade guide for
each intermediate version as well. Each guide documents breaking changes and migration steps specific to that release.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.8.2 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.8.2"
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
This will read the remote state of the resources and migrate the local Terraform state to the v1.8.2 version.

## Significant Changes

### Custom Role VIEW_CLUSTER Now Requires VIEW_CLUSTER_REFERENCE

The `polaris_custom_role` resource now validates, at plan time, that any role granting the `VIEW_CLUSTER` operation
also grants `VIEW_CLUSTER_REFERENCE`. RSC automatically grants `VIEW_CLUSTER_REFERENCE` whenever `VIEW_CLUSTER` is
granted, so a configuration that listed `VIEW_CLUSTER` alone never converged: every `terraform plan` reported the
auto-granted `VIEW_CLUSTER_REFERENCE` as a diff. `VIEW_CLUSTER_REFERENCE` may still be granted on its own — it is a
narrower permission that RSC does not expand.

If you have a role that grants `VIEW_CLUSTER`, add a matching `VIEW_CLUSTER_REFERENCE` permission block. Otherwise
`terraform plan` fails with:
```
Error: VIEW_CLUSTER requires VIEW_CLUSTER_REFERENCE
```
For example, a role that previously granted only `VIEW_CLUSTER`:
```terraform
resource "polaris_custom_role" "cluster_viewer" {
  name = "Cluster Viewer"

  permission {
    operation = "VIEW_CLUSTER"
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids     = ["CLUSTER_ROOT"]
    }
  }

  permission {
    operation = "VIEW_CLUSTER_REFERENCE"
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids     = ["CLUSTER_ROOT"]
    }
  }
}
```
After adding the `VIEW_CLUSTER_REFERENCE` permission, the previously perpetual diff disappears and the Terraform state
matches the configuration.
