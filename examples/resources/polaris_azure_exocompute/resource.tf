# Host configuration.
resource "polaris_azure_exocompute" "host_exocompute" {
  cloud_account_id = polaris_azure_subscription.host_subscription.id
  region           = "eastus2"
  subnet           = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test/subnets/default"
}

# Application configuration.
resource "polaris_azure_exocompute" "app_exocompute" {
  cloud_account_id      = polaris_azure_subscription.app_subscription.id
  host_cloud_account_id = polaris_azure_subscription.host_subscription.id
}
