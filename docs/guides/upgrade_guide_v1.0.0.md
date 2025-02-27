---
page_title: "Upgrade Guide: v1.0.0"
---

# Upgrade Guide v1.0.0

## Before Upgrading
Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note, deprecated resources and fields will be removed in a future release, please migrate your configurations
to use the recommended replacements as soon as possible.

## How to Upgrade
Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.0.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```hcl
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.0.0"
    }
  }
}
```
Next, upgrade the provider to the new version by running:
```bash
$ terraform init -upgrade
```
After the provider has been updated, validate the correctness of the Terraform configuration files by running:
```bash
$ terraform plan
```
If you get an error or an unwanted diff, please see the _Significant Changes and New Features_ below for additional
instructions. Otherwise, proceed by running:
```bash
$ terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.0.0 version.

## Significant Changes and New Features

### Cloud Native Blob Protection
Support for Cloud Native Blob Protection has been added to the `polaris_azure_subscription` resource. Since this feature
has just been added, it should normally not cause an issue when upgrading the provider. However, if the Cloud Native
Blob Protection feature has been added using the RSC UI, it may cause issues when upgrading. Note, the Cloud Native Blob
Protection feature requires the use of Permission Groups, describe further down in this document.

An example of how the `cloud_native_blob_protection` nested schema of the `polaris_azure_subscription` resource should
be used can be found in the
[azure](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L169)
example in the examples repository.

After upgrading the provider, if you have enabled the Cloud Native Blob Protection feature using the RSC UI, a diff
similar to this will occur:
```hcl
# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
    id                          = "60967b1e-20cb-4b61-acf6-454a55599b82"
    # (4 unchanged attributes hidden)

  - cloud_native_blob_protection {
      - permission_groups = [
          - "BASIC",
          - "RECOVERY",
        ] -> null
      - regions           = [
          - "eastus2",
        ] -> null
      - status            = "CONNECTED" -> null
        # (1 unchanged attribute hidden)
    }

    # (1 unchanged block hidden)
}
```
This is expected, since the `cloud_native_blob_protection` is not in the Terraform configuration. Do NOT apply the diff,
instead add the `cloud_native_blob_protection` definition that Terraform wants to remove to your configuration. Note,
the Cloud Native Blob Protection feature requires additional role definitions and role assignments. By passing the
`CLOUD_NATIVE_BLOB_PROTECTION` value to the `polaris_azure_permissions` data source, and using the
`polaris_azure_permissions` data source as input to the `azurerm_role_definition` and `azurerm_role_assignment`
resources, the required role definitions and role assignments will be created, see
[here](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L72),
[here](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L107)
and
[here](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L123)
in the example.

After updating the `polaris_azure_permissions` data source and adding the `cloud_native_blob_protection` nested schema
to the configuration, a diff similar to this will occur:
```hcl
# azurerm_role_assignment.resource_group["CLOUD_NATIVE_BLOB_PROTECTION"] will be created
+ resource "azurerm_role_assignment" "resource_group" {
  + id                               = (known after apply)
  + name                             = (known after apply)
  + principal_id                     = "32bbeaba-92b4-4162-9a69-0d39753b82c7"
  + principal_type                   = (known after apply)
  + role_definition_id               = (known after apply)
  + role_definition_name             = (known after apply)
  + scope                            = "/subscriptions/18677418-4fe7-43db-baf1-99646d610dd6/resourceGroups/terraform-azure-permissions-example"
  + skip_service_principal_aad_check = (known after apply)
}

# azurerm_role_assignment.subscription["CLOUD_NATIVE_BLOB_PROTECTION"] will be created
+ resource "azurerm_role_assignment" "subscription" {
  + id                               = (known after apply)
  + name                             = (known after apply)
  + principal_id                     = "32bbeaba-92b4-4162-9a69-0d39753b82c7"
  + principal_type                   = (known after apply)
  + role_definition_id               = (known after apply)
  + role_definition_name             = (known after apply)
  + scope                            = "/subscriptions/18677418-4fe7-43db-baf1-99646d610dd6"
  + skip_service_principal_aad_check = (known after apply)
}

# azurerm_role_definition.resource_group["CLOUD_NATIVE_BLOB_PROTECTION"] will be created
+ resource "azurerm_role_definition" "resource_group" {
  + assignable_scopes           = (known after apply)
  + id                          = (known after apply)
  + name                        = "Terraform3 - Azure Permissions Example Resource Group Level - CLOUD_NATIVE_BLOB_PROTECTION"
  + role_definition_id          = (known after apply)
  + role_definition_resource_id = (known after apply)
  + scope                       = "/subscriptions/18677418-4fe7-43db-baf1-99646d610dd6/resourceGroups/terraform-azure-permissions-example"
}

# azurerm_role_definition.subscription["CLOUD_NATIVE_BLOB_PROTECTION"] will be created
+ resource "azurerm_role_definition" "subscription" {
  + assignable_scopes           = (known after apply)
  + id                          = (known after apply)
  + name                        = "Terraform3 - Azure Permissions Example Subscription Level - CLOUD_NATIVE_BLOB_PROTECTION"
  + role_definition_id          = (known after apply)
  + role_definition_resource_id = (known after apply)
  + scope                       = "/subscriptions/18677418-4fe7-43db-baf1-99646d610dd6"

  + permissions {
      + actions      = [
          + "Microsoft.Insights/Metrics/Read",
          + "Microsoft.Resources/subscriptions/resourceGroups/read",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/delete",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/read",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/write",
          + "Microsoft.Storage/storageAccounts/delete",
          + "Microsoft.Storage/storageAccounts/read",
          + "Microsoft.Storage/storageAccounts/write",
        ]
      + data_actions = [
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/delete",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/manageOwnership/action",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/modifyPermissions/action",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/read",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/tags/read",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/tags/write",
          + "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/write",
        ]
      + not_actions  = []
    }
}

# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
    id                          = "60967b1e-20cb-4b61-acf6-454a55599b82"
    # (4 unchanged attributes hidden)

  ~ cloud_native_blob_protection {
      + permissions       = "b7dba84b286e4088f12b3a90852483add05b68f17be9cdab5e5eac055b6584d6"
        # (3 unchanged attributes hidden)
    }

    # (1 unchanged block hidden)
}
```
If the only thing changing is the `permissions` field of the nested `cloud_native_blob_protection` schema, along with
new Cloud Native Blob Protection role definitions and role assignments, the diff can be applied without any issues.

### New Permissions Field
A new `permissions` field has been added to the nested `role` schema of the `polaris_aws_cnp_account_attachments`
resource. This field should be used with the `id` field of the `polaris_aws_cnp_permissions` data source to trigger an
update of the resource whenever the permissions changes. This update will move the RSC cloud account from the missing
permissions state.

An example of how the `permissions` field should be used can be found in the
[aws_cnp_account](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/aws_cnp_account/main.tf#L172)
example in the examples repository.

The new `permissions` field is optional and should not cause a diff when upgrading the provider. However, if the
Terraform configuration is updated to use the new `permissions` field, a diff similar to this one will occur:
```hcl
# polaris_aws_cnp_account_attachments.attachments will be updated in-place
~ resource "polaris_aws_cnp_account_attachments" "attachments" {
    id         = "55cdd9de-1030-4fbb-b10c-8e703d98f1cb"
    # (2 unchanged attributes hidden)

  - role {
      - arn         = "arn:aws:iam::123456789012:role/rubrik-crossaccount-20250221151913448000000003" -> null
      - key         = "CROSSACCOUNT" -> null
        # (1 unchanged attribute hidden)
    }
  - role {
      - arn         = "arn:aws:iam::123456789012:role/rubrik-exocompute_eks_masternode-20250221151913441600000001" -> null
      - key         = "EXOCOMPUTE_EKS_MASTERNODE" -> null
        # (1 unchanged attribute hidden)
    }
  - role {
      - arn         = "arn:aws:iam::123456789012:role/rubrik-exocompute_eks_workernode-20250221151913442900000002" -> null
      - key         = "EXOCOMPUTE_EKS_WORKERNODE" -> null
        # (1 unchanged attribute hidden)
    }
  + role {
      + arn         = "arn:aws:iam::123456789012:role/rubrik-crossaccount-20250221151913448000000003"
      + key         = "CROSSACCOUNT"
      + permissions = "bd2b39938dc306d3cda3d5a29fbc1616a0e4db1f69c6603a0d36b8244f5389ee"
    }
  + role {
      + arn         = "arn:aws:iam::123456789012:role/rubrik-exocompute_eks_masternode-20250221151913441600000001"
      + key         = "EXOCOMPUTE_EKS_MASTERNODE"
      + permissions = "7a6d52cb96fd481a3ee7233fa69c6feecea3a6b4c5819e29c5e2e40e384e5946"
    }
  + role {
      + arn         = "arn:aws:iam::123456789012:role/rubrik-exocompute_eks_workernode-20250221151913442900000002"
      + key         = "EXOCOMPUTE_EKS_WORKERNODE"
      + permissions = "5f78cb3b57ac46eea4cbb80fd3f7e78fed27be53b68a73405318f1b61f1df3b4"
    }

    # (1 unchanged block hidden)
}
```
If the only thing changing is the addition of the `permissions` field, the diff can be applied without any issues.

### Permission Groups
Support for Permission Groups has been added to the `polaris_azure_permissions` data source and the
`polaris_azure_subscription` resource. Permission Groups are used to manage granular permissions of an RSC feature. New
RSC features, such as Cloud Native Blob Protection, require the use of Permission Groups. When Permission Groups are
used for an RSC feature, it should be used for all RSC features of the subscription.

An example of how Permission Groups should be used can be found in the
[azure](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure)
example in the examples repository. The RSC feature and its Permission Groups are passed to the
`polaris_azure_permissions` data source, seen
[here](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L75)
in the example. The `id` field of the `polaris_azure_permissions` data source is then assigned to the `permissions`
field of each of the RSC features configured by the `polaris_azure_subscription` resource, seen
[here](https://github.com/rubrikinc/terraform-provider-polaris-examples/blob/d2b0bf0b5458b3cd3ebcc6ab401a43f4daa89cd7/azure/main.tf#L181)
in the example.

After upgrading the provider, the `permission_groups` field of the `polaris_azure_subscription` resource will have a
diff for each RSC feature similar to this:
```hcl
# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
    id                          = "771de203-b9e1-4f79-8a96-3300e219bc21"
    # (4 unchanged attributes hidden)

  ~ cloud_native_protection {
      ~ permission_groups     = [
          - "BASIC",
          - "CLOUD_CLUSTER_ES",
          - "EXPORT_AND_RESTORE",
          - "FILE_LEVEL_RECOVERY",
          - "SNAPSHOT_PRIVATE_ACCESS",
        ]
        # (6 unchanged attributes hidden)
    }
}
```
This is expected, since no Permission Groups have been specified in the configuration. Do NOT apply this diff, instead
add the Permission Groups that Terraform wants to remove to the `permission_groups` field of the
`polaris_azure_permissions` data source and assign the `permission_groups` field of the `polaris_azure_permissions` data
source to the `permission_groups` field of each RSC feature of the `polaris_azure_subscription` resource. When the
configuration has been updated correctly, there will be no diff when running `terraform plan`.
