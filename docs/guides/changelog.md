---
page_title: "Changelog"
---

# Changelog

## v0.10.0-beta.10
* Fix a type conversion error in the `polaris_aws_exocompute` resource. During a prior refactoring, a new type was
  introduced for AWS regions to handle cases where the same region has multiple representations in the GraphQL API.
  This type was not properly converted in all code paths.

## v0.10.0-beta.9
* Fix a bug in the `polaris_aws_cnp_permissions` data source where the data source's id was accidentally calculated for
  the complete set of role keys and not just the specified role key.

## v0.10.0-beta.8
* Add the `permissions` field to the `polaris_aws_cnp_account_attachments` resource. The `permissions` field should be
  used with the `id` field of the `polaris_aws_cnp_permissions` data source to trigger an update of the resource
  whenever the permissions changes. This update will move the RSC cloud account from the missing permissions state.

## v0.10.0-beta.7
* Add support for Azure Bring Your Own Kubernetes Exocompute, also known as BYOK and customer managed Exocompute.
  [[docs](../resources/azure_exocompute_cluster_attachment)], [[docs](../resources/azure_private_container_registry)]

## v0.10.0-beta.6
* Add support for the Cloud Native Blob Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-cloud_native_blob_protection)]

## v0.10.0-beta.5
* The `data_center_archival_location_amazon_s3` resource will now monitor and wait for the asynchronous CDM operations
  to finish.

## v0.10.0-beta.4
* Fix a regression in the cloud native archival location resources. An extra level of structure in the RSC response
  caused resource refreshes to fail.

## v0.10.0-beta.3
* Add support for creating data center AWS accounts. [[docs](../resources/data_center_aws_account)]
* Add support for creating data center Azure subscriptions. [[docs](../resources/data_center_azure_subscription)]
* Add support for creating Amazon S3 data center archival locations.
  [[docs](../resources/data_center_archival_location_amazon_s3)]
* Add `polaris_data_center_aws_account` data source. [[docs](../data-sources/data_center_aws_account)]
* Add `polaris_data_center_azure_subscription` data source. [[docs](../data-sources/data_center_azure_subscription)]

## v0.10.0-beta.2
* Add the field `manifest` to the `polaris_aws_exocompute_cluster_attachment` resource. The `manifest` field contains
  a Kubernetes manifest that can be passed to the Kubernetes Terraform provider or `kubectl` to establish a connection
  between the AWS EKS cluster and RSC. [[docs](../resources/aws_exocompute_cluster_attachment)]
* Deprecate the `setup_yaml` field in the `polaris_aws_exocompute_cluster_attachment` resource. Use the `manifest` field
  instead.

## v0.10.0-beta.1
* The authentication token cache can now be controlled by the `polaris` provider configuration.
* The `credentials` field of the `polaris` provider configuration now accepts, in addition to what it already accepts,
  the content of an RSC service account credentials file.

## v0.9.0
* Update the `polaris_aws_archival_location` resource to support updates of the `bucket_tags` field without recreating
  the resources.
* Add `polaris_aws_account` data source. [[docs](../data-sources/aws_account)]
* Add `polaris_azure_subscription` data source. [[docs](../data-sources/azure_subscription)]
* Deprecate the `archival_location_id` field in the `polaris_aws_archival_location` data source. Use the `id` field
  instead.
* Deprecate the `archival_location_id` field in the `polaris_azure_archival_location` data source. Use the `id` field
  instead.
* Add the field `setup_yaml` to the `polaris_aws_exocompute_cluster_attachment` resource. The `setup_yaml` field
  contains K8s specs that can be passed to `kubectl` to establish a connection between the AWS EKS cluster and RSC.
  [[docs](../resources/aws_exocompute_cluster_attachment)]
* Fix a bug in the AWS feature removal code that causes removal of the `CLOUD_NATIVE_S3_PROTECTION` feature to fail.
* Improve the code that waits for RSC features to be disabled. The code now checks both the status of the job and the
  status of the cloud account.
* Improve the documentation for AWS data sources and resources.
* Update guides.
* Add `polaris_azure_archival_location` data source. [[docs](../data-sources/azure_archival_location)]
* Fix a bug in the `polaris_azure_archival_location` resource where the cloud account UUID would be passed to the RSC
  API instead of the Azure subscription UUID when creating an Azure archival location.
* Fix a bug in the `polaris_aws_cnp_account` resource where destroying it would constantly result in an *objects not
  authorized* error.
* Increase the wait time for asynchronous RSC operations to 8.5 minutes.
* Fix an issue with the permissions of subscriptions onboarded using the `polaris_azure_subscription` resource where
  the RSC UI would show the status as "Update permissions" even though the app registration would have all the required
  permissions.
* Move changelog and upgrade guides to guides folder.
* Add support for creating Azure cloud native archival locations. [[docs](../resources/azure_archival_location)]
* Fix a bug in the `polaris_aws_exocompute` resource where customer supplied security groups were not validated
  correctly.
* Add support for shared Exocompute to the `polaris_azure_exocompute` resource.
  [[docs](../resources/azure_exocompute#host_cloud_account_id)]
* Add the `polaris_account` data source. [[docs](../data-sources/account)]
* Add support for the Cloud Native Archival feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-cloud_native_archival)]
* Add support for the Cloud Native Archival Encryption feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-cloud_native_archival_encryption)]
* Add support for the Azure SQL Database Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-sql_db_protection)]
* Add support for the Azure SQL Managed Instance Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#nested-schema-for-sql_mi_protection)]
* Add support for specifying an Azure resource group when onboarding the Cloud Native Archival, Cloud Native Archival
  Encryption, Cloud Native Protection or Exocompute features using the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription#optional)]
