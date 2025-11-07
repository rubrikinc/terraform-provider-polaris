---
page_title: "Upgrade Guide: v1.3.0"
---

# Upgrade Guide v1.3.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note, deprecated resources and fields will be removed in a future release, please migrate your configurations
to use the recommended replacements as soon as possible.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.3.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.3.0"
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
If you get an error or an unwanted diff, please see the _Significant Changes and New Features_ below for additional
instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.3.0 version.

## New Features

### GCP Custom Labels
Support for custom labels has been added for GCP. The `polaris_gcp_custom_labels` resource is used to manage the custom
labels. Custom labels are applied to all resources created in GCP cloud accounts by RSC. Here's a simple example,
showing how to add two custom GCP labels:
```terraform
resource "polaris_gcp_custom_labels" "labels" {
  custom_tags = {
    "app"    = "RSC"
    "vendor" = "Rubrik"
  }
}
```

### GCP Archival Locations
Support for archival locations has been added for GCP. The `polaris_gcp_archival_location` resource is used to create an
archival location. Here's a simple example, showing how to create an archival location:
```terraform
data "polaris_gcp_project" "project" {
  name = "my-gcp-project"
}

resource "polaris_gcp_archival_location" "archival_location" {
  cloud_account_id = data.polaris_gcp_project.project.id
  name             = "my-archival-location"
  bucket_prefix    = "my-bucket-prefix"
}
```

### GCP Permission Groups
Support for permission groups has been added for GCP. Permission groups are used to manage granular permissions of an
RSC feature. The new GCP permission model with conditional permissions requires that permission groups are used. The
`polaris_gcp_project` resource and the `polaris_gcp_permissions` data source have been updated to support permission
groups. When permission groups are used for an RSC feature, it should be used for all RSC features of the GCP project.

The following example shows how to use permission groups to look up the required permissions for the Cloud Native
Protection RSC feature with the `polaris_gcp_permissions` resource:
```terraform
data "polaris_gcp_permissions" "cloud_native_protection" {
  feature = "CLOUD_NATIVE_PROTECTION"
  permission_groups = [
    "BASIC",
    "EXPORT_AND_RESTORE",
    "FILE_LEVEL_RECOVERY",
  ]
}
```

The following example shows how to use permission groups to onboard the Cloud Native Protection RSC feature with the
`polaris_gcp_project` resource:
```terraform
resource "polaris_gcp_project" "project" {
  credentials    = google_service_account_key.service_account.private_key
  project        = data.google_project.project.id
  project_name   = data.google_project.project.name
  project_number = data.google_project.project.number

  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
      "FILE_LEVEL_RECOVERY",
    ]
  }
}
```

As part of adding support for permission groups, the `cloud_native_protection` field of the `polaris_gcp_project`
resource and the `features` field of the `polaris_gcp_permissions` data source have been deprecated.

### GCP Conditional Permissions
The `polaris_gcp_perissions` data source has been extended with new fields to support an improved permissions model with
conditional permissions. Previously, the `polaris_gcp_permissions` data source was used to get a list of permissions for
a set of RSC features. The permissions were then used to create a custom role which was assigned to the RSC service
account. Now, the `polaris_gcp_permissions` data source is used to get two sets of permissions, one for permissions with
conditions and one for permissions without conditions, for a single RSC feature. The permissions are then used to create
two custom roles, one for the permissions with conditions and one for the permissions without conditions. To get the
required permissions for multiple RSC features, multiple instances of the `polaris_gcp_permissions` data source would be
created, each specifying a single RSC feature. The permissions without conditions, for all the data source instances,
can optionally be merged into a single custom role.

The following example shows how the `polaris_gcp_permissions` data source can be used to get the required permissions
for the Cloud Native Protection RSC feature and create a custom role for the permissions with conditions and a custom
role for the permissions without conditions:
```terraform
data "polaris_gcp_permissions" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"
  permission_groups = [
    "BASIC",
    "EXPORT_AND_RESTORE",
    "FILE_LEVEL_RECOVERY",
  ]
}

data "google_service_account" "account" {
  account_id = "rubrik-service-account"
}

resource "google_project_iam_custom_role" "cnp_with_conditions" {
  role_id     = "rubrik_cnp_with_conditions"
  title       = "Rubrik Cloud Native Protection with Conditions"
  permissions = data.polaris_gcp_permissions.cnp.with_conditions
  project     = var.project_id
}

resource "google_project_iam_custom_role" "cnp_without_conditions" {
  role_id     = "rubrik_cnp_without_conditions"
  title       = "Rubrik Cloud Native Protection without Conditions"
  permissions = data.polaris_gcp_permissions.cnp.without_conditions
  project     = var.project_id
}

resource "google_project_iam_member" "cnp_with_conditions" {
  member  = data.google_service_account.account.member
  project = var.project_id
  role    = google_project_iam_custom_role.cnp_with_conditions.id

  condition {
    title      = "Rubrik Condition CNP"
    expression = join(" || ", data.polaris_gcp_permissions.cnp.conditions)
  }
}

resource "google_project_iam_member" "cnp_without_conditions" {
  member  = data.google_service_account.account.member
  project = var.project_id
  role    = google_project_iam_custom_role.without_conditions.id
}
```

### GCP Shared VPC Host
Support for GCP Shared VPC Host has been added. The `polaris_gcp_project` resource is used to onboard a GCP project as a
Shared VPC Host. Here's a simple example, showing how to add a GCP project as a Shared VPC Host:
```terraform
data "polaris_gcp_permissions" "gcp_shared_vpc_host" {
  feature = "GCP_SHARED_VPC_HOST"
  permission_groups = [
    "BASIC",
  ]
}

resource "polaris_gcp_project" "project" {
  credentials    = google_service_account_key.service_account.private_key
  project        = data.google_project.project.id
  project_name   = data.google_project.project.name
  project_number = data.google_project.project.number

  feature {
    name              = data.polaris_gcp_permissions.gcp_shared_vpc_host.feature
    permission_groups = data.polaris_gcp_permissions.gcp_shared_vpc_host.permission_groups
    permissions       = data.polaris_gcp_permissions.gcp_shared_vpc_host.id
  }
}
```

## Significant Changes

### GCP Project
The `project`, `project_name` and `project_number` fields of the `polaris_gcp_project` resource are now required.
Previously they were optional, but due to changes in the permissions required by RSC, they are now required. Existing
Terraform configurations will need to be updated to include these fields. Not having these fields included in the
Terraform configuration will result in an error similar to:
```console
╷
│ Error: Missing required argument
│
│   on main.tf line 43, in resource "polaris_gcp_project" "project":
│   43: resource "polaris_gcp_project" "project" {
│
│ The argument "project_name" is required, but no definition was found.
╵
╷
│ Error: Missing required argument
│
│   on main.tf line 43, in resource "polaris_gcp_project" "project":
│   43: resource "polaris_gcp_project" "project" {
│
│ The argument "project_number" is required, but no definition was found.
╵
╷
│ Error: Missing required argument
│
│   on main.tf line 43, in resource "polaris_gcp_project" "project":
│   43: resource "polaris_gcp_project" "project" {
│
│ The argument "project" is required, but no definition was found.
╵
```
To resolve this, add the values for the fields to the `polaris_gcp_project` resource. The current, implicit values, of
the fields can be found in the Terraform state for the `polaris_gcp_project` resource. Use the `terraform state show`
command to print the state for the `polaris_gcp_project` resource. E.g:
```console
terraform state show polaris_gcp_project.<resource_name>
```
