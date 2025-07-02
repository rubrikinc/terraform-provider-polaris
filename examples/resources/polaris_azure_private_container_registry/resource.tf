data "polaris_azure_subscription" "subscription" {
  name = "subscription"
}

resource "polaris_azure_private_container_registry" "registry" {
  cloud_account_id = data.polaris_azure_subscription.subscription.id
  app_id           = "927dd42c-5517-4de0-944c-466b3e3c6e70"
  url              = "234567890121.dkr.ecr.us-east-2.amazonaws.com"
}
