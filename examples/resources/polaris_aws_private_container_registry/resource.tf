resource "polaris_aws_private_container_registry" "registry" {
  account_id = polaris_aws_account.account.id
  native_id  = "123456789012"
  url        = "234567890121.dkr.ecr.us-east-2.amazonaws.com"
}
