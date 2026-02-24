---
page_title: "Upgrade Guide: v1.5.0"
---

# Upgrade Guide v1.5.0

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note that deprecated resources and fields will be removed in a future release. Please migrate your configurations
to use the recommended replacements as soon as possible.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.5.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.5.0"
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
If you get an error or an unwanted diff, please see the _Significant Changes_ and _New Features_ sections below for
additional instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.5.0 version.

## New Features

### Multi-Tag Rules

The `polaris_tag_rule` resource and `polaris_tag_rule` data source now support multiple tag conditions via the new `tag`
block. Previously a tag rule could only match a single key and value. With this change, a rule can match any number of
key-value conditions, and each condition can independently match all values for a given key.

The following example shows how to create a tag rule that matches two tag conditions:
```terraform
resource "polaris_tag_rule" "example" {
  name        = "My Tag Rule"
  object_type = "AWS_EC2_INSTANCE"

  tag {
    key    = "Environment"
    values = ["Production", "Staging"]
  }

  tag {
    key       = "Owner"
    match_all = true
  }
}
```

Each `tag` block accepts the following arguments:
* `key` — (Required) Tag key to match.
* `values` — (Optional) List of tag values to match. If empty and `match_all` is false, matches resources where the
  tag value is empty.
* `match_all` — (Optional) If true, all tag values for the given key are matched. Cannot be combined with `values`.
  Defaults to `false`.

### New Feature Blocks
The following feature blocks have been added to the `polaris_aws_account` resource:
* `cloud_discovery` - Cloud Discovery.
* `cloud_native_archival` - Cloud Native Archival.
* `cloud_native_dynamodb_protection` - Cloud Native DynamoDB Protection.
* `cloud_native_s3_protection` - Cloud Native S3 Protection.
* `kubernetes_protection` - Kubernetes Protection.
* `rds_protection` - RDS Protection.
* `servers_and_apps` - Servers and Apps.

The following example shows how to use the `cloud_native_s3_protection` feature block to onboard an AWS account with
Cloud Native S3 Protection:
```terraform
resource "polaris_aws_account" "example" {
  cloud_native_s3_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-1",
    ]
  }
}
```

### Optional Cloud Native Protection
The `cloud_native_protection` feature block has changed from required to optional, allowing the `polaris_aws_account`
resource to be used for onboarding accounts with any combination of the above features.

## Significant Changes

### polaris_tag_rule: deprecated fields

The `tag_key`, `tag_value`, and `tag_all_values` fields of the `polaris_tag_rule` resource are now deprecated. Use the
`tag` block instead. Existing configurations using the deprecated fields will continue to work without any changes and
without producing spurious plan diffs, but the fields will be removed in a future release.

To migrate an existing configuration from the deprecated fields to the `tag` block, replace the deprecated fields with a
`tag` block. For example, the following configuration using the deprecated fields:
```terraform
data "polaris_sla_domain" "gold" {
  name = "Gold"
}

resource "polaris_tag_rule" "example" {
  name        = "My Tag Rule"
  object_type = "AWS_EC2_INSTANCE"
  tag_key     = "Environment"
  tag_value   = "Production"
}

resource "polaris_sla_domain_assignment" "example" {
  sla_domain_id = data.polaris_sla_domain.gold.id
  object_ids    = [polaris_tag_rule.example.id]
}
```
should be migrated to:
```terraform
data "polaris_sla_domain" "gold" {
  name = "Gold"
}

resource "polaris_tag_rule" "example" {
  name        = "My Tag Rule"
  object_type = "AWS_EC2_INSTANCE"

  tag {
    key    = "Environment"
    values = ["Production"]
  }
}

resource "polaris_sla_domain_assignment" "example" {
  sla_domain_id = data.polaris_sla_domain.gold.id
  object_ids    = [polaris_tag_rule.example.id]
}
```
And a configuration using `tag_all_values`:
```terraform
resource "polaris_tag_rule" "example" {
  name           = "My Tag Rule"
  object_type    = "AWS_EC2_INSTANCE"
  tag_key        = "Environment"
  tag_all_values = true
}
```
should be migrated to:
```terraform
resource "polaris_tag_rule" "example" {
  name        = "My Tag Rule"
  object_type = "AWS_EC2_INSTANCE"

  tag {
    key       = "Environment"
    match_all = true
  }
}
```

Since all tag-related fields are `ForceNew`, changing from the deprecated fields to the `tag` block will destroy and
recreate the resource. If the tag rule is paired with a `polaris_sla_domain_assignment`, destroying the tag rule will
temporarily remove the SLA domain assignment until the tag rule is recreated with its new ID. To avoid this gap,
migrate using the following steps:

1. Note the tag rule ID from the current Terraform state:
```shell
% terraform state show polaris_tag_rule.example
```
2. Update the configuration to use the `tag` block (as shown above).
3. Remove the resource from state without destroying the remote object:
```shell
% terraform state rm polaris_tag_rule.example
```
4. Import the existing remote object into the updated configuration:
```shell
% terraform import polaris_tag_rule.example <tag-rule-id>
```
5. Run `terraform plan` to verify that no changes are planned before applying.

The `polaris_sla_domain_assignment` does not need to be imported or modified since the tag rule ID is unchanged.

-> **Note:** The `tag_key`, `tag_value`, and `tag_all_values` fields of the `polaris_tag_rule` data source are also
deprecated. The data source always populates both the `tag` block and the deprecated fields for backward compatibility,
so no immediate action is required for data sources.

### Permission Groups
The `permission_groups` field is now `Required` for the `cloud_native_protection` and `exocompute` feature blocks in the
`polaris_aws_account` resource. Previously, `permission_groups` was `Optional` and could be omitted. Not having
`permission_groups` included in the Terraform configuration will result in an error similar to the following:
```console
╷
│ Error: Missing required argument
│
│   on main.tf line 1, in resource "polaris_aws_account" "example":
│    1: resource "polaris_aws_account" "example" {
│
│ The argument "permission_groups" is required, but no definition was found.
╵
```
To resolve this error, add `permission_groups` to each `cloud_native_protection` and `exocompute` feature block. For
example:
```terraform
resource "polaris_aws_account" "example" {
  cloud_native_protection {
    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
    ]

    regions = [
      "us-east-1",
    ]
  }

  exocompute {
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]

    regions = [
      "us-east-1",
    ]
  }
}
```
