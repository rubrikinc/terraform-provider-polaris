resource "polaris_aws_cnp_account" "account" {
  features  = ["CLOUD_NATIVE_PROTECTION"]
  name      = "My Account"
  native_id = "123456789123"
  regions   = ["us-east-2", "us-west-2"]
}
