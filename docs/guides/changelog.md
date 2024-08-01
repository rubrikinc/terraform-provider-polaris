---
page_title: "Changelog"
---

# Changelog

## v0.9.0-beta.11
* Update the `polaris_aws_archival_location` resource to support updates of the `bucket_tags` field without recreating
  the resources.

## v0.9.0-beta.10
* Add `polaris_aws_account` data source. [[docs](../data-sources/aws_account)]
* Add `polaris_azure_subscription` data source. [[docs](../data-sources/azure_subscription)]
* Deprecate the `archival_location_id` field in the `polaris_aws_archival_location` data source. Use the `id` field
  instead.
* Deprecate the `archival_location_id` field in the `polaris_azure_archival_location` data source. Use the `id` field
  instead.

## v0.9.0-beta.9
* Add the field `setup_yaml` to the `polaris_aws_exocompute_cluster_attachment` resource. The `setup_yaml` fields
  contains K8s specs that can be passed to `kubectl` to establish a connection between the cluster and RSC.
  [[docs](../resources/aws_exocompute_cluster_attachment)]
* Fix a bug in the AWS feature removal code that causes removal of the `CLOUD_NATIVE_S3_PROTECTION` feature to fail.
* Improve the code that waits for RSC features to be disabled. The code now checks both the status of the job and the
  status of the cloud account.

## v0.9.0-beta.8
* Improve the documentation for AWS data sources and resources.
* Update guides.

## v0.9.0-beta.7
* Add `polaris_azure_archival_location` data source. [[docs](../data-sources/azure_archival_location)]
* Fix a bug in the `polaris_azure_archival_location` resource where the cloud account UUID would be passed to the RSC
  API instead of the Azure subscription UUID when creating an Azure archival location.
* Fix a bug in the `polaris_aws_cnp_account` resource where destroying it would constantly result in an *objects not
  authorized* error.
* Increase the wait time for asynchronous RSC operations to 8.5 minutes.

## v0.9.0-beta.6
* Fix an issue with the permissions of subscriptions onboarded using the `polaris_azure_subscription` resource where
  the RSC UI would show the status as "Update permissions" even though the app registration would have all the required
  permissions.

## v0.9.0-beta.5
* Move changelog and upgrade guides to guides folder.

## v0.9.0-beta.4
* Add support for creating Azure cloud native archival locations. [[docs](../resources/azure_archival_location)]

## v0.9.0-beta.3
* Fix a bug in the `polaris_aws_exocompute` resource where customer supplied security groups were not validated
  correctly.

## v0.9.0-beta.2
* Add support for shared Exocompute to the `polaris_azure_exocompute` resource.
  [[docs](../resources/azure_exocompute#host_cloud_account_id)]
* Add the `polaris_account` data source. [[docs](../data-sources/account)]

## v0.9.0-beta.1
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
