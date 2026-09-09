---
page_title: "Upgrade Guide: v1.10.0"
---

# Upgrade Guide v1.10.0

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.10.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.10.0"
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
Otherwise, refresh the state to the v1.10.0 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.10.0 version.

## New Features

### Azure Database for PostgreSQL Flexible Server Protection

The new `postgres_flexible_server_protection` feature in the `polaris_azure_subscription` resource enables backup and
recovery of Azure Database for PostgreSQL flexible servers. The permissions RSC requires are returned by the
`polaris_azure_permissions` data source for the new `AZURE_POSTGRES_FLEXIBLE_SERVER_PROTECTION` feature, which has the
`BASIC` and `RECOVERY` permission groups.

Unlike the other subscription features, RSC requires both an Azure resource group and a user-assigned managed identity
for this feature, so the resource group and identity fields are mandatory rather than optional. RSC assigns the
identity to the temporary and recovery flexible servers it creates, giving them an identity to access Key Vault. The
identity must be in the feature's own resource group, so a configuration where
`user_assigned_managed_identity_resource_group_name` does not match `resource_group_name` is rejected during plan.
Create the identity out of band, for example with the `azurerm_user_assigned_identity` resource.

```terraform
data "polaris_azure_permissions" "postgres_flexible_server_protection" {
  feature = "AZURE_POSTGRES_FLEXIBLE_SERVER_PROTECTION"
  permission_groups = [
    "BASIC",
    "RECOVERY",
  ]
}

resource "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  postgres_flexible_server_protection {
    permissions           = data.polaris_azure_permissions.postgres_flexible_server_protection.id
    permission_groups     = data.polaris_azure_permissions.postgres_flexible_server_protection.permission_groups
    resource_group_name   = "my-postgres-rg"
    resource_group_region = "eastus2"

    user_assigned_managed_identity_name                = "my-postgres-identity"
    user_assigned_managed_identity_principal_id        = "9c8b7a60-c76c-11eb-8f2a-0f1e2d3c4b5a"
    user_assigned_managed_identity_region              = "eastus2"
    user_assigned_managed_identity_resource_group_name = "my-postgres-rg"

    regions = [
      "eastus2",
    ]
  }
}
```

Changing any of the resource group or identity fields forces the RSC feature to be re-onboarded.

~> **Note:** RSC does not return the region or the resource group name of a user-assigned managed identity, so
`user_assigned_managed_identity_region` and `user_assigned_managed_identity_resource_group_name` are not read back from
RSC. After importing a subscription, set both to the values the feature was onboarded with.

To protect the servers, use the new `AZURE_POSTGRES_FLEXIBLE_SERVER_OBJECT_TYPE` object type in the
`polaris_sla_domain` resource. The object type cannot be combined with any other object type, and the SLA Domain
specifies its location with a `backup_location` block rather than an `archival` block. The optional
`azure_postgres_flexible_server_config` block sets the point-in-time restore retention, between 7 and 35 days, that RSC
enforces on the source flexible server; omit the block to leave the server's existing Azure-side retention untouched.

```terraform
resource "polaris_sla_domain" "postgres_flexible_server" {
  name         = "postgres-flexible-server"
  object_types = ["AZURE_POSTGRES_FLEXIBLE_SERVER_OBJECT_TYPE"]

  daily_schedule {
    frequency = 1
    retention = 30
  }

  azure_postgres_flexible_server_config {
    backup_retention_in_days = 14
  }

  backup_location {
    archival_group_id = data.polaris_azure_archival_location.example.id
  }
}
```

Assign the SLA Domain with the `polaris_sla_domain_assignment` resource, resolving a server to its RSC ID with the new
`AzurePostgresFlexibleServer` object type in the `polaris_object` data source. Set `subscription_id` only to
disambiguate a server name shared across subscriptions.

```terraform
data "polaris_object" "server" {
  object_type = "AzurePostgresFlexibleServer"
  name        = "my-flexible-server"
}

resource "polaris_sla_domain_assignment" "server" {
  sla_domain_id = polaris_sla_domain.postgres_flexible_server.id
  object_ids    = [data.polaris_object.server.id]
}
```

### Google Cloud SQL Protection

The new `CLOUD_SQL_PROTECTION` feature in the `polaris_gcp_project` resource enables backup and in-place restore of
Google Cloud SQL instances. It has the `BASIC` and `EXPORT_AND_RESTORE` permission groups, and the permissions it
requires are returned by the `polaris_gcp_permissions` data source for the same new feature.

RSC runs Cloud SQL archival and archived recovery on Exocompute, which additionally requires the new `CLOUDSQL`
permission group on the `EXOCOMPUTE` feature. The permission group grants the Private Service Access networking
permissions and the permissions for the temporary Cloud SQL instances RSC creates. When Exocompute uses a VPC network
in a shared VPC host project, add the `CLOUDSQL` permission group to the `GCP_SHARED_VPC_HOST` feature of the host
project as well.

