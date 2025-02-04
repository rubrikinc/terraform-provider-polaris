data "polaris_data_center_aws_account" "archival" {
  name = "archival-account"
}

output "cloud_account_id" {
  value = data.polaris_data_center_aws_account.archival.id
}
