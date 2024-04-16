---
page_title: "Upgrade Guide: beta release"
subcategory: "Upgrade"
---

# RSC provider beta changes
The latest beta release introduces changes to the following data sources and resources:
* `polaris_azure_permissions` - The `hash` field has been deprecated and replaced with the `id` field. Both fields will
  have same value until the `hash` field is removed, in a future release.
* `polaris_azure_exocompute` - The `subscription_id` field has been deprecated and replaced with the `cloud_account_id`
  field. The `subscription_id` field referred to the ID of the `polaris_azure_subscription` resource and not the Azure
  subscription ID, which was confusing. Note, changing an existing `polaris_azure_exocompute` resource to use the
  `cloud_account_id` field will recreate the resource.
* `polaris_azure_service_principal` - The `permissions_hash` field has been deprecated and replaced with the
  `permissions` field. With the changes in the `polaris_azure_permissions` data source, use
   `permissions = data.polaris_azure_permissions.<name>.id` to connect the `polaris_azure_permissions` data source to
   the permissions updated signal.
* `polaris_azure_subscription` - Support for onboarding `cloud_native_archival`, `cloud_native_archival_encryption`,
  `sql_db_protection` and `sql_mi_protection`. Note, there is no additional Terraform resources for managing the
   features yet. Support for specifying an Azure resource group per RSC feature.

Deprecated fields will be removed in a future release, please migrate your configurations to use the replacement field
as soon as possible.

# Upgrade to the latest beta release
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

## Upgrade issues
When upgrading to the latest beta release you may encounter one or more of the following issues.

### polaris_azure_exocompute
Replacing the `subscription_id` field with the `cloud_account_id` field will result in the `polaris_azure_exocompute`
resource being recreated, a diff similar to the following will be shown:
```hcl
  # polaris_azure_exocompute.default must be replaced
-/+ resource "polaris_azure_exocompute" "default" {
      + cloud_account_id = "a677433c-954c-4af6-842e-0268c4a82a9f" # forces replacement
      ~ id               = "45d68b3f-a78f-4098-922e-367d2a22cb92" -> (known after apply)
      - subscription_id  = "a677433c-954c-4af6-842e-0268c4a82a9f" -> null # forces replacement
        # (2 unchanged attributes hidden)
    }
```
Apply the diff to recreate the resource and replace the field.

### polaris_azure_service_principal
Replacing the `permissions_hash` field with the `permissions` field will result in the resource being updated in-place,
a diff similar to the following will be shown:
```hcl
# polaris_azure_service_principal.default will be updated in-place
~ resource "polaris_azure_service_principal" "default" {
    id               = "6f35cc58-e1c9-445d-8bb0-a0e30dd53a40"
  + permissions      = "0a79e15a989ef9a5191fe9fba62f40f5bd7f7062a90fbe367b29d1ae3dd34e50"
  - permissions_hash = "0a79e15a989ef9a5191fe9fba62f40f5bd7f7062a90fbe367b29d1ae3dd34e50" -> null
    # (2 unchanged attributes hidden)
}
```
Apply the diff to replace the field.

### polaris_azure_subscription
Because of the new Azure resource group support, using the `cloud_native_protection` or `exocompute` fields will result
in a diff similar to the following:
```hcl
# polaris_azure_subscription.default will be updated in-place
~ resource "polaris_azure_subscription" "default" {
    id                          = "f7b298c4-bf1d-4af4-900e-bf69ddfc6187"
    # (4 unchanged attributes hidden)

  ~ cloud_native_protection {
      - resource_group_name   = "RubrikBackups-RG-DontDelete-9f68a830-36a7-4363-9cf9-c81189fdc410" -> null
      - resource_group_region = "westus" -> null
        # (3 unchanged attributes hidden)
    }

  ~ exocompute {
      - resource_group_name   = "RubrikBackups-RG-DontDelete-e9ee0004-dcb2-4ec5-91b5-329c561c8311" -> null
      - resource_group_region = "westus" -> null
        # (3 unchanged attributes hidden)
    }
}
```
To remove the diff, copy the `resource_group_name` and `resource_group_region` values from the diff and add them to
their respective places in the Terraform configuration.
