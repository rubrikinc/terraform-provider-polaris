data "polaris_azure_subscription" "subscription" {
  name = "example"
}

output "cloud_account_id" {
  value = data.polaris_azure_subscription.subscription.id
}
