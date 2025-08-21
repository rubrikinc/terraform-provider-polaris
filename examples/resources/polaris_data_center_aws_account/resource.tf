resource "polaris_data_center_aws_account" "account" {
  name        = "dc-archival-account"
  description = "AWS account used for data center archival"
  access_key  = "AKIAIOSFODNN7EXAMPLE"
  secret_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
