# Create an Azure cloud cluster using RSC
resource "polaris_azure_cloud_cluster" "example" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"

  cluster_config {
    cluster_name            = "my-cloud-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    dns_search_domains      = ["example.com"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version                     = "9.2.3-p7-29713"
    instance_type                   = "STANDARD_D8S_V5"
    location                        = "westus"
    resource_group                  = "my-resource-group"
    network_resource_group          = "my-network-resource-group"
    vnet_resource_group             = "my-vnet-resource-group"
    subnet                          = "my-subnet"
    vnet                            = "my-vnet"
    network_security_group          = "my-network-security-group"
    network_security_resource_group = "my-network-security-resource-group"
    vm_type                         = "EXTRA_DENSE"
    availability_zone               = "1"
  }
}

# Create an Azure cloud cluster with Multi-AZ resiliency
resource "polaris_azure_cloud_cluster" "multi_az" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  az_resilient     = true

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version                     = "9.2.3-p7-29713"
    instance_type                   = "STANDARD_D8S_V5"
    location                        = "westus"
    resource_group                  = "my-resource-group"
    network_resource_group          = "my-network-resource-group"
    vnet_resource_group             = "my-vnet-resource-group"
    vnet                            = "my-vnet"
    network_security_group          = "my-network-security-group"
    network_security_resource_group = "my-network-security-resource-group"
    vm_type                         = "EXTRA_DENSE"
    storage_account                 = "my-storage-account"
    container_name                  = "my-container"
    enable_immutability             = true
    user_assigned_managed_identity  = "my-managed-identity"

    subnet_az_config {
      availability_zone = "1"
      subnet            = "subnet-zone-1"
    }

    subnet_az_config {
      availability_zone = "2"
      subnet            = "subnet-zone-2"
    }

    subnet_az_config {
      availability_zone = "3"
      subnet            = "subnet-zone-3"
    }
  }
}
