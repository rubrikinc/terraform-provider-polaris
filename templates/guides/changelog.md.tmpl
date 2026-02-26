---
page_title: "Changelog"
---

# Changelog

## v1.5.0

* Add support for managing the AWS Outpost account as a separate `polaris_aws_account` resource. The
  `outpost_account_id` and `outpost_account_profile` fields have been made optional.
* Add support for the following feature blocks in the `polaris_aws_account` resource: `cloud_discovery`,
  `cloud_native_archival`, `cloud_native_dynamodb_protection`, `cloud_native_s3_protection`, `kubernetes_protection`,
  `rds_protection`, and `servers_and_apps`. [[docs](../resources/aws_account.md)]
* The `cloud_native_protection` feature block has changed from required to optional in the `polaris_aws_account`
  resource.
* Remove the default value from the `cluster_access` field of the `polaris_aws_exocompute` resource. The default value
  would be set, but have no effect, when creating a shared Exocompute configuration.
* Remove the in-place update functionality from the `polaris_aws_exocompute` resource due to API issues. The in-place
  update functionality was only used for the `cluster_access` field. Updating the field now requires the resource to be
  re-created.
* Add support for user-assigned managed identity in the `polaris_azure_subscription` resource for the SQL DB Protection
  feature. The managed identity fields (`user_assigned_managed_identity_name`, `user_assigned_managed_identity_principal_id`,
  `user_assigned_managed_identity_region`, and `user_assigned_managed_identity_resource_group_name`) can be specified
  directly in the `sql_db_protection` block. This is required when using Transparent Data Encryption (TDE) with customer
  managed keys. [[docs](../resources/azure_subscription.md#nested-schema-for-sql_db_protection)]
  directly in the `sql_db_protection` block. These fields are required once the RSC account has the
  `CNP_AZURE_SQL_DB_TDE_CMK_SUPPORT` feature flag enabled for Transparent Data Encryption (TDE) with customer managed keys.
  Specifying these fields before the feature flag is enabled will result in an error. Supports upgrade scenarios where the
  feature flag is enabled on existing SQL DB Protection configurations.
  [[docs](../resources/azure_subscription.md#nested-schema-for-sql_db_protection)]

## v1.4.0

**SLA Domain Management:**
* Add `polaris_sla_domain` resource for managing RSC global SLA Domains. The resource supports creating and updating
  SLA domains with frequency, retention, archival, and replication configurations.
  [[docs](../resources/sla_domain.md)]
* Add support for the following object types with specific configurations in SLA domains: vSphere Object, KUPR,
  SAP HANA, Microsoft SQL Server, Db2, Oracle, Mongo, Managed Volume, PostgreSQL, MySQL, NCD, Informix, GCP Cloud SQL,
  Azure SQL Databases, Azure SQL Managed Instance, Azure Blob Storage, AWS RDS, AWS S3, AWS DynamoDB, GCE Instance/Disk,
  Okta, and Microsoft 365.
* Add support for the following object types without specific configurations in SLA domains: Linux and
  Windows Fileset, NAS, Active Directory, AWS EC2/EBS, Nutanix, HyperV, Exchange, VCD, Volume Group,
  OLVM, Cassandra, MongoDB, Azure AD, Azure DevOps, K8S, SnapMirror Cloud, Atlassian Jira, Salesforce,
  Google Workspace, D365, M365 Backup Storage, OpenStack, and Proxmox.
* Add support for `DoNotProtect` SLA assignment to the `polaris_sla_domain_assignment` resource. This explicitly tells
  RSC that a workload should not be protected, even if an inherited SLA would otherwise apply.
  [[docs](../resources/sla_domain_assignment.md)]
* Add support for backup windows in SLA domains to control when snapshots are taken.
* Add support for retention lock in SLA domains. Retention lock ensures that backups cannot be deleted or modified
  before the retention period expires.
* Update `polaris_sla_domain` data source with additional computed fields for archival specifications, and various
  schedule types (daily, hourly, minute, monthly, quarterly, weekly, yearly).
* Fix SLA schedule issues for certain object types.

**Archival and Replication:**
* Add support for replication pairs in the SLA replication specification. This allows configuring replication between
  specific source and target clusters.
* Add support for cascading archival in SLA domains. Cascading archival archives snapshots from a replicated cluster
  instead of directly from the source.
* Add support for cluster archival in SLA domains. This allows archiving snapshots to a CDM cluster archival location.
* Add support for data center archival tiering in SLA domains. This allows configuring tiering settings for data center
  archival locations including instant tiering and intelligent tiering.
* Add `polaris_ncd_archival_location` data source. The data source is used to look up NCD (Native Cloud Data) archival
  locations in RSC. [[docs](../data-sources/ncd_archival_location.md)]
* Add `polaris_data_center_archival_location` data source. The data source is used to look up data center archival
  locations in RSC. [[docs](../data-sources/data_center_archival_location.md)]
* Add `polaris_sla_source_cluster` data source. The data source is used to look up SLA source clusters in RSC.
  [[docs](../data-sources/sla_source_cluster.md)]
* Fix AWS archival location data source query. It is now possible to query existing AWS archival locations by name.

**Cloud Cluster Management:**
* Add `dynamic_scaling` field to the `polaris_aws_cloud_cluster` resource. The `dynamic_scaling` field can be used to
  enable dynamic scaling for the AWS cloud cluster. [[docs](../resources/aws_cloud_cluster.md)]
* Add `delete_cluster` field to the `polaris_aws_cloud_cluster` and `polaris_azure_cloud_cluster` resources. The
  `delete_cluster` field can be used to control whether the cloud cluster is deleted when the resource is destroyed.
  [[docs](../resources/aws_cloud_cluster.md)] [[docs](../resources/azure_cloud_cluster.md)]

**Permissions and Account Management:**
* Add `permission_groups` field to the `polaris_account` data source. The `permission_groups` field is used to look up
  permission groups for RSC features. [[docs](../data-sources/account.md)]
* Improve Azure permission groups for the `polaris_azure_subscription` resource and `polaris_azure_permissions` data
  source to include additional permissions.

**Cloud Exocompute**
* Add support for creating regional Exocompute configurations for GCP when using customer managed networking.
  [[docs](../resources/gcp_exocompute.md)]
* Add support for Azure Exocompute optional configuration. The optional configuration can be used to configure cluster
  tier, cluster access, etc. [[docs](../resources/azure_exocompute.md#nested-schema-for-optional_config)]
* Add support for AWS Exocompute optional configuration. The optional configuration can be used to configure cluster
  access including Private Exocompute. [[docs](../resources/aws_exocompute.md#optional)]

**Maintenance:**
* Update Go version to 1.25.6.
* Update Rubrik Polaris SDK for Go to v1.2.0.

## v1.4.0-beta.5
* Fix AWS archival location data source query. It is now possible to query existing AWS archival locations.

## v1.4.0-beta.4
* Add support for `DoNotProtect` SLA assignment to the `polaris_sla_domain_assignment` resource. This explicitly tells
  RSC that a workload should not be protected, even if an inherited SLA would otherwise apply.
* Add support for the following object types with specific configurations in SLA domains: vSphere Object, KUPR,
  SAP HANA, Microsoft SQL Server, Db2, Oracle, Mongo, Managed Volume, PostgreSQL, MySQL, NCD, Informix and GCP Cloud SQL.
* Add support for the following object types without specific configurations in SLA domains: Linux and
  Windows Fileset, NAS, Active Directory, AWS EC2/EBS, Nutanix, HyperV, Exchange, VCD, Volume Group,
  OLVM, Cassandra, MongoDB, Azure AD, Azure DevOps, K8S, SnapMirror Cloud, Atlassian Jira, Salesforce,
  Google Workspace, D365, M365 Backup Storage, OpenStack, and Proxmox.
* Add `polaris_ncd_archival_location` data source. The data source is used to look up NCD (Native Cloud Data) archival
  locations in RSC. [[docs](../data-sources/ncd_archival_location.md)]
* Add `polaris_dc_archival_location` data source. The data source is used to look up data center archival locations in
  RSC. [[docs](../data-sources/dc_archival_location.md)]
* Add `polaris_sla_source_cluster` data source. The data source is used to look up SLA source clusters in RSC.
  [[docs](../data-sources/sla_source_cluster.md)]
* Add support for retention lock in SLA domains. Retention lock ensures that backups cannot be deleted or modified
  before the retention period expires.
* Add support for replication pairs in the SLA replication specification. This allows configuring replication between
  specific source and target clusters.
* Add support for cascading archival in SLA domains. Cascading archival archives snapshots from a replicated cluster
  instead of directly from the source.
* Add support for cluster archival in SLA domains. This allows archiving snapshots to a CDM cluster archival location.
* Add support for data center archival tiering in SLA domains. This allows configuring tiering settings for data center
  archival locations including instant tiering and intelligent tiering.
* Fix SLA schedule issues for certain object types.

## v1.3.2
* Add `availability_zone` field to the `polaris_azure_cloud_cluster` resource. The `availability_zone` field can be used
  to specify the availability zone for the Azure cloud cluster. [[docs](../resources/azure_cloud_cluster.md)]
* Add support for bootstrapping Azure CCES clusters using user assigned managed identity. The
  `polaris_cdm_bootstrap_cces_azure` resource now supports `storage_account_name`, `storage_account_endpoint_suffix`,
  and `user_assigned_managed_identity_client_id` fields as an alternative to `connection_string`.
  [[docs](../resources/cdm_bootstrap_cces_azure.md)]
* Update Go version to 1.24.11.
* Update Rubrik Polaris SDK for Go to v1.1.13.

## v1.3.1
* Fix a bug in the `polaris_tag_rule` resource where the wrong ID was used when scoping a tag rule to a particular Azure
  subscription.
* Add support for onboarding the Kubernetes Protection RSC feature using the `polaris_aws_cnp_account` resource.

## v1.4.0-beta.1
* Add `polaris_sla_domain` resource for managing RSC global SLA Domains. The resource supports creating and updating
  SLA domains with frequency, retention, archival, and replication configurations.
  [[docs](../resources/sla_domain.md)]
* Add support for the following object types in SLA domains:
  - Azure SQL Databases with instant archival and continuous backup for point-in-time recovery
  - Azure SQL Managed Instance with log backup configuration
  - Azure Blob Storage with backup location configuration
  - AWS RDS with continuous backup for point-in-time recovery (archival supported for PostgreSQL and Aurora PostgreSQL)
  - AWS S3 with backup location configuration
  - AWS DynamoDB with primary backup encryption KMS key and continuous backup configuration
  - GCE Instance/Disk
  - Okta
  - Microsoft 365 (minimum 8 hours SLA, 12 hours or more recommended)
* Add support for backup windows in SLA domains to control when snapshots are taken.
* Update `polaris_sla_domain` data source with additional computed fields for archival specifications, and various
  schedule types (daily, hourly, minute, monthly, quarterly, weekly, yearly).

## v1.3.0
* Add support for GCP custom labels. [[docs](../resources/gcp_custom_labels.md)]
* Add support for GCP archival locations. [[docs](../data-sources/gcp_archival_location.md)]
  [[docs](../resources/gcp_archival_location.md)]
* Add `polaris_gcp_project` data source. [[docs](../data-sources/gcp_project.md)]
* The `id` field of the `polaris_gcp_service_account` resource is now the SHA-256 hash sum of the service account name.
  Note, this is a breaking change if a Terraform configuration expects the `id` field to be the service account name.
* The `credentials` field of the `polaris_gcp_service_account` resource can now be updated without recreating the
  resource.
* The `credentials` field of the `polaris_gcp_service_account` now accepts the credentials both as a file and as a
  base64 encoded string.
* The `credentials` field of the `polaris_gcp_service_account` resource is now marked as sensitive.
* Deprecate the `permissions_hash` field of the `polaris_gcp_service_account` resource. Use the `permissions` field
  of the `feature` block in the `polaris_gcp_project` resource instead.
* The `id` field of the `polaris_gcp_permissions` data source is now the SHA-256 hash sum of all the GCP permissions
  returned.
* Deprecate the `features` field of the `polaris_gcp_permissions` data source. Use the `feature` field instead.
* Deprecate the `hash` field of the `polaris_gcp_permissions` data source. Use the `id` field instead.
* Deprecate the `permissions` field of the `polaris_gcp_permissions` data source. Use the `with_conditions` and
  `without_conditions` fields instead.
* Add `feature` field to the `polaris_gcp_permissions` data source. The `feature` field is used to specify the RSC
  feature and permissions groups to look up the GCP permissions for.
* Add `conditions` field to the `polaris_gcp_permissions` data source. The `conditions` field holds the GCP conditions
  for the `with_conditions` GCP permissions.
* Add `services` field to the `polaris_gcp_permissions` data source. The `services` field holds the GCP services
  required for the RSC feature and permission groups.
* Add `with_conditions` field to the `polaris_gcp_permissions` data source. The `with_conditions` field holds the GCP
  permissions with conditions required for the RSC feature and permission groups.
* Add `without_conditions` field to the `polaris_gcp_permissions` data source. The `without_conditions` field holds the
  GCP permissions without conditions required for the RSC feature and permission groups.
* Deprecate the `cloud_native_protection` field of the `polaris_gcp_project` resource. Use the `feature` field instead.
  Using the `feature` field, setting `name` to `CLOUD_NATIVE_PROTECTION` and `permission_groups` to `BASIC`,
  `EXPORT_AND_RESTORE` and `FILE_LEVEL_RECOVERY` should be equivalent.
* Deprecate the `permissions_hash` field of the `polaris_gcp_project` resource. Use the `permissions` field of the
  `feature` block instead.
* Add `feature` field to the `polaris_gcp_project` resource. The `feature` field is used to specify one or more RSC
  features with permissions groups to onboard the GCP project with.
* The `credentials` field of the `polaris_gcp_project` resource is now marked as optional and sensitive.
* The `credentials` field of the `polaris_gcp_project` now accepts the credentials both as a file and as a base64
  encoded string.
* The `project`, `project_name`, and `project_number` fields of the `polaris_gcp_project` resource are now marked as
  required. Note, this is a breaking change if they are not specified in a Terraform configuration. Use the
  `terraform state show <resource-address>` command to look up the values to add to the configuration.
* Adds `servers_and_apps` field to the `polaris_azure_subscription` resource. The `servers_and_apps` field is used to
  specify the permission groups for the `SERVERS_AND_APPS` feature which is required for provisioning an Azure cloud
  cluster.
* Adds a new resource `polaris_azure_cloud_cluster`. The `polaris_azure_cloud_cluster` resource creates an Azure cloud
  cluster using RSC. [[docs](../resources/azure_cloud_cluster.md)]

## v1.2.1
* Update Go version to 1.24.8.

## v1.2.0
* Add support for the `EXOCOMPUTE_EKS_LAMBDA` role artifact to the `polaris_aws_cnp_account_trust_policy` resource.
* Add support for importing existing resources into Terraform. All resources that support the import workflow has a
  section in the documentation on how to import the resource.
* Change the `id` field of the `polaris_aws_cnp_account_trust_policy` resource to be a combination of the role key and
  the RSC cloud account ID. Note, this is a breaking change if a Terraform configuration expects the `id` field to be
  just the RSC cloud account ID.
* Deprecate the `features` field in the `polaris_aws_cnp_account_trust_policy` resource. The field has no replacement
  and is no longer used by the provider.
* Add `trust_policies` computed field to the `polaris_aws_cnp_account` resource. Whenever the `features` field of the
  `polaris_aws_cnp_account` resource changes, the `trust_policies` field will be updated with the new trust policies.
  The `trust_policies` field can be used instead of the `polaris_aws_cnp_account_trust_policy` resource.
  [[docs](../resources/aws_cnp_account.md#trust_policies)]
* Add support for Data Scanning Cyber Assisted Recovery feature to the `polaris_aws_account` resource.
* Add support for onboarding AWS DynamoDB using the `polaris_aws_cnp_account` resource.
* Add support for managing tag rules for AWS DynamoDB using the `polaris_tag_rule` resource.
* Improve logging by reducing the number of feature flags fetched.

## v1.1.7
* Add support for adding AWS Cloud Cluster with Elastic Storage. [[docs](../resources/aws_cloud_cluster.md)]
* Add support for onboarding the RSC feature `SERVERS_AND_APPS` using the `polaris_cnp_aws_account` resource.
  [[docs](../resources/aws_cnp_account.md)]

## v1.1.6
* Fix a bug in the `polaris_azure_subscription` resource where the wrong mutation was used to update the subscription
  when the subscription was updated to use permission groups and a resource group at the same time.

## v1.1.5
* Add support for creating DSPM, Data Scanning and Outpost features under `polaris_aws_account`.
  [[docs](../resources/aws_account.md)]

## v1.1.4
* Fix a bug where AWS Gov accounts would be erroneously onboarded as standard AWS accounts.

## v1.1.3
* CCES NTP servers can now be specified using a FQDN. Previously they were required to be IP addresses, now both IP
  addresses and FQDN are allowed.

## v1.1.2
* Update documentation for the `polaris_aws_cnp_account` and `polaris_azure_subscription` resources.
* Update documentation for the `polaris_aws_cnp_artifacts`, `polaris_aws_cnp_permissions`, and
  `polaris_azure_permissions` data sources.
* Add missing documentation to the `polaris_sso_group` and `polaris_user` data sources.

## v1.1.1
* Add optional `cluster_node_ip_address` field to CDM resources. The field can be used to specify the IP address of the
  cluster node to connect to. [[docs](../resources/cdm_bootstrap.md)] [[docs](../resources/cdm_bootstrap_cces_aws.md)]
  [[docs](../resources/cdm_bootstrap_cces_azure.md)]

## v1.1.0
* Add `resource_group_name`, `resource_group_region` and `resource_group_tags` fields to the
  `polaris_azure_subscription` resource. These fields can only be used if the `CNP_AZURE_SQL_DB_COPY_BACKUP` feature
  flag has been enabled for the RSC account.
* Fix a bug in the `polaris_aws_account` data source where the cloud account ID was not properly converted to a string
  causing the data source to error out.
* Require user email addresses of the `polaris_user` resources to be all lower case. RSC automatically converts all
  letters to lower case before storing the email addresses.
* Add support for AWS custom tags. [[docs](../resources/aws_custom_tags.md)]
* Add support for Azure custom tags. [[docs](../resources/azure_custom_tags.md)]
* The behavior of the `sdk_auth` field of the `polaris_azure_service_principal` resource has changed. The Azure app name
  is no longer looked up using the Azure AD Graph API. Instead, the app name is generated in a consistent way using the
  Azure app and tenant IDs. This change is made because of the deprecation of the Azure AD Graph API by Microsoft.
* Deprecate the `role_id` field in the `polaris_role_assignment` resource. Use the `role_ids` field instead.
* Deprecate the `user_email` field in the `polaris_role_assignment` resource. Use the `user_id` field with the
  `polaris_user` data source instead.
* Add `sso_group_id` field to the `polaris_role_assignment` resource. The `sso_group_id` field can be used to assign RSC
  roles to an SSO group.
* The `id` field of the `polaris_role_assignment` resource has changed from being the hash of the user email address and
  the role ID to being the user ID or SSO group ID.
* The `id` field of the `polaris_user` resource has changed from being the email address to being the user ID. Note,
  this is a breaking change if a Terraform configuration expects the `id` field be an email address.
* Add `polaris_sso_group` data source. The `polaris_sso_group` data source is used to look up SSO groups in RSC.
  [[docs](../data-sources/sso_group.md)]
* Add `polaris_user` data source. The `polaris_user` data source is used to look up users in RSC.
  [[docs](../data-sources/user.md)]
* Add `polaris_tag_rule` data source and resource. The `polaris_tag_rule` resource is used to create and manage RSC tag
  rules. [[docs](../data-sources/tag_rule.md)]  [[docs](../resources/tag_rule.md)]
* Add `polaris_sla_domain_assignment` resource. The `polaris_sla_domain_assignment` resource is used to assign an SLA
  domain to a workload. [[docs](../resources/sla_domain_assignment.md)]
* Add support for updating the `app_name` and `app_secret` fields of the `polaris_azure_service_principal` resource
  without recreating the resource.
* Add `feature` field to the `polaris_aws_account` data source.
* Add support for looking up an AWS account in RSC using the `polaris_aws_account` data source by the RSC cloud account
  ID.
* Improve CDM resource backwards compatibility. Align the CDM resource state of the RSC provider with the state of the
  older Rubrik (CDM) provider. This simplifies the state migration of Terraform modules switching to the RSC provider.
* Only add AWS subnets with names to the set of subnets. When using AWS Bring Your Own Kubernetes (BYOK) no subnets are
  specified. In this case RSC will return an empty string in the API response.
* Replace `APPROVED` with `ACCEPTED` in the Private Container Registry (PCR) documentation.
* The `polaris_cdm_bootstrap`, `polaris_cdm_bootstrap_cces_aws` and `polaris_cdm_bootstrap_cces_azure` resources now
  captures any status information returned in response to a bootstrap request failing.
* Fix a bug in the `polaris_azure_exocompute` resource where an AWS GraphQL endpoint was incorrectly called when mapping
  an Azure cloud account.
* Add support for registering clusters with RSC using the `polaris_cdm_registration` resource.
  [[docs](../resources/cdm_registration.md)]

## v1.0.0
* Fix a bug in the `polaris_aws_exocompute` resource. The `subnets` field was erroneously populated with subnets even
  when the Exocompute configuration did not contain any subnets.
* Fix a regression in the `polaris_azure_archival_location` data source. An extra level of structure in the RSC response
  caused reading the data source to fail.
* Fix a type conversion error in the `polaris_aws_exocompute` resource. During a prior refactoring, a new type was
  introduced for AWS regions to handle cases where the same region has multiple representations in the GraphQL API.
  This type was not properly converted on all code paths.
* Fix a bug in the `polaris_aws_cnp_permissions` data source where the data source's ID was accidentally calculated for
  the complete set of role keys and not just the specified role key.
* Add the `permissions` field to the `polaris_aws_cnp_account_attachments` resource. The `permissions` field should be
  used with the `id` field of the `polaris_aws_cnp_permissions` data source to trigger an update of the resource
  whenever the permissions changes. This update will move the RSC cloud account from the missing permissions state.
* Add support for Azure Bring Your Own Kubernetes Exocompute, also known as BYOK and customer managed Exocompute.
  [[docs](../resources/azure_exocompute_cluster_attachment.md)]
  [[docs](../resources/azure_private_container_registry.md)]
* Add support for the Cloud Native Blob Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#nested-schema-for-cloud_native_blob_protection)]
* Fix a regression in the cloud native archival location resources. An extra level of structure in the RSC response
  caused resource refreshes to fail.
* Add support for creating data center AWS accounts. [[docs](../resources/data_center_aws_account.md)]
* Add support for creating data center Azure subscriptions. [[docs](../resources/data_center_azure_subscription.md)]
* Add support for creating Amazon S3 data center archival locations.
  [[docs](../resources/data_center_archival_location_amazon_s3.md)]
* Add `polaris_data_center_aws_account` data source. [[docs](../data-sources/data_center_aws_account.md)]
* Add `polaris_data_center_azure_subscription` data source. [[docs](../data-sources/data_center_azure_subscription.md)]
* Add the field `manifest` to the `polaris_aws_exocompute_cluster_attachment` resource. The `manifest` field contains
  a Kubernetes manifest that can be passed to the Kubernetes Terraform provider or `kubectl` to establish a connection
  between the AWS EKS cluster and RSC. [[docs](../resources/aws_exocompute_cluster_attachment.md)]
* Deprecate the `setup_yaml` field in the `polaris_aws_exocompute_cluster_attachment` resource. Use the `manifest` field
  instead.
* The authentication token cache can now be controlled by the `polaris` provider configuration.
* The `credentials` field of the `polaris` provider configuration now accepts, in addition to what it already accepts,
  the content of an RSC service account credentials file.

## v0.9.0
* Update the `polaris_aws_archival_location` resource to support updates of the `bucket_tags` field without recreating
  the resources.
* Add `polaris_aws_account` data source. [[docs](../data-sources/aws_account.md)]
* Add `polaris_azure_subscription` data source. [[docs](../data-sources/azure_subscription.md)]
* Deprecate the `archival_location_id` field in the `polaris_aws_archival_location` data source. Use the `id` field
  instead.
* Deprecate the `archival_location_id` field in the `polaris_azure_archival_location` data source. Use the `id` field
  instead.
* Add the field `setup_yaml` to the `polaris_aws_exocompute_cluster_attachment` resource. The `setup_yaml` field
  contains K8s specs that can be passed to `kubectl` to establish a connection between the AWS EKS cluster and RSC.
  [[docs](../resources/aws_exocompute_cluster_attachment.md)]
* Fix a bug in the AWS feature removal code that causes removal of the `CLOUD_NATIVE_S3_PROTECTION` feature to fail.
* Improve the code that waits for RSC features to be disabled. The code now checks both the status of the job and the
  status of the cloud account.
* Improve the documentation for AWS data sources and resources.
* Update guides.
* Add `polaris_azure_archival_location` data source. [[docs](../data-sources/azure_archival_location.md)]
* Fix a bug in the `polaris_azure_archival_location` resource where the cloud account UUID would be passed to the RSC
  API instead of the Azure subscription UUID when creating an Azure archival location.
* Fix a bug in the `polaris_aws_cnp_account` resource where destroying it would constantly result in an *objects not
  authorized* error.
* Increase the wait time for asynchronous RSC operations to 8.5 minutes.
* Fix an issue with the permissions of subscriptions onboarded using the `polaris_azure_subscription` resource where
  the RSC UI would show the status as "Update permissions" even though the app registration would have all the required
  permissions.
* Move changelog and upgrade guides to guides folder.
* Add support for creating Azure cloud native archival locations. [[docs](../resources/azure_archival_location.md)]
* Fix a bug in the `polaris_aws_exocompute` resource where customer supplied security groups were not validated
  correctly.
* Add support for shared Exocompute to the `polaris_azure_exocompute` resource.
  [[docs](../resources/azure_exocompute.md#host_cloud_account_id)]
* Add the `polaris_account` data source. [[docs](../data-sources/account.md)]
* Add support for the Cloud Native Archival feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#nested-schema-for-cloud_native_archival)]
* Add support for the Cloud Native Archival Encryption feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#nested-schema-for-cloud_native_archival_encryption)]
* Add support for the Azure SQL Database Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#nested-schema-for-sql_db_protection)]
* Add support for the Azure SQL Managed Instance Protection feature to the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#nested-schema-for-sql_mi_protection)]
* Add support for specifying an Azure resource group when onboarding the Cloud Native Archival, Cloud Native Archival
  Encryption, Cloud Native Protection or Exocompute features using the `polaris_azure_subscription` resource.
  [[docs](../resources/azure_subscription.md#optional)]
