# Source region.
resource "polaris_azure_archival_location" "archival_location" {
  cloud_account_id            = polaris_azure_subscription.subscription.id
  name                        = "my-archival-location"
  storage_account_name_prefix = "archival"
}

# Source region with a customer managed key.
resource "polaris_azure_archival_location" "archival_location" {
  cloud_account_id            = polaris_azure_subscription.subscription.id
  name                        = "my-archival-location"
  storage_account_name_prefix = "archival"

  customer_managed_key {
    name       = "my-archival-key"
    region     = "eastus"
    vault_name = "my-archival-key-vault"
  }
}

# Specific region.
resource "polaris_azure_archival_location" "archival_location" {
  cloud_account_id            = polaris_azure_subscription.subscription.id
  name                        = "my-archival-location"
  storage_account_name_prefix = "archival"
  storage_account_region      = "eastus2"
}