```terraform
data "polaris_gcp_permissions" "cloud_sql_protection" {
  feature = "CLOUD_SQL_PROTECTION"
  permission_groups = [
    "BASIC",
    "EXPORT_AND_RESTORE",
  ]
}

data "polaris_gcp_permissions" "exocompute" {
  feature = "EXOCOMPUTE"
  permission_groups = [
    "BASIC",
    "CLOUDSQL",
  ]
}

resource "polaris_gcp_project" "project" {
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012

  feature {
    name              = "CLOUD_SQL_PROTECTION"
    permission_groups = data.polaris_gcp_permissions.cloud_sql_protection.permission_groups
    permissions       = data.polaris_gcp_permissions.cloud_sql_protection.id
  }

  feature {
    name              = "EXOCOMPUTE"
    permission_groups = data.polaris_gcp_permissions.exocompute.permission_groups
    permissions       = data.polaris_gcp_permissions.exocompute.id
  }
}
```

~> **Note:** Cloud SQL protection must be enabled for the RSC account, otherwise RSC rejects the `CLOUDSQL` permission
group.

Instances are protected with the existing `GCP_CLOUD_SQL_OBJECT_TYPE` object type in the `polaris_sla_domain` resource.

### Export and Recovery Permission Groups for AWS S3 Protection

The `CLOUD_NATIVE_S3_PROTECTION` feature gains two permission groups. `RECOVERY` grants the AWS permissions required to
write objects back into an existing bucket, and `EXPORT` the permissions required to export an S3 recovery to a newly
created target bucket. Both require S3 recovery to be enabled for the RSC account.

The permission groups are supported in the `polaris_aws_account` and `polaris_aws_cnp_account` resources, and in the
`polaris_aws_cnp_artifacts` and `polaris_aws_cnp_permissions` data sources.

```terraform
resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"

  feature {
    name = "CLOUD_NATIVE_S3_PROTECTION"
    permission_groups = [
      "BASIC",
      "EXPORT",
      "RECOVERY",
    ]
  }

  regions = [
    "us-east-2",
  ]
}
```

~> **Note:** Use the `polaris_aws_permission_groups` data source to read the permission groups currently available for
a feature.

### New `polaris_object` Object Types

Besides `AzurePostgresFlexibleServer`, covered above, the `polaris_object` data source supports two new object types,
each resolving an object to its RSC ID by name for use with the `polaris_sla_domain_assignment` resource:

* `AzureSqlManagedInstanceServer` — an Azure SQL Managed Instance server. Set `subscription_id` to disambiguate a
  server name shared across subscriptions.
* `CloudNativeTagRule` — a cloud native tag rule.

## Significant Changes

### The `timeouts` block in `polaris_object` is now a nested attribute

The optional `timeouts` block in the `polaris_object` data source is now a nested attribute rather than a block. If you
set a custom read timeout, change the block syntax to an attribute assignment. This is a result of migrating the data
source to the Terraform Plugin Framework; lookups themselves behave the same.
```terraform
# Before
data "polaris_object" "account" {
  name        = "my-account"
  object_type = "AwsNativeAccount"

  timeouts {
    read = "10m"
  }
}

# After
data "polaris_object" "account" {
  name        = "my-account"
  object_type = "AwsNativeAccount"

  timeouts = {
    read = "10m"
  }
}
```
Configurations that do not set a `timeouts` block are unaffected.

### `polaris_object` validate optional attributes at plan time

In the `polaris_object` data source, the `subscription_id`, `org_id` and `project_id` fields each apply only to specific
object types:

* `subscription_id` — `AzureNativeResourceGroup`, `AzurePostgresFlexibleServer`, `AzureSqlManagedInstanceServer`
* `org_id` — `AzureDevOpsProject`, `AzureDevOpsRepository`, `GitHubRepository`
* `project_id` — `AzureDevOpsRepository`

Previously, setting one of these fields for any other `object_type` was silently ignored. The data source now validates
this at plan time and returns an error identifying the offending field. If your configuration set one of these fields
for an `object_type` it does not apply to, remove it; the field had no effect before, so removing it does not change the
resolved object.

In addition, `subscription_id` is no longer required when `object_type` is `AzureNativeResourceGroup`. A resource group
is now looked up by name alone; set `subscription_id` only to disambiguate a resource group name that is shared across
subscriptions. Existing configurations that set `subscription_id` continue to work unchanged.

### The `polaris_sla_domain` resource rejects `backup_location` for unsupported object types

The `backup_location` block in the `polaris_sla_domain` resource applies only to the object types which have a backup
location:

