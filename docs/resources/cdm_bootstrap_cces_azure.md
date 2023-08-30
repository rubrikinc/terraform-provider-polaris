---
page_title: "polaris_cdm_bootstrap_cces_azure Resource - terraform-provider-polaris"
subcategory: ""
description: |-

---

# polaris_cdm_bootstrap_cces_azure (Resource)

## Example Usage

```terraform
resource "polaris_cdm_bootstrap_cces_azure" "default" {
  admin_email            = "admin@example.org"
  admin_password         = "password"
  cluster_name           = "my-cluster"
  cluster_nodes          = {
    "my-cluster-node-1" = "10.1.100.100",
    "my-cluster-node-2" = "10.1.100.101",
    "my-cluster-node-3" = "10.1.100.102",
  }
  connection_string      = ""
  container_name         = "my-cluster"
  dns_search_domain      = ["example.org"]
  dns_name_servers       = ["10.1.150.100", "10.1.150.200"]
  enable_immutability    = true
  management_gateway     = "10.1.100.1"
  management_subnet_mask = "255.255.255.0"
  ntp_server1_name       = "10.1.200.100"
  ntp_server2_name       = "10.1.200.200"
}
```

## Schema

### Required

- `admin_email` (String) The Rubrik cluster sends messages for the admin account to this email address.
- `admin_password` (String, Sensitive) Password for the admin account.
- `cluster_name` (String) Unique name to assign to the Rubrik cluster.
- `cluster_nodes` (Map of String) The node name and IP formatted as a map.
- `connection_string` (String) The connection string for the Azure storage account where CCES will store its data.
- `container_name` (String) The name of the container in the Azure storage account where CCES will store its data.
- `dns_name_servers` (List of String) IPv4 addresses of DNS servers.
- `dns_search_domain` (List of String) The search domain that the DNS Service will use to resolve hostnames that are not fully qualified.
- `management_gateway` (String) IP address assigned to the management network gateway
- `management_subnet_mask` (String) Subnet mask assigned to the management network.
- `ntp_server1_name` (String) IP address for NTP server #1.
- `ntp_server2_name` (String) IP address for NTP server #2.

### Optional

- `enable_immutability` (Boolean) Flag to determine if versioning will be used on the Azure Blob storage to enable immutability.
- `ntp_server1_key` (String) Symmetric key material for NTP server #1.
- `ntp_server1_key_id` (Number) Key id number for NTP server #1 (typically this is 0).
- `ntp_server1_key_type` (String) Symmetric key type for NTP server #1.
- `ntp_server2_key` (String) Symmetric key material for NTP server #2.
- `ntp_server2_key_id` (Number) Key id number for NTP server #2 (typically this is 1).
- `ntp_server2_key_type` (String) Symmetric key type for NTP server #2.
- `timeout` (String) The time to wait to establish a connection the Rubrik cluster before returning an error (defaults to `4m`).
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `wait_for_completion` (Boolean) Flag to determine if Terraform should wait for the bootstrap process to complete.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) Create resource timeout (defaults to `40m`).
- `default` (String) Default resource timeout (defaults to `20m`).
- `read` (String) Read resource timeout (defaults to `20m`).
