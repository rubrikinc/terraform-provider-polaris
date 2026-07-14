# Full RSC-managed AWS (BaaS) onboarding flow:
#
#   1. polaris_aws_account_managed        - validate + finalize; returns the RSC
#                                           account UUID and CloudFormation template.
#   2. aws_cloudformation_stack           - the AWS provider deploys the stack.
#   3. polaris_aws_account_managed_stack  - waits for features to connect and
#                                           completes onboarding.

resource "polaris_aws_account_managed" "example" {
  native_id = "123456789012"
  name      = "my-aws-account"
}

resource "aws_cloudformation_stack" "rubrik" {
  name         = polaris_aws_account_managed.example.stack_name
  template_url = polaris_aws_account_managed.example.template_url
  capabilities = ["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"]
  tags         = {} # avoids the hashicorp/aws empty-tags refresh drift
}

resource "polaris_aws_account_managed_stack" "example" {
  account_id          = polaris_aws_account_managed.example.id
  stack_arn           = aws_cloudformation_stack.rubrik.id
  permissions_version = polaris_aws_account_managed.example.permissions_version
}