* `AWS_S3_OBJECT_TYPE`
* `AZURE_POSTGRES_FLEXIBLE_SERVER_OBJECT_TYPE`
* `AZURE_SQL_DATABASE_OBJECT_TYPE` and `AZURE_SQL_MANAGED_INSTANCE_OBJECT_TYPE`, on accounts where the
  `CNP_AZURE_SQL_SLA_REVAMP` feature is enabled

Previously the block was routed on the account's AWS S3 multiple backup locations feature rather than on the object
types, so it was passed on for every object type — as an AWS S3 configuration when that feature was not enabled, and
as an SLA-level backup location when it was — and RSC ignored it for the object types which have no backup location.
Setting it for any other object type is now an error, on both create and update:
```shell
backup_location is not supported by the configured object types
```

The error is raised during apply rather than plan, so a configuration carrying a stray `backup_location` block still
plans clean.

If your configuration sets `backup_location` for an object type which is not listed above, remove the block. It had no
effect before, so removing it does not change what the SLA Domain protects or where its snapshots are stored:
```terraform
# Before
resource "polaris_sla_domain" "vsphere" {
  name         = "vsphere"
  object_types = ["VSPHERE_OBJECT_TYPE"]

  daily_schedule {
    frequency = 1
    retention = 30
  }

  backup_location {
    archival_group_id = data.polaris_aws_archival_location.example.id
  }
}

# After
resource "polaris_sla_domain" "vsphere" {
  name         = "vsphere"
  object_types = ["VSPHERE_OBJECT_TYPE"]

  daily_schedule {
    frequency = 1
    retention = 30
  }
}
```
To archive the snapshots of an object type which has no backup location, use the `archival` block and its
`archival_location_id` field instead.

~> **Note:** Azure SQL Database and Managed Instance SLA Domains on accounts where the `CNP_AZURE_SQL_SLA_REVAMP`
feature is not enabled keep the legacy model and are affected as well. There, an Azure SQL Database SLA Domain requires
an `archival` block with instant archival enabled, and an Azure SQL Managed Instance SLA Domain supports no archival at
all. See the [v1.9.0 upgrade guide](upgrade_guide_v1.9.0.md) for the V1/V2 model and how the feature changes it.

### Security group fields in the AWS Exocompute resource are deprecated

The `cluster_security_group_id` and `node_security_group_id` fields in the `polaris_aws_exocompute` resource are
deprecated. RSC now always creates and manages the security groups for RSC managed Exocompute configurations, and a
future RSC release will reject configurations that supply them.

RSC scopes its security group permissions on the name and tags of the security group it creates, notably the
`rk_managed` tag. It cannot apply that tag to a security group you created without holding `CreateTags` on every
security group in the account, so customer-supplied groups can fail with an authorization error during some
operations.

Setting either field still works in this release and produces a deprecation warning. To resolve the warning, remove
both fields and let RSC create the security groups:
```terraform
# Before
resource "polaris_aws_exocompute" "host" {
  account_id                = data.polaris_aws_account.host.id
  cluster_security_group_id = "sg-005656347687b8170"
  node_security_group_id    = "sg-00e147656785d7e2f"
  region                    = "us-east-2"
  vpc_id                    = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# After
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}
```
Run `terraform plan` before applying the change and read the plan. Both fields are marked `ForceNew`, so if the plan
does show a change to either of them it replaces the Exocompute configuration, which tears down and redeploys the
Exocompute cluster. Treat a replacement in the plan as a maintenance operation rather than applying it straight away.

Leaving the fields in place is the riskier option over time. Once RSC manages the security groups for a configuration,
a configuration that still supplies security group IDs differs from what RSC reports for it, and because both fields
force a new resource that difference is planned as a replacement of the Exocompute configuration.

Customer managed Exocompute — where you attach your own EKS cluster with the
`polaris_aws_exocompute_cluster_attachment` resource — never used these fields and is unaffected.

### `CLOUD_COST_REPORT` is no longer tracked on AWS IAM roles accounts

RSC enables the `CLOUD_COST_REPORT` feature on its own for any AWS account carrying a workload feature which accrues
AWS spend, whatever feature set was passed when onboarding. It cannot be declared in the `feature` block of the
`polaris_aws_cnp_account` resource, so tracking it produced a persistent diff removing it. Applying that diff silently
disabled cost reporting for the account, and RSC added the feature back the next time the features were onboarded.

Only the features which can be declared are tracked now. Expect a one-time change on upgrade for accounts where RSC
enabled cost reporting: the feature drops out of state on the first refresh, and the diff removing it goes with it.
Nothing needs to change in the configuration, and cost reporting in RSC is left alone.

The same filtering applies to the `features` field of the `polaris_aws_cnp_account_attachments` resource, and to
imports of both resources.

Destroying a `polaris_aws_cnp_account` now removes cost reporting explicitly. RSC removes an account once its last
feature is removed, but it never removes `CLOUD_COST_REPORT` along with the features it was enabled for, so removing
only the declared features could leave the feature — and with it the account — behind in RSC.
