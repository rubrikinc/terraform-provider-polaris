---
page_title: "Upgrade Guide: v1.8.0"
---

# Upgrade Guide v1.8.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

## Prerequisites

This release requires **Terraform 1.11 or later**. The `admin_email` and `admin_password` fields on the
`polaris_aws_cloud_cluster` and `polaris_azure_cloud_cluster` resources now use write-only attributes, which are only
supported in Terraform 1.11+. These fields are only used during initial cluster creation and cannot be changed after
deployment, so they no longer need to be stored in state.

If you are running an older version of Terraform, you will see the following error when applying your configuration:

```
Error: Write-only Attribute Not Allowed

The resource contains a non-null value for write-only attribute
"admin_email" Write-only attributes are only supported in Terraform
1.11 and later.
```

Upgrade Terraform before upgrading the provider:
```shell
% brew upgrade terraform
```

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.8.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.8.0"
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
If you get an error or an unwanted diff, please see the _New Features_ section below for additional instructions.
Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.8.0 version.

## New Features

### Multi-AZ Resiliency for Cloud Clusters

The `polaris_aws_cloud_cluster` and `polaris_azure_cloud_cluster` resources now support deploying clusters across
multiple availability zones for AZ resiliency. This is controlled by two new fields:

- `is_az_resilient` (bool) - Set to `true` to enable Multi-AZ deployment.
- `subnet_az_config` (block list in `vm_config`) - Specifies a subnet for each availability zone. Required when
  `is_az_resilient` is `true`.

When `is_az_resilient` is enabled:
- `use_placement_groups` must be `false` (AWS only).
- At least 3 availability zones should be specified in `subnet_az_config`.
- For AWS, `subnet_id` in `vm_config` becomes optional.
- For Azure, `availability_zone` and `subnet` in `vm_config` are replaced by `subnet_az_config` entries.

#### AWS Example

```terraform
resource "polaris_aws_cloud_cluster" "multi_az" {
  cloud_account_id     = "12345678-1234-1234-1234-123456789012"
  region               = "us-west-2"
  is_az_resilient      = true
  use_placement_groups = false

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "SecurePassword123!"
    dns_name_servers        = ["8.8.8.8"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    bucket_name             = "my-s3-bucket"
    enable_immutability     = true
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version           = "9.4.0-p2-30507"
    instance_type         = "M6I_2XLARGE"
    instance_profile_name = "RubrikCloudClusterInstanceProfile"
    vpc_id                = "vpc-12345678"
    security_group_ids    = ["sg-12345678"]

    subnet_az_config {
      availability_zone = "us-west-2a"
      subnet            = "subnet-11111111"
    }

    subnet_az_config {
      availability_zone = "us-west-2b"
      subnet            = "subnet-22222222"
    }

    subnet_az_config {
      availability_zone = "us-west-2c"
      subnet            = "subnet-33333333"
    }
  }
}
```

#### Azure Example

```terraform
resource "polaris_azure_cloud_cluster" "multi_az" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  is_az_resilient  = true

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "SecurePassword123!"
    dns_name_servers        = ["8.8.8.8"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version                     = "9.2.3-p7-29713"
    instance_type                   = "STANDARD_D8S_V5"
    location                        = "westus"
    resource_group                  = "my-resource-group"
    network_resource_group          = "my-network-resource-group"
    vnet_resource_group             = "my-vnet-resource-group"
    vnet                            = "my-vnet"
    network_security_group          = "my-network-security-group"
    network_security_resource_group = "my-network-security-resource-group"
    vm_type                         = "EXTRA_DENSE"
    storage_account                 = "my-storage-account"
    container_name                  = "my-container"
    enable_immutability             = true
    user_assigned_managed_identity  = "my-managed-identity"

    subnet_az_config {
      availability_zone = "1"
      subnet            = "subnet-zone-1"
    }

    subnet_az_config {
      availability_zone = "2"
      subnet            = "subnet-zone-2"
    }

    subnet_az_config {
      availability_zone = "3"
      subnet            = "subnet-zone-3"
    }
  }
}
```

For more details, see the [polaris_aws_cloud_cluster documentation](../resources/aws_cloud_cluster.md) and the
[polaris_azure_cloud_cluster documentation](../resources/azure_cloud_cluster.md).

~> **Note:** Multi-AZ resiliency requires the `CCES_AZ_RESILIENCY_ENABLED` feature flag to be enabled on the RSC
account. You can verify this using the `polaris_feature_flag` data source:

```terraform
data "polaris_feature_flag" "az_resiliency" {
  name = "CCES_AZ_RESILIENCY_ENABLED"
}

output "az_resiliency_enabled" {
  value = data.polaris_feature_flag.az_resiliency.enabled
}
```

If the feature flag is not enabled, contact Rubrik support to enable it before using Multi-AZ resiliency.
