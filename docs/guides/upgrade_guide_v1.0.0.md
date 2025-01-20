---
page_title: "Upgrade Guide: v1.0.0"
---

# Upgrade Guide v1.0.0

## RSC provider new features
The v1.0.0 release introduces new data sources and resources to support the following new features:
* Azure Bring Your Own Kubernetes Exocompute, also known as BYOK and customer managed Exocompute.
  [[docs](../resources/azure_exocompute_cluster_attachment)], [[docs](../resources/azure_private_container_registry)]
* Data center Amazon S3 archival location.
  [[docs](../resources/data_center_archival_location_amazon_s3)]
* Data center AWS account. [[docs](../resources/data_center_aws_account)]
* Data center Azure subscription. [[docs](../resources/data_center_azure_subscription)]

## RSC provider changes
The v1.0.0 release introduces changes to the following data sources and resources:
* The authentication token cache can now be controlled by the `polaris` provider configuration.
* The `credentials` field of the `polaris` provider configuration is now optional. If omitted, the provider will look
  for credentials in the `RUBRIK_POLARIS_*` environment variables.
* The `credentials` field of the `polaris` provider configuration now accepts, in addition to what it already accepts,
  the content of an RSC service account credentials file.
* Add support for the Cloud Native Blob Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-cloud_native_blob_protection)]
* Add the `permissions` field to the `polaris_aws_cnp_account_attachments` resource. The `permissions` field should be
  used with the `id` field of the `polaris_aws_cnp_permissions` data source to trigger an update of the resource
  whenever the permissions changes. This update will move the RSC cloud account from the missing permissions state.
* Fix a bug in the `polaris_aws_cnp_permissions` data source where the data source's id was accidentally calculated for
  the complete set of role keys and not just the specified role key.
* Add the field `manifest` to the `polaris_aws_exocompute_cluster_attachment` resource. The `manifest` field contains
  a Kubernetes manifest that can be passed to the Kubernetes Terraform provider or `kubectl` to establish a connection
  between the AWS EKS cluster and RSC. [[docs](../resources/aws_exocompute_cluster_attachment)]
* Deprecate the `setup_yaml` field in the `polaris_aws_exocompute_cluster_attachment` resource. Use the `manifest` field
  instead.

Deprecated fields will be removed in a future release, please migrate your configurations to use the replacement field
as soon as possible.

## Known issues
* The user-assigned managed identity for `cloud_native_archival_encryption` is not refreshed when the
  `polaris_azure_subscription` resource is updated. This will be fixed in a future release.

In addition to the issues listed above, affecting this particular release of the provider, additional issues reported
can be found on [GitHub](https://github.com/rubrikinc/terraform-provider-polaris/issues).

## How to upgrade
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
Next, upgrade the Terraform provider to the new version by running:
```bash
$ terraform init -upgrade
```
After the Terraform provider has been updated, validate the correctness of the Terraform configuration files by running:
```bash
$ terraform plan
```
If this doesn't produce an error or unwanted diff, proceed by running:
```bash
$ terraform apply
```
This will read the remote state of the resources, migrate the local Terraform state to version v1.0.0 and apply any
outstanding changes to the remote state.

## Upgrade issues
When upgrading to the v1.0.0 release you may encounter one or more of the following issues.

### polaris_aws_cnp_account_attachments
Setting the `permissions` field of a `polaris_aws_cnp_account_attachments` resource to the `id` field of the
`polaris_aws_cnp_permissions` data source will result in a diff similar to this:
```hcl
# polaris_aws_cnp_account_attachments.attachments will be updated in-place
~ resource "polaris_aws_cnp_account_attachments" "attachments" {
      id         = "9dad45e3-dbe5-4d49-a24c-ea6b83062dac"
      # (2 unchanged attributes hidden)

    - role {
        - arn         = "arn:aws:iam::123456789012:role/rubrik-crossaccount-20250128102507017200000001" -> null
        - key         = "CROSSACCOUNT" -> null
          # (1 unchanged attribute hidden)
      }
    + role {
        + arn         = "arn:aws:iam::123456789012:role/rubrik-crossaccount-20250128102507017200000001"
        + key         = "CROSSACCOUNT"
        + permissions = "146fdc8ee0de5d762efd853d1ac50bdfdb2f3d5dacdfcc76e6a0268ec760f928"
      }
  }
```
This is expected since the new `permissions` field is being updated. Applying the diff will update the
`polaris_aws_cnp_account_attachments` in place. If the AWS account is in a `MISSING_PERMISSIONS` state, the account will
be moved to the `CONNECTED` state.

### polaris_azure_subscription
Because of the new Azure permission groups support, RSC feature fields will result in a diff similar to this:
```hcl
# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
      id                          = "31c7fd10-6c0e-410a-a7a8-0d7fd395852b"
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
To remove the diff, copy the `permission_groups` values from the diff and add them to the Terraform configuration. Note,
for the `cloud_native_protection` field, you most likely don't need the `CLOUD_CLUSTER_ES` permission group. Removing it
will greatly reduce the number of permissions granted to RSC.

If the Azure Blob Protection feature has already been onboarded using the RSC UI, a diff similar to this will occur:
```hcl
# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
      id                          = "31c7fd10-6c0e-410a-a7a8-0d7fd395852b"
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
  }
```
To remove the diff, copy the `permission_groups` and `regions` values from the diff and add them to the Terraform
configuration.
