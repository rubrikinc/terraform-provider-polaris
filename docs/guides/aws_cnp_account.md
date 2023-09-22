---
page_title: "AWS CNP Account"
---

# Adding an AWS account without CloudFormation
The `polaris_aws_account` resource uses a CloudFormation stack to grant RSC permissions to access AWS. The permissions
granted to RSC by the CloudFormation stack can be difficult to understand and track as RSC will request the permissions
to be updated as new features, requiring new permissions, are released.

To make the whole AWS permissions process more transparent a couple of new resources and data sources has been added to
the RSC Terraform provider:
 * `polaris_aws_cnp_account`
 * `polaris_aws_cnp_account_attachments`
 * `polaris_aws_cnp_account_trust_policy`
 * `polaris_aws_cnp_artifacts`
 * `polaris_aws_cnp_permissions`
 * `polaris_features`

Using these resources, it's possible to add an AWS account to RSC without using a CloudFormation stack.

To add an AWS account to RSC using the new CNP resources, start by using the `polaris_aws_cnp_artifacts` data source:
```terraform
data "polaris_aws_cnp_artifacts" "artifacts" {
  features = ["CLOUD_NATIVE_PROTECTION"]
}
```
`features` lists the RSC features to enabled for the AWS account. Use the `polaris_features` data source to obtain a
list of RSC features available for the RSC account. The `polaris_aws_cnp_artifacts` data source returns the instance
profiles and roles, referred to as _artifacts_ by RSC, required by RSC.

Next, use the `polaris_aws_cnp_permissions` data source to obtain the role permission policies, customer managed
policies and managed policies, required by RSC:
```terraform
data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  features = data.polaris_aws_cnp_artifacts.artifacts.features
  role_key = each.key
}
```

After defining the two data sources, use the `polaris_aws_cnp_account` resource to start the onboarding of the AWS
account:
```terraform
resource "polaris_aws_cnp_account" "account" {
  features  = polaris_aws_cnp_artifacts.artifacts.features
  name      = "My Account"
  native_id = "123456789123"
  regions   = ["us-east-2", "us-west-2"]
}
```
`name` is the name given to the AWS account in RSC, `native_id` is the AWS account ID and `regions` the AWS regions.
When Terraform processes this resource the AWS account will show up in the connecting state in the RSC UI.

Next, the `polaris_aws_cnp_account_trust_policy` resource needs to be used to define the trust policies required by RSC
for the AWS account:
```terraform
resource "polaris_aws_cnp_account_trust_policy" "trust_policy" {
  for_each    = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  account_id  = polaris_aws_cnp_account.account.id
  features    = polaris_aws_cnp_account.account.features
  role_key    = each.key
}
```
This resource provides the trust policies to attach to the IAM roles created, so that RSC can assume the roles to
elevate it's permissions for various tasks.

Now it's time to create the required IAM instance profiles and roles using the AWS Terraform provider:
```terraform
resource "aws_iam_instance_profile" "profile" {
  for_each    = data.polaris_aws_cnp_artifacts.artifacts.instance_profile_keys
  name_prefix = "rubrik-${lower(each.key)}-"
  role        = aws_iam_role.role[each.value].name
}

resource "aws_iam_role" "role" {
  for_each            = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  assume_role_policy  = polaris_aws_cnp_account_trust_policy.trust_policy[each.key].policy
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
[documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs).

Lastly, to finalize the onboarding of the AWS account, use the `polaris_aws_cnp_account_attachments` resource:
```terraform
resource "polaris_aws_cnp_account_attachments" "attachments" {
  account_id = polaris_aws_cnp_account.account.id
  features   = polaris_aws_cnp_account.account.features

  dynamic "instance_profile" {
    for_each = aws_iam_instance_profile.profile
    content {
      key  = instance_profile.key
      name = instance_profile.value["name"]
    }
  }

  dynamic "role" {
    for_each = aws_iam_role.role
    content {
      key = role.key
      arn = role.value["arn"]
    }
  }
}
```
This attaches the instance profiles and roles to the AWS account in RSC. When Terraform processes this resource the AWS
account will go from the connecting state to the connected state in the RSC UI.
