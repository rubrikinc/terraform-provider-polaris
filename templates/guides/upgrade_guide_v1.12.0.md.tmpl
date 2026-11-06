---
page_title: "Upgrade Guide: v1.12.0"
---

# Upgrade Guide v1.12.0

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

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.12.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.12.0"
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
Otherwise, refresh the state to the v1.12.0 version:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.12.0 version.

## New Features

### Exocompute health checks

The `polaris_aws_exocompute` and `polaris_azure_exocompute` resources have a new `trigger_health_check` field. When it
is `true`, the provider starts an RSC health check after the Exocompute configuration is created, or when the field is
changed from `false` to `true`, and waits for the result. Changing the field doesn't replace the configuration, and
changing it to `false` has no effect.

A health check which doesn't report the configuration as healthy, doesn't finish in time, or which RSC refuses to start,
is reported as a warning, and the configuration is kept. The field is ignored for customer managed configurations, and
it conflicts with the host cloud account ID, since an application configuration has no cluster to check.

When RSC has to launch a cluster first, a health check can take around 30 minutes. The resources get a `timeouts` block,
with create and update timeouts defaulting to 60 minutes:
```terraform
resource "polaris_aws_exocompute" "exocompute" {
  account_id           = polaris_aws_account.account.id
  region               = "us-east-2"
  subnets              = ["subnet-0123456789abcdef0", "subnet-0fedcba9876543210"]
  vpc_id               = "vpc-0123456789abcdef0"
  trigger_health_check = true

  timeouts {
    create = "90m"
  }
}
```
See [Health Checks](../resources/aws_exocompute.md#health-checks) for more information.

### Azure Exocompute networking and disk encryption

The `optional_config` block of the `polaris_azure_exocompute` resource has five new fields:
* `postgres_subnet_id` and `postgres_private_dns_zone_id` configure the networking of Azure Postgres Flexible Server
  backups. They must be specified together.
* `sql_private_dns_zone_id` configures the private DNS zone used for Azure SQL.
* `disk_encryption_set_id` encrypts the cluster disks with a disk encryption set. It needs the new `ENCRYPTION`
  permission group for the `EXOCOMPUTE` feature in the `polaris_azure_subscription` resource, which lets RSC list the
  disk encryption sets in the subscription.
* `health_check_vm_name_prefix` sets the name prefix of the virtual machine RSC uses for health checks.

When one of the new fields is not set, the value stored in RSC is kept, so a value configured using the RSC UI does not
cause a diff. Changing one of the fields replaces the Exocompute configuration, like the other fields in the
`optional_config` block. See the [polaris_azure_exocompute docs](../resources/azure_exocompute.md) for an example.

## Significant Changes

### GCP Cloud Discovery

The `polaris_gcp_project` resource now supports the `CLOUD_DISCOVERY` feature. Cloud Discovery is the read-only parent
of the GCP protection features. When it is enabled for the RSC account, the RSC UI adds it to a project whenever a
protection feature is onboarded, without showing it as a separate choice. RSC refuses to remove it from a project while
a protection feature is still enabled on the project.

Previous versions of the provider read the feature from RSC into the state but rejected it in the configuration, so a
project with Cloud Discovery enabled always showed a plan removing it, and applying that plan failed in RSC. This
version accepts the feature in the configuration and stops a plan which removes it while a protection feature remains,
with an error like:
```
Error: feature CLOUD_DISCOVERY cannot be removed while the protection feature CLOUD_NATIVE_PROTECTION is enabled
```
If you get this error for a project you didn't change, Cloud Discovery is enabled on the project in RSC but not in the
configuration. Add it to the `polaris_gcp_project` resource:
```terraform
resource "polaris_gcp_project" "project" {
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012

  feature {
    name              = "CLOUD_DISCOVERY"
    permission_groups = ["BASIC"]
  }

  feature {
    name              = "CLOUD_NATIVE_PROTECTION"
    permission_groups = ["BASIC"]
  }
}
```
To remove Cloud Discovery, first remove the protection features from the project, then remove Cloud Discovery in a
later apply. When the project is destroyed, the provider removes Cloud Discovery last.

Projects without Cloud Discovery enabled are not affected.

### Azure cloud clusters which failed to refresh

Reading a `polaris_azure_cloud_cluster` resource failed with an internal error from `azureCcCdmVersions` when RSC
reported the cluster region in a different format than the native Azure region name, e.g. `GermanyWestCentral`
instead of `germanywestcentral`. This version matches the region case-insensitively, so the refresh succeeds.

If a cluster's refresh was failing before upgrading, the first successful refresh writes the CDM version reported by
RSC to the `cdm_version` field of the `vm_config` block. The field forces a new resource when changed, so if the CDM
version in the configuration differs from the version the cluster is running, e.g. because the cluster has been
upgraded, the plan proposes replacing the cluster.

~> **Note:** Review the plan before applying it. If it proposes replacing an Azure cloud cluster because of
   `cdm_version`, update the `cdm_version` field in the configuration to the version the cluster is running, then run
   `terraform plan` again.

### Azure Exocompute DNS zone warnings no longer block creation

Creating a `polaris_azure_exocompute` resource no longer fails when a private DNS zone is in a different subscription
than the Exocompute VNet, or when the subnet has blocking network security group rules. RSC only reports these as
warnings, and the RSC UI lets the user continue past them. All other validation failures, such as an incorrectly
delegated subnet or an invalid private DNS zone, still block creation.

### Additional permission policy for AWS cloud cost reporting

On RSC accounts where AWS cloud cost reporting is enabled, the `polaris_aws_cnp_permissions` data source can return one
more customer managed policy than before, for the `CLOUD_COST_REPORT` feature. RSC adds the feature to AWS accounts with
features that accrue AWS spend, and the provider now requests the permissions the same way the RSC UI does. If the
data source is used to create IAM roles, the plan shows the new policy being added to the role. Apply the plan so the
role has the permissions RSC expects.

### More features in the account data source

The `aws`, `azure` and `gcp` blocks of the `polaris_account` data source list the features supported by each cloud
vendor, together with their permission groups. They now also include the `CLOUD_NATIVE_UEM_KEY_MANAGEMENT` feature for
AWS, Azure and GCP, and the `CLOUD_DISCOVERY` feature for GCP, when RSC returns permission groups for them.
Configurations which iterate over the features in these blocks see the new entries. The `features` field, which lists
the features enabled for the RSC account, is not affected.
