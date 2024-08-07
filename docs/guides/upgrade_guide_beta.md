---
page_title: "Upgrade Guide: beta release"
---

# Upgrade Guide Beta Release
~> **Note:** The beta provider might have breaking changes between beta releases.

## RSC provider changes
The latest beta release introduces changes to the following data sources and resources:
  * Add 3 new fields to the `polaris` provider configuration: `token_cache`, `token_cache_dir` and `token_cache_secret`.
    The fields can be used to control the authentication token cache through the provider configuration. The cache can
    already be controlled through environment variables.
  * Update the  `credentials` field of the `polaris` provider configuration to accept, in addition to what it already
    accepts, the content of an RSC service account credentials file.

Deprecated fields will be removed in a future release, please migrate your configurations to use the replacement field
as soon as possible.

## How to upgrade
Start by assigning the version of the latest beta release to the `version` field in the `provider` block of the
Terraform configuration:
```hcl
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "=<beta-version>
    }
  }
}
```
With beta releases, it's important the version is pinned to the exact version number otherwise Terraform will not find
the version in the Terraform registry. Next, upgrade the Terraform provider to the new version by running:
```bash
$ terraform init -upgrade
```
After the Terraform provider has been updated, validate the correctness of the Terraform configuration files by running:
```bash
$ terraform plan
```
If this doesn't produce an error or unwanted diff, proceed by running:
```bash
$ terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the latest beta version.
