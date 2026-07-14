# Phase 1 of the RSC-managed AWS (BaaS) onboarding flow. Validates the AWS
# account with RSC, registers it, and returns the CloudFormation template
# information used to deploy the RSC cross-account stack.
#
# The account's features and regions are all chosen here. See the
# polaris_aws_account_managed_stack resource for the full flow.

resource "polaris_aws_account_managed" "example" {
  native_id = "123456789012"
  name      = "my-aws-account"

  # features + regions are optional. When omitted, all BaaS-supported values
  # are used (features: EC2 / RDS / S3 / Cloud Discovery).
  # features = ["CLOUD_NATIVE_PROTECTION", "RDS_PROTECTION", "CLOUD_NATIVE_S3_PROTECTION", "CLOUD_DISCOVERY"]
  # regions  = ["us-east-1", "us-west-2", "eu-west-1"]
}

output "cloudformation_template_url" {
  value = polaris_aws_account_managed.example.template_url
}

output "cloudformation_stack_name" {
  value = polaris_aws_account_managed.example.stack_name
}
