---
page_title: "polaris_azure_cloud_cluster Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_azure_cloud_cluster` resource creates an Azure cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in Azure using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual Azure infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The Azure subscription must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.

~> **Note:** Cloud Cluster deletion is now supported. When destroying this resource,
   the cluster will be removed from RSC. If the cluster has blocking conditions
   (active SLAs, global SLAs, or RCV locations), the deletion will fail and you must
   resolve these conditions first. Use the 'force_cluster_delete_on_destroy' option
   to force removal when eligible.

---

# polaris_azure_cloud_cluster (Resource)


The `polaris_azure_cloud_cluster` resource creates an Azure cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in Azure using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual Azure infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The Azure subscription must be onboarded to RSC with the Server and Apps
   feature enabled before creating a cloud cluster.

~> **Note:** Cloud Cluster deletion is now supported. When destroying this resource,
   the cluster will be removed from RSC. If the cluster has blocking conditions
   (active SLAs, global SLAs, or RCV locations), the deletion will fail and you must
   resolve these conditions first. Use the 'force_cluster_delete_on_destroy' option
   to force removal when eligible.



## Example Usage

```terraform
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
```

## Schema

### Required

- `cloud_account_id` (String) RSC cloud account ID (UUID).
- `cluster_config` (Block List, Min: 1, Max: 1) Configuration for the cloud cluster. Changing this forces a new resource to be created. (see [below for nested schema](#nestedblock--cluster_config))
- `vm_config` (Block List, Min: 1, Max: 1) VM configuration for the cluster nodes. Changing this forces a new resource to be created. (see [below for nested schema](#nestedblock--vm_config))

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) Cloud cluster ID (UUID).

<a id="nestedblock--cluster_config"></a>
### Nested Schema for `cluster_config`

Required:

- `admin_email` (String) Email address for the cluster admin user. Changing this value will have no effect on the cluster.
- `admin_password` (String, Sensitive) Password for the cluster admin user. Changing this value will have no effect on the cluster.
- `cluster_name` (String) Unique name to assign to the cloud cluster.
- `dns_name_servers` (Set of String) DNS name servers for the cluster.
- `keep_cluster_on_failure` (Boolean) Whether to keep the cluster on failure (can be useful for troubleshooting). Changing this forces a new resource to be created.
- `ntp_servers` (Set of String) NTP servers for the cluster.
- `num_nodes` (Number) Number of nodes in the cluster. Changing this forces a new resource to be created.

Optional:

- `dns_search_domains` (Set of String) DNS search domains for the cluster.
- `location` (String) Location for the cluster.
- `timezone` (String) Timezone for the cluster.


<a id="nestedblock--vm_config"></a>
### Nested Schema for `vm_config`

Required:

- `cdm_version` (String) CDM version to use. Changing this forces a new resource to be created.
- `container_name` (String) Azure storage container name for the cluster. Changing this forces a new resource to be created.
- `enable_immutability` (Boolean) Whether to enable immutability for the storage account. Changing this forces a new resource to be created.
- `instance_type` (String) Azure instance type for the cluster nodes. Allowed values are `STANDARD_DS5_V2`, `STANDARD_D16S_V5`, `STANDARD_D8S_V5`, `STANDARD_D32S_V5`, `STANDARD_E16S_V5`, `STANDARD_D8AS_V5`, `STANDARD_D16AS_V5`, `STANDARD_D32AS_V5` and `STANDARD_E16AS_V5`. Changing this forces a new resource to be created.
- `network_resource_group` (String) Azure resource group name for network resources. Changing this forces a new resource to be created.
- `network_security_group` (String) Azure network security group name. Changing this forces a new resource to be created.
- `network_security_resource_group` (String) Azure resource group name for the network security group. Changing this forces a new resource to be created.
- `region` (String) Azure region to deploy the cluster in. The format should be the native Azure format, e.g. `eastus`, `westus`, etc. Changing this forces a new resource to be created.
- `resource_group_name` (String) Azure resource group name where the cluster will be deployed. Changing this forces a new resource to be created.
- `storage_account_name` (String) Azure storage account name for the cluster. Changing this forces a new resource to be created.
- `subnet` (String) Azure subnet name for the cluster nodes. Changing this forces a new resource to be created.
- `user_assigned_managed_identity_name` (String) Name of the user-assigned managed identity. Changing this forces a new resource to be created.
- `vnet` (String) Azure virtual network name. Changing this forces a new resource to be created.
- `vnet_resource_group` (String) Azure resource group name for the virtual network. Changing this forces a new resource to be created.

Optional:

- `availability_zone` (String) Availability zone for the cluster. If not specified, the cluster will be deployed in availability zone 1. Changing this forces a new resource to be created.
- `vm_type` (String) VM type for the cluster. Changing this forces a new resource to be created. Possible values are `STANDARD`, `DENSE` and `EXTRA_DENSE`. `DENSE` is recommended for CCES.

Read-Only:

- `cdm_product` (String) CDM Product Code. This is a read-only field and computed based on the CDM version.


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) Create resource timeout (defaults to `60m`).
- `default` (String) Default resource timeout (defaults to `20m`).
- `read` (String) Read resource timeout (defaults to `20m`).