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

### Pod Subnet Support for AWS Exocompute

The `polaris_aws_exocompute` resource now supports a `subnet` block that allows specifying a `pod_subnet_id` for each
cluster subnet. This can be used to address the issue where pods do not receive an IP address due to exhaustion of the
existing IP address space by other resources. Nodes will launch in the existing subnets, while pods will launch in the 
pod subnets.

The following example shows how to create an Exocompute configuration with pod subnets:
```terraform
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnet {
    subnet_id     = "subnet-ea67b67b"
    pod_subnet_id = "subnet-0cf281be"
  }
  subnet {
    subnet_id     = "subnet-ea43ec78"
    pod_subnet_id = "subnet-0f6b8efa"
  }
}
```

The `subnet` block conflicts with the existing `subnets` field. Use `subnet` blocks when you need to specify pod
subnets, and `subnets` when you do not. Both fields are read back on every refresh, so switching between them requires
a resource replacement.

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

The `polaris_aws_account` resource has been updated to support additional feature blocks. The following new feature
blocks have been added:
* `cloud_discovery` — Cloud Discovery.
* `cloud_native_archival` — Cloud Native Archival.
* `cloud_native_dynamodb_protection` — Cloud Native DynamoDB Protection.
* `cloud_native_s3_protection` — Cloud Native S3 Protection.
* `kubernetes_protection` — Kubernetes Protection.
* `rds_protection` — RDS Protection.
* `servers_and_apps` — Servers and Apps.

The full list of currently supported feature blocks is:
* `cloud_discovery` — Cloud Discovery.
* `cloud_native_archival` — Cloud Native Archival.
* `cloud_native_dynamodb_protection` — Cloud Native DynamoDB Protection.
* `cloud_native_protection` — Cloud Native Protection.
* `cloud_native_s3_protection` — Cloud Native S3 Protection.
* `cyber_recovery_data_scanning` — Cyber Recovery Data Scanning.
* `data_scanning` — Data Scanning.
* `dspm` — DSPM.
* `exocompute` — Exocompute.
* `kubernetes_protection` — Kubernetes Protection.
* `outpost` — Outpost.
* `rds_protection` — RDS Protection.
* `servers_and_apps` — Servers and Apps.

The `cloud_native_protection` feature block has changed from required to optional, allowing the `polaris_aws_account`
resource to be used for onboarding accounts with any combination of the supported features. The `outpost` feature block
is still required whenever the `data_scanning`, `dspm`, or `cyber_recovery_data_scanning` feature is onboarded.

### Cloud Discovery

The `cloud_discovery` feature block is currently optional but will become required whenever a protection feature is
onboarded. It can be added to existing AWS accounts and new AWS accounts can be onboarded with it. Once the
`cloud_discovery` feature has been onboarded, it cannot be removed unless all protection features are removed first.

The `CLOUD_DISCOVERY` feature is also now supported in the `polaris_aws_cnp_account` and
`polaris_aws_cnp_account_attachments` resources, and in the `polaris_aws_cnp_artifacts` and
`polaris_aws_cnp_permissions` data sources. As with the `polaris_aws_account` resource, once
onboarded, it cannot be removed unless all protection features are removed first.

## Significant Changes

### polaris_aws_account: separate outpost account resource

It's now possible to manage the AWS outpost account as a separate `polaris_aws_account` resource. When using a separate
account, the outpost account must be onboarded first, using `depends_on` to ensure it is created before and destroyed
after the main account. In addition to this, the `outpost_account_id` and `outpost_account_profile` fields of the
`outpost` feature block are now optional. Using the `outpost_account_id` and `outpost_account_profile` fields for new
accounts is not recommended.

It's possible to migrate an existing `polaris_aws_account` resource that uses the `outpost_account_id` field to two
separate resources by removing the account from the local state and importing it back as two `polaris_aws_account`
resources, one for the outpost account and one for the account with the data scanning and protection features. Note, if
the outpost account is the same as the account with the data scanning and protection features, it should not be split
into two resources.

If the Terraform configuration looks something like this:
```terraform
resource "polaris_aws_account" "account" {
  profile = "account"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2"
    ]
  }

  data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }

  outpost {
    outpost_account_id      = "456789012345"
    outpost_account_profile = "outpost"

    permission_groups = [
      "BASIC",
    ]
  }
}
```

Start by looking up the RSC Cloud account ID for the account. This can be found in the local state for the
`polaris_aws_account` resource. Run `terraform state show polaris_aws_account.account` to find the RSC Cloud account
ID. The output should look something like this:
```shell
resource "polaris_aws_account" "account" {
    id   = "a695fe0f-1b6e-4e9f-974a-b6bec322a535"
    name = "123456789012 : account"

    ...
}
```
The RSC Cloud account ID is the `id` field in the output.

Next, look up the RSC Cloud account ID for the outpost account. This can be done using the AWS account ID and the
RSC API Playground. The following query can be used to find the RSC Cloud account for the outpost account:
```graphql
{
  allAwsCloudAccountsWithFeatures(
    awsCloudAccountsArg: {
      feature: OUTPOST,
      columnSearchFilter: "456789012345",
      statusFilters: [],
    }
  ) {
    awsCloudAccount {
      id
      nativeId
      accountName
    }
  }
}
```
Replace `456789012345` with your outpost account's AWS account ID. The RSC cloud account ID is the `id` field in the
response.

After collecting the RSC cloud account IDs, update the configuration to have two `polaris_aws_account` resources, one
for the outpost account and one for the account with the data scanning and protection features:
```terraform
resource "polaris_aws_account" "outpost" {
  profile = "outpost"

  outpost {
    permission_groups = [
      "BASIC",
    ]
  }
}

resource "polaris_aws_account" "account" {
  profile = "account"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2"
    ]
  }

  data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }

  depends_on = [
    polaris_aws_account.outpost,
  ]
}
```

-> **Note:** The `depends_on` ensures the outpost account is created before and destroyed after the account with the
   data scanning and protection features.

Now the local state needs to be updated to match the new configuration. Remove the resource from the local state without
destroying the remote object:
```shell
% terraform state rm polaris_aws_account.account
```

Import the outpost account using the RSC Cloud account ID looked up using the AWS account ID and RSC API Playground:
```shell
% terraform import polaris_aws_account.outpost <outpost-rsc-cloud-account-id>
```

Import the account with the data scanning and protection features using the RSC Cloud account ID looked up in the
local state:
```shell
% terraform import polaris_aws_account.account <account-rsc-cloud-account-id>
```

Run `terraform apply` to complete the migration. The following in-place updates are expected since `profile` and
`delete_snapshots_on_destroy` are only stored in the local state. The `outpost_account_id` defaults to the AWS account
ID of the `polaris_aws_account` resource.
```shell
Terraform will perform the following actions:

  # polaris_aws_account.account will be updated in-place
  ~ resource "polaris_aws_account" "account" {
      + delete_snapshots_on_destroy = false
        id                          = "a695fe0f-1b6e-4e9f-974a-b6bec322a535"
        name                        = "123456789012 : account"
      + profile                     = "account"

        # (2 unchanged blocks hidden)
    }

  # polaris_aws_account.outpost will be updated in-place
  ~ resource "polaris_aws_account" "outpost" {
      + delete_snapshots_on_destroy = false
        id                          = "3d2abe21-b5f8-4ed4-a11c-f9f13bf1de51"
        name                        = "456789012345 : outpost"
      + profile                     = "outpost"

      ~ outpost {
          - outpost_account_id      = "456789012345" -> null
            # (4 unchanged attributes hidden)
        }
    }

Plan: 0 to add, 2 to change, 0 to destroy.
```

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
