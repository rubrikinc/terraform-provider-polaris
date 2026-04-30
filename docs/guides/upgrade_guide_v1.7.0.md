---
page_title: "Upgrade Guide: v1.7.0"
---

# Upgrade Guide v1.7.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

The v1.7.0 release is also published as the renamed `rubrikinc/rubrik` provider. The `rubrikinc/polaris` provider will
continue to be released and supported for some time, so there is no need to switch right now. The
`rubrikinc/polaris` provider will eventually be retired, however, and you will need to switch to the `rubrikinc/rubrik`
provider before then. The migration paths will improve over time as more resources gain support for Terraform's
`moved {}` block, making the switch progressively simpler. See the
[v1.7.0 upgrade guide for the rubrikinc/rubrik provider](https://registry.terraform.io/providers/rubrikinc/rubrik/latest/docs/guides/upgrade_guide_v1.7.0)
for the currently available migration paths.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.7.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.7.0"
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
If you get an error or an unwanted diff, please see the _New Features_ and _Bug Fixes_ sections below for additional
instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.7.0 version.

## New Features

### Initial Terraform Search Support

An initial set of three resource types now support Terraform list resources, allowing them to be discovered with
`terraform query`. This includes resources that are not managed by Terraform. Additional resource types will gain
`terraform query` support in future releases. For background on the feature, see HashiCorp's blog post
[Terraform Search and Import: Find resources and bring them into Terraform](https://www.hashicorp.com/en/blog/terraform-search-and-import-find-resources-and-bring-them-into-terraform).

#### polaris_custom_role

List all custom roles in RSC, or filter by name:
```terraform
list "polaris_custom_role" "all" {
  provider = polaris
}

list "polaris_custom_role" "by_name" {
  provider = polaris

  config {
    name = "Compliance Auditor"
  }
}
```

#### polaris_user

List all users in RSC, or filter by email:
```terraform
list "polaris_user" "all" {
  provider = polaris
}

list "polaris_user" "by_email" {
  provider = polaris

  config {
    email = "auditor@example.org"
  }
}
```

#### polaris_sso_group

List all SSO groups in RSC, or filter by name and optionally by auth domain ID:
```terraform
list "polaris_sso_group" "all" {
  provider = polaris
}

list "polaris_sso_group" "by_name_and_domain" {
  provider = polaris

  config {
    name           = "Auditors"
    auth_domain_id = "1a5629cb-2681-4ea4-b36c-ea8b2f3990cd"
  }
}
```

## Bug Fixes

### polaris_sla_domain: object-specific config drift fix

Optional retention unit fields in the `sap_hana_config`, `db2_config`, `oracle_config` and `informix_config` blocks of
the `polaris_sla_domain` resource now mirror the schema default when the matching duration is unset, eliminating
spurious diffs after apply. In addition, the `storage_snapshot_config` block in `sap_hana_config` is only emitted when
it has data, and omitted retention fields in all object-specific configuration blocks are left out of the API request
instead of being sent with empty values.

### polaris_sla_domain: db2_config.log_archival_method default

The `log_archival_method` field in the `db2_config` block of the `polaris_sla_domain` resource now defaults to
`LOGARCHMETH1`, matching the RSC backend default. Previously, omitting the field produced a drift on subsequent plans
because the API returned `LOGARCHMETH1` while the schema treated the field as unset.
