---
page_title: "AWS CNP Account"
---

# Adding an AWS account without CloudFormation
The `polaris_aws_account` resource uses a CloudFormation stack to grant RSC permissions to access AWS. The permissions
granted to RSC by the CloudFormation stack can be difficult to understand and track as RSC will request the permissions
to be updated as new features, requiring new permissions, are released.

To make the process of granting AWS permissions more transparent, a couple of new resources and data sources have been
added to the RSC Terraform provider:
 * `polaris_aws_cnp_account` _(Resource)_
 * `polaris_aws_cnp_account_attachments` _(Resource)_
 * `polaris_aws_cnp_artifacts`  _(Data Source)_
 * `polaris_aws_cnp_permissions`  _(Data Source)_
 * `polaris_account` _(Data Source)_

Using these resources, it's possible to add an AWS account to RSC without using a CloudFormation stack.

To add an AWS account to RSC using the new CNP resources, start by using the `polaris_aws_cnp_artifacts` data source:
```hcl
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
    ]
  }
}
```
One or more `feature` blocks lists the RSC features to enabled for the AWS account. Use the `polaris_account` data
source to obtain a list of RSC features available for the RSC account. The `polaris_aws_cnp_artifacts` data source
returns the instance profiles and roles, referred to as _artifacts_ by RSC, which are required by RSC.

Next, use the `polaris_aws_cnp_permissions` data source to obtain the role permission policies, customer managed
policies and managed policies, required by RSC:
```hcl
data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  role_key = each.key

  dynamic "feature" {
    for_each = data.polaris_aws_cnp_artifacts.artifacts.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}
```

After defining the two data sources, use the `polaris_aws_cnp_account` resource to start the onboarding of the AWS
account:
```hcl
resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"

  regions = [
    "us-east-2",
    "us-west-2",
  ]

  dynamic "feature" {
    for_each = polaris_aws_cnp_artifacts.artifacts.features
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}
```
`name` is the name given to the AWS account in RSC, `native_id` is the AWS account ID and `regions` the AWS regions to
protect with RSC. When Terraform processes this resource, the AWS account will show up in the connecting state in the
RSC UI.

In addition to the fields mentioned above, the `polaris_aws_cnp_account` resource has a computed field called
`trust_policies`, which holds the IAM trust policies allowing RSC to assume the roles to elevate its privileges for
various tasks.

The next step is to create the required IAM instance profiles and roles using the AWS Terraform provider:
```hcl
locals {
  trust_policies = {
    for policy in polaris_aws_cnp_account.account.trust_policies : policy.role_key => policy.policy
  }
}

resource "aws_iam_instance_profile" "profile" {
  for_each    = data.polaris_aws_cnp_artifacts.artifacts.instance_profile_keys
  name_prefix = "rubrik-${lower(each.key)}-"
  role        = aws_iam_role.role[each.value].name
}

resource "aws_iam_role" "role" {
  for_each            = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  assume_role_policy  = local.trust_policies[each.key]
  managed_policy_arns = data.polaris_aws_cnp_permissions.permissions[each.key].managed_policies
  name_prefix         = "rubrik-${lower(each.key)}-"

  dynamic "inline_policy" {
    for_each = data.polaris_aws_cnp_permissions.permissions[each.key].customer_managed_policies
    content {
      name   = inline_policy.value["name"]
      policy = inline_policy.value["policy"]
    }
  }
}
```
The permissions given to each role will be displayed in the output of the Terraform command. A detailed explanation of
the `aws_iam_instance_profile` and `aws_iam_role` resources can be found in the AWS Terraform provider
[documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs). Note, the above example uses IAM
customer inline policies which has a size limit of 10200 bytes, see the
[aws_cnp_account](https://github.com/rubrikinc/terraform-provider-polaris-examples/tree/main/aws_cnp_account) for an
example of how to work around this limit.

Lastly, to finalize the onboarding of the AWS account, use the `polaris_aws_cnp_account_attachments` resource:
```hcl
resource "polaris_aws_cnp_account_attachments" "attachments" {
  account_id = polaris_aws_cnp_account.account.id
  features   = polaris_aws_cnp_account.account.feature.*.name

  dynamic "instance_profile" {
    for_each = aws_iam_instance_profile.profile
    content {
      key  = instance_profile.key
      name = instance_profile.value["arn"]
    }
  }

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
This attaches the instance profiles and roles to the AWS account in RSC. When Terraform processes this resource the AWS
account will transition from the connecting state to the connected state in the RSC UI. Note the `permissions` field of
the `polaris_aws_cnp_account_attachments` resource requires version `0.10.0` or later of the provider.
