data "polaris_data_center_azure_subscription" "archival" {
  name = "archival-subscription"
}

output "cloud_account_id" {
  value = data.polaris_data_center_azure_subscription.archival.id
}
