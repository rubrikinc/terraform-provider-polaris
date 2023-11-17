resource "polaris_aws_private_container_registry" "default" {
  account_id = polaris_aws_account.default.id
  url        = "https://123456789012.dkr.ecr.us-east-2.amazonaws.com"
}
