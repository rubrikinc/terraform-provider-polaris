---
page_title: "Upgrade Guide: v1.11.0"
---

# Upgrade Guide v1.11.0

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.11.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.11.0"
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
Otherwise, refresh the state to the v1.11.0 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.11.0 version.

## New Features

### GCP Exocompute on a Shared VPC

The `regional_config` block of the `polaris_gcp_exocompute` resource gains two optional fields, `host_project_id` and
`secondary_range_name`.

`host_project_id` is the GCP project ID of the project owning the VPC network. It is only needed when the network is a
Shared VPC, in which case the network and the subnet belong to a host project rather than to the project running
Exocompute. Without it, RSC looks for the subnet in the Exocompute project and the GKE cluster setup fails. Note that
this is the GCP project ID of the host project, not the RSC cloud account ID.

`secondary_range_name` is the name of the GKE pods secondary IP range on the subnet. It defaults to `pods-cidr-range`,
which is the name RSC uses when the regional configuration does not specify one.

```terraform
data "polaris_gcp_project" "shared_vpc_host" {
  name = "my-shared-vpc-host-project"
}

resource "polaris_gcp_exocompute" "exocompute" {
  cloud_account_id = polaris_gcp_project.project.id

  regional_config {
    region               = "us-west1"
    subnet_name          = "my-shared-vpc-subnet-01"
    vpc_name             = "my-shared-vpc-01"
    host_project_id      = data.polaris_gcp_project.shared_vpc_host.project_id
    secondary_range_name = "my-secondary-range"
  }
}
```

~> **Note:** Running Exocompute on a Shared VPC requires the host project to be onboarded with the
`GCP_SHARED_VPC_HOST` feature.

### Google BigQuery Protection

RSC can now protect Google BigQuery datasets. Two new features in the `polaris_gcp_project` resource,
`GCP_BIGQUERY_PROTECTION` and `GCP_BIGQUERY_RESERVATION`, onboard a GCP project for it, and the new
`GCP_BIGQUERY_OBJECT_TYPE` object type in the `polaris_sla_domain` resource protects the datasets.

`GCP_BIGQUERY_PROTECTION` enables backup and restore of the BigQuery datasets in the project. It has the `BASIC` and
`EXPORT_AND_RESTORE` permission groups. `GCP_BIGQUERY_RESERVATION` designates the project where RSC creates the BigQuery
slot reservation it runs BigQuery backup and recovery jobs on. It has the `BASIC` permission group. The permissions
required by the two features are returned by the `polaris_gcp_permissions` data source for the same new features.

Only one project per RSC account can have the `GCP_BIGQUERY_RESERVATION` feature, so to move it to another project,
remove it from the current project before adding it to the new one. If the reservation project also contains datasets to
protect, add both features to the same `polaris_gcp_project` resource.

```terraform
resource "polaris_gcp_project" "bigquery" {
  project        = "my-bigquery-project"
  project_name   = "My BigQuery Project"
  project_number = 123456789012

  feature {
    name              = "GCP_BIGQUERY_PROTECTION"
    permission_groups = ["BASIC", "EXPORT_AND_RESTORE"]
  }
}

resource "polaris_gcp_project" "bigquery_reservation" {
  project        = "my-reservation-project"
  project_name   = "My Reservation Project"
  project_number = 210987654321

  feature {
    name              = "GCP_BIGQUERY_RESERVATION"
    permission_groups = ["BASIC"]
  }
}
```

~> **Note:** Both BigQuery features require BigQuery protection to be enabled for the RSC account.

The datasets are protected by an SLA Domain with the `GCP_BIGQUERY_OBJECT_TYPE` object type. BigQuery backs up directly
to its backup locations, so the SLA Domain requires a `backup_location` and does not use the `archival` block.

```terraform
data "polaris_gcp_archival_location" "archival_location" {
  name = "my-archival-location"
}

resource "polaris_sla_domain" "bigquery" {
  name         = "gcp-bigquery"
  description  = "GCP BigQuery SLA"
  object_types = ["GCP_BIGQUERY_OBJECT_TYPE"]

  daily_schedule {
    frequency      = 1
    retention      = 30
    retention_unit = "DAYS"
  }

  backup_location {
    archival_group_id = data.polaris_gcp_archival_location.archival_location.id
  }
}
```

The provider checks the following rules for an SLA Domain with this object type during `terraform plan`, so a violation
fails the plan instead of failing part way through an apply:

* The object type cannot be combined with other object types.
* At least one `backup_location` is required, and the `archival` block is not supported.
* Replication is not supported.
* The `minute_schedule` block is not supported.
* The most frequent schedule must take a snapshot at least every 7 days, which means an `hourly_schedule` with a
  `frequency` of at most 168, a `daily_schedule` with a `frequency` of at most 7, or a `weekly_schedule` with a
  `frequency` of 1. A `monthly_schedule`, `quarterly_schedule` or `yearly_schedule` is always further apart than that,
  so it cannot be the only schedule, but it can be combined with one of the others.

## Significant Changes

### `regional_config` now manages the host project and the secondary range

If an Exocompute configuration has been modified outside of Terraform, e.g. using the GraphQL API, to use a Shared VPC
or a named secondary range, declare the `host_project_id` and `secondary_range_name` fields in `regional_config`. Both
fields are now read from RSC, so the first plan after upgrading shows a diff until the Terraform configuration matches
what the Exocompute configuration already uses.

### The permission group data sources only return supported permission groups

The `polaris_aws_permission_groups` and `polaris_azure_permission_groups` data sources now only return the permission
groups which the account resources accept for the requested feature. Previously they returned every permission group
RSC offered for the feature, including groups the provider does not support yet. Passing those groups to the
`polaris_aws_account`, `polaris_aws_cnp_account` or `polaris_azure_subscription` resources failed validation during
plan.

When this was written, RSC offered the following permission groups which are no longer returned:

| Cloud | Feature                        | Permission groups no longer returned                                     |
|-------|--------------------------------|--------------------------------------------------------------------------|
| AWS   | `EXOCOMPUTE`                   | `ADVANCED_DIAGNOSTICS`, `SURGICAL_RECOVERY`, `RECOVERY_RDS_CONNECTIVITY` |
| AWS   | `RDS_PROTECTION`               | `RECOVER_TO_S3`                                                          |
| AWS   | `OUTPOST`                      | `RSC_MANAGED_CLUSTER`                                                    |
| Azure | `CLOUD_NATIVE_BLOB_PROTECTION` | `INVENTORY_GENERATION`                                                   |
| Azure | `EXOCOMPUTE`                   | `ENCRYPTION`                                                             |

A feature which the account resources do not support at all, such as the AWS `CLOUD_NATIVE_CONFIG_PROTECTION` feature,
now returns an empty `permission_groups` set.

No change is needed for configurations which pass the result of the data sources to the account resources. A
configuration which uses the result in some other way, for example in an output or in a condition on
`permission_groups`, now sees the filtered set. The `id` attribute is a hash of the returned permission groups and
statements, so it changes for the features listed above.
