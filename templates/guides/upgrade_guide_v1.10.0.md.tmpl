---
page_title: "Upgrade Guide: v1.10.0"
---

# Upgrade Guide v1.10.0

The v1.10.0 release migrates the `polaris_object` data source to the Terraform Plugin Framework, adds support for the
`CloudNativeTagRule` object type, validates the `subscription_id`, `org_id` and `project_id` fields at plan time, and
deprecates the security group fields in the `polaris_aws_exocompute` resource.
See the [changelog](changelog.md) for the full list of changes.

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

* `subscription_id` — `AzureNativeResourceGroup`
* `org_id` — `AzureDevOpsProject`, `AzureDevOpsRepository`, `GitHubRepository`
* `project_id` — `AzureDevOpsRepository`

Previously, setting one of these fields for any other `object_type` was silently ignored. The data source now validates
this at plan time and returns an error identifying the offending field. If your configuration set one of these fields
for an `object_type` it does not apply to, remove it; the field had no effect before, so removing it does not change the
resolved object.

In addition, `subscription_id` is no longer required when `object_type` is `AzureNativeResourceGroup`. A resource group
is now looked up by name alone; set `subscription_id` only to disambiguate a resource group name that is shared across
subscriptions. Existing configurations that set `subscription_id` continue to work unchanged.

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
Both fields are optional and computed, so removing them from the configuration does not produce a diff and does not
replace the Exocompute configuration. The security groups already recorded in state stay there and the running cluster
continues to use them, so removing the fields is safe and only silences the warning.

If you want RSC to take ownership of the security groups for an existing configuration, the Exocompute configuration
has to be recreated, for example with `terraform apply -replace=polaris_aws_exocompute.host`. This tears down and
redeploys the Exocompute cluster, so plan it as a maintenance operation.

Customer managed Exocompute — where you attach your own EKS cluster with the
`polaris_aws_exocompute_cluster_attachment` resource — never used these fields and is unaffected.
