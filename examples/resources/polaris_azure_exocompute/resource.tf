data "polaris_azure_subscription" "host" {
  name = "host-subscription"
}

# RSC managed Exocompute.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id         = data.polaris_azure_subscription.host.id
  pod_overlay_network_cidr = "10.244.0.0/16"
  region                   = "eastus2"
  subnet                   = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test/subnets/default"
}

# Customer managed Exocompute.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id = data.polaris_azure_subscription.host.id
  region           = "eastus2"
}

resource "polaris_azure_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-aks-cluster"
  exocompute_id = polaris_azure_exocompute.host.id
}


data "polaris_azure_subscription" "application" {
  name = "application-subscription"
}

# Application Exocompute.
resource "polaris_azure_exocompute" "application" {
  cloud_account_id      = data.polaris_azure_subscription.application.id
  host_cloud_account_id = data.polaris_azure_subscription.host.id
}
