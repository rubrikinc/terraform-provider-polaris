data "polaris_aws_account" "account" {
  name = "example"
}

output "cloud_account_id" {
  value = data.polaris_aws_account.account.id
}
