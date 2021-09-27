resource "polaris_azure_exocompute" "default" {
  subscription_id = polaris_azure_subscription.default.id
  region          = "eastus2"
  subnet_id       = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test/subnets/default"
}
