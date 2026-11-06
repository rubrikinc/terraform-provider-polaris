---
page_title: "Upgrade Guide: v1.12.0"
---

# Upgrade Guide v1.12.0

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.12.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.12.0"
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
Otherwise, refresh the state to the v1.12.0 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.12.0 version.

## Significant Changes

### GCP Cloud Discovery

The `polaris_gcp_project` resource now supports the `CLOUD_DISCOVERY` feature. Cloud Discovery is the read-only parent
of the GCP protection features. When it is enabled for the RSC account, the RSC UI adds it to a project whenever a
protection feature is onboarded, without showing it as a separate choice. RSC refuses to remove it from a project while
a protection feature is still enabled on the project.

Previous versions of the provider read the feature from RSC into the state but rejected it in the configuration, so a
project with Cloud Discovery enabled always showed a plan removing it, and applying that plan failed in RSC. This
version accepts the feature in the configuration and stops a plan which removes it while a protection feature remains,
with an error like:
```
Error: feature CLOUD_DISCOVERY cannot be removed while the protection feature CLOUD_NATIVE_PROTECTION is enabled
```
If you get this error for a project you didn't change, Cloud Discovery is enabled on the project in RSC but not in the
configuration. Add it to the `polaris_gcp_project` resource:
```terraform
resource "polaris_gcp_project" "project" {
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012

  feature {
    name              = "CLOUD_DISCOVERY"
    permission_groups = ["BASIC"]
  }

  feature {
    name              = "CLOUD_NATIVE_PROTECTION"
    permission_groups = ["BASIC"]
  }
}
```
To remove Cloud Discovery, first remove the protection features from the project, then remove Cloud Discovery in a
later apply. When the project is destroyed, the provider removes Cloud Discovery last.

Projects without Cloud Discovery enabled are not affected.
