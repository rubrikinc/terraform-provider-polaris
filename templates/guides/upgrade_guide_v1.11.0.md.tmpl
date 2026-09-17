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

## Significant Changes

### `regional_config` now manages the host project and the secondary range

If an Exocompute configuration has been modified outside of Terraform, e.g. using the GraphQL API, to use a Shared VPC
or a named secondary range, declare the `host_project_id` and `secondary_range_name` fields in `regional_config`. Both
fields are now read from RSC, so the first plan after upgrading shows a diff until the Terraform configuration matches
what the Exocompute configuration already uses.
