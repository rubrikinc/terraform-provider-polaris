data "polaris_azure_subscription" "host" {
  name = "host-subscription"
}

# RSC managed Exocompute.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id         = data.polaris_azure_subscription.host.id
  pod_overlay_network_cidr = "10.244.0.0/16"
  region                   = "eastus2"
  subnet                   = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../virtualNetworks/test/subnets/default"
}

# RSC managed Exocompute with optional configuration.
resource "polaris_azure_exocompute" "host" {
  cloud_account_id         = data.polaris_azure_subscription.host.id
  pod_overlay_network_cidr = "10.244.0.0/16"
  region                   = "eastus2"
  subnet                   = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../virtualNetworks/test/subnets/default"

  optional_config {
    allowlist_additional_ips            = ["1.2.3.4"]
    allowlist_rubrik_ips                = true
    cluster_access                      = "AKS_CLUSTER_ACCESS_TYPE_PRIVATE"
    cluster_tier                        = "AKS_CLUSTER_TIER_FREE"
    disk_encryption_at_host             = true
    max_node_count                      = "AKS_NODE_COUNT_BUCKET_SMALL"
    private_exocompute_dns_zone_id      = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/privatelink.eastus2.azmk8s.io"
    resource_group_prefix               = "my-resource-group-prefix"
    snapshot_private_access_dns_zone_id = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/privatelink.blob.core.windows.net"
    user_defined_routing                = true
  }
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
