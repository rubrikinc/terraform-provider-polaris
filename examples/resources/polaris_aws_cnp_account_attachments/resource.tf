# The configuration assumes that an AWS account has been been added
# to RSC and that one AWS IAM instance profile and role has been
# created for each RSC artifact.
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
      key = role.key
      arn = role.value["arn"]
    }
  }
}
