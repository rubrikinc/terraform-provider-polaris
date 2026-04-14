---
page_title: "Upgrade Guide: v1.6.3"
---

# Upgrade Guide v1.6.3

## Before Upgrading

Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider.

## How to Upgrade

Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.6.3 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.6.0"
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
This will read the remote state of the resources and migrate the local Terraform state to the v1.6.3 version.

## New Features

### polaris_feature_flag data source

The new `polaris_feature_flag` data source checks if a feature flag is enabled for the RSC account.

```terraform
data "polaris_feature_flag" "my_feature_flag" {
  name = "MY_FEATURE_FLAG"
}

output "my_feature_flag_enabled" {
  value = data.polaris_feature_flag.my_feature_flag.enabled
}
```

For more details, see the [polaris_feature_flag documentation](../data-sources/feature_flag.md).

### polaris_aws_cnp_account: role chaining support

The `polaris_aws_cnp_account` and `polaris_aws_cnp_account_attachments` resources now support the `ROLE_CHAINING`
feature and the `role_chaining_account_id` field for cross-account role chaining. The `polaris_aws_cnp_artifacts` and
the `polaris_aws_cnp_permissions` data sources have also been updated with role chaining support.

To onboard the role-chaining AWS account, use the `ROLE_CHAINING` feature with the `BASIC` permission group:
```terraform
# Lookup the artifacts required for the role-chaining feature.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "ROLE_CHAINING"
    permission_groups = [
      "BASIC",
    ]
  }
}

# Lookup the permissions required for each role artifact for the
# role-chaining feature.
data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  role_key = each.key

  feature {
    name = "ROLE_CHAINING"
    permission_groups = [
      "BASIC",
    ]
  }
}

# Start onboarding the specified account as the role-chaining account.
resource "polaris_aws_cnp_account" "account" {
  name      = "Role-chaining Account"
  native_id = "123456789123"

  feature {
    name = "ROLE_CHAINING"
    permission_groups = [
      "BASIC",
    ]
  }

  regions = [
    "us-east-2",
  ]
}

# At this point the AWS Terraform provider should be used to create
# the required role artifacts. The polaris_aws_cnp_artifacts and the
# polaris_aws_cnp_permissions data sources together with the
# trust_policies field from the polaris_aws_cnp_account resource is
# used as input.

# Attach the role artifacts to the role-chaining account.
resource "polaris_aws_cnp_account_attachments" "attachments" {
  account_id = polaris_aws_cnp_account.account.id

  features = [
    "ROLE_CHAINING"
  ]
  
  dynamic "role" {
    for_each = aws_iam_role.role
    content {
      key         = role.key
      arn         = role.value["arn"]
      permissions = data.polaris_aws_cnp_permissions.permissions[role.key].id
    }
  }
}
```
Note that the `ROLE_CHAINING` feature is mutually exclusive with all other RSC features.

To onboard a role-chained AWS account, i.e. an AWS account using the role-chaining AWS account, proceed as you normally
would when you onboard an AWS account, but pass the role-chaining RSC cloud account ID (UUID) to the
`polaris_aws_cnp_account` and the `polaris_aws_cnp_account_attachments` resources using the `role_chaining_account_id`
field.

For a complete example of how to onboard an AWS account to RSC, look at the
[AWS IAM account](https://github.com/rubrikinc/terraform-provider-polaris-examples/tree/main/modules/aws_iam_account)
module in the [examples](https://github.com/rubrikinc/terraform-provider-polaris-examples/tree/main) repository. 

Note that the `polaris_aws_cnp_account_trust_policy` resource does not support role chaining. Use the `trust_policies`
field of the `polaris_aws_cnp_account` resource when using role chaining.

For more details, see the [polaris_aws_cnp_account documentation](../resources/aws_cnp_account.md) and the
[polaris_aws_cnp_account_attachments documentation](../resources/aws_cnp_account_attachments.md).

## Bug Fixes

### polaris_azure_archival_location: storage_account_name_prefix max length fix

The `storage_account_name_prefix` field was hardcoded to a maximum of 14 characters, but the backend accepts up to 16
for `SOURCE_REGION` (prefix + 8-character UID = 24, Azure's max) and up to 24 for `SPECIFIC_REGION` (name used
directly). The provider now enforces the correct limit based on whether `storage_account_region` is set.
