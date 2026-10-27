---
page_title: "polaris_azure_exocompute Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration
for Azure workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When a host configuration is created, RSC will
    automatically deploy the necessary resources in the specified Azure region
    to run the Exocompute service. A host configuration can be used by both the
    host cloud account and application cloud accounts mapped to the host
    account.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the Azure
    AKS cluster attached by the customer, using the
    `polaris_azure_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Item 1 and 2 above requires that the Azure subscription has been onboarded with
the `exocompute` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_azure_exocompute` resource:
 1. Using the `cloud_account_id`, `region`, `subnet` and
   `pod_overlay_network_cidr` fields creates an RSC managed host configuration.
 2. Using the `cloud_account_id` and `region` fields creates a customer managed
    host configuration. Note, the `polaris_azure_exocompute_cluster_attachment`
    resource must be used to attach an Azure AKS cluster to the Exocompute
    configuration.
 3. Using the `cloud_account_id` and `host_cloud_account_id` fields creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   `pod_overlay_network_cidr` field, this is discouraged and should only be done
   for backwards compatibility reasons.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.

---

# polaris_azure_exocompute (Resource)


The `polaris_azure_exocompute` resource creates an RSC Exocompute configuration
for Azure workloads.

There are 3 types of Exocompute configurations:
 1. *RSC Managed Host* - When a host configuration is created, RSC will
    automatically deploy the necessary resources in the specified Azure region
    to run the Exocompute service. A host configuration can be used by both the
    host cloud account and application cloud accounts mapped to the host
    account.
 2. *Customer Managed Host* - When a customer managed host configuration is
    created, RSC will not deploy any resources. Instead it will use the Azure
    AKS cluster attached by the customer, using the
    `polaris_azure_exocompute_cluster_attachment` resource, for all operations.
 3. *Application* - An application configuration is created by mapping the
    application cloud account to a host cloud account. The application cloud
    account will leverage the Exocompute resources deployed for the host
    configuration.

Item 1 and 2 above requires that the Azure subscription has been onboarded with
the `exocompute` feature.

Since there are 3 types of Exocompute configurations, there are 3 ways to create
a `polaris_azure_exocompute` resource:
 1. Using the `cloud_account_id`, `region`, `subnet` and
   `pod_overlay_network_cidr` fields creates an RSC managed host configuration.
 2. Using the `cloud_account_id` and `region` fields creates a customer managed
    host configuration. Note, the `polaris_azure_exocompute_cluster_attachment`
    resource must be used to attach an Azure AKS cluster to the Exocompute
    configuration.
 3. Using the `cloud_account_id` and `host_cloud_account_id` fields creates an
    application configuration.

~> **Note:** A host configuration can be created without specifying the
   `pod_overlay_network_cidr` field, this is discouraged and should only be done
   for backwards compatibility reasons.

-> **Note:** Customer managed Exocompute is sometimes referred to as Bring Your
   Own Kubernetes (BYOK). Using both host and application Exocompute
   configurations is sometimes referred to as shared Exocompute.



## Example Usage

```terraform
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
    disk_encryption_set_id              = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../diskEncryptionSets/my-disk-encryption-set"
    health_check_vm_name_prefix         = "my-health-check"
    max_node_count                      = "AKS_NODE_COUNT_BUCKET_SMALL"
    postgres_private_dns_zone_id        = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/my-server.private.postgres.database.azure.com"
    postgres_subnet_id                  = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../virtualNetworks/test/subnets/postgres"
    private_exocompute_dns_zone_id      = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/privatelink.eastus2.azmk8s.io"
    resource_group_prefix               = "my-resource-group-prefix"
    snapshot_private_access_dns_zone_id = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/privatelink.blob.core.windows.net"
    sql_private_dns_zone_id             = "/subscriptions/65774f88-da6a-11eb-bc8f-e798f8b54eba/.../privateDnsZones/privatelink.database.windows.net"
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
```


## Schema

### Optional

- `cloud_account_id` (String) RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for which the Exocompute service runs. Changing this forces a new resource to be created.
- `host_cloud_account_id` (String) RSC cloud account ID of the shared exocompute host account. Changing this forces a new resource to be created.
- `optional_config` (Block List, Max: 1) (see [below for nested schema](#nestedblock--optional_config))
- `pod_overlay_network_cidr` (String) The CIDR range assigned to pods when launching Exocompute with the CNI overlay network plugin mode. Rubrik recommends a size of /18 or larger. The pod CIDR must not overlap with the cluster subnet or any IP ranges used in on-premises networks and other peered VNets. The default space assigned by Azure is 10.244.0.0/16. Changing this forces a new resource to be created.
- `region` (String) Azure region to run the exocompute service in. Should be specified in the standard Azure style, e.g. `eastus`. Changing this forces a new resource to be created.
- `subnet` (String) Azure subnet ID of the cluster subnet corresponding to the Exocompute configuration. This subnet will be used to allocate IP addresses to the nodes of the cluster. Changing this forces a new resource to be created.
- `subscription_id` (String, Deprecated) RSC cloud account ID. This is the ID of the `polaris_azure_subscription` resource for which the Exocompute service runs. Changing this forces a new resource to be created. **Deprecated:** use `cloud_account_id` instead.

### Read-Only

- `id` (String) Exocompute configuration ID (UUID).

<a id="nestedblock--optional_config"></a>
### Nested Schema for `optional_config`

Optional:

- `allowlist_additional_ips` (Set of String) Allowlist additional IP addresses for the API server on the Kubernetes cluster. Requires that the `allowlist_rubrik_ips` field is set to `true`. Changing this forces a new resource to be created.
- `allowlist_rubrik_ips` (Boolean) Allowlist Rubrik IPs for the API server on the Kubernetes cluster. Defaults to `false`. Changing this forces a new resource to be created.
- `cluster_access` (String) Azure cluster access type. Possible values are `AKS_CLUSTER_ACCESS_TYPE_PUBLIC` and `AKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Defaults to `AKS_CLUSTER_ACCESS_TYPE_PRIVATE`. Changing this forces a new resource to be created.
- `cluster_tier` (String) Azure cluster tier. Possible values are `AKS_CLUSTER_TIER_FREE` and `AKS_CLUSTER_TIER_STANDARD`. Defaults to `AKS_CLUSTER_TIER_FREE`. Changing this forces a new resource to be created.
- `disk_encryption_at_host` (Boolean) Enable disk encryption at host. Defaults to `false`. Changing this forces a new resource to be created.
- `disk_encryption_set_id` (String) Azure resource ID of the disk encryption set used to encrypt the disks of the cluster nodes with customer managed keys. The disk encryption set must be in the same region as the Exocompute configuration. RSC ignores the disk encryption set unless the `CNP_AZURE_AKS_CMK_ENABLED` feature flag is enabled for the RSC account, and RSC does not validate the disk encryption set when the Exocompute configuration is created. To let RSC list the disk encryption sets, add the `ENCRYPTION` permission group to the Exocompute feature of the `polaris_azure_subscription` resource. If not set, the value in RSC is kept. Changing this forces a new resource to be created.
- `health_check_vm_name_prefix` (String) Name prefix of the virtual machine RSC launches for the Exocompute health check. RSC appends a marker and a UUID to the prefix. The prefix can be at most 22 characters long, can only contain letters, digits and hyphens, and must start and end with a letter or a digit. When RSC has no prefix, it uses `test-aks-connectivity-rk-`. If not set, the value in RSC is kept. Changing this forces a new resource to be created.
- `max_node_count` (String) The maximum number of nodes each cluster can use. Make sure you have enough IP addresses in the subnet or a node pool large enough to handle the number of nodes to avoid launch failure. Possible values are `AKS_NODE_COUNT_BUCKET_SMALL` (32 nodes), `AKS_NODE_COUNT_BUCKET_MEDIUM` (64 nodes), `AKS_NODE_COUNT_BUCKET_LARGE` (128 nodes) and `AKS_NODE_COUNT_BUCKET_XLARGE` (256 nodes). Defaults to `AKS_NODE_COUNT_BUCKET_MEDIUM`. Changing this forces a new resource to be created.
- `postgres_private_dns_zone_id` (String) Azure resource ID of the private DNS zone used to resolve the FQDNs of RSC managed Azure Postgres Flexible Servers from the Exocompute subnet. The name of the zone must end with `.postgres.database.azure.com` and the zone must be linked to the Exocompute VNet. Must be specified together with `postgres_subnet_id`. Requires the `REL_ENABLE_AZURE_POSTGRES_FLEXIBLE_SERVER` feature flag to be enabled for the RSC account. If not set, the value in RSC is kept. Changing this forces a new resource to be created.
- `postgres_subnet_id` (String) Azure resource ID of the subnet used for VNet integration of RSC managed Azure Postgres Flexible Servers. RSC uses these servers to back up Azure Postgres Flexible Servers with private access. The subnet must be in the Exocompute VNet, delegated to `Microsoft.DBforPostgreSQL/flexibleServers`, at least `/28` in size, and different from the Exocompute subnet. Must be specified together with `postgres_private_dns_zone_id`. Requires the `REL_ENABLE_AZURE_POSTGRES_FLEXIBLE_SERVER` feature flag to be enabled for the RSC account. If not set, the value in RSC is kept. Changing this forces a new resource to be created.
- `private_exocompute_dns_zone_id` (String) Azure resource ID of the private DNS zone which will resolve the API server URL for a private cluster. If empty, Azure will automatically create a private DNS zone in the node resource group, and will delete it when the AKS cluster is deleted. Changing this forces a new resource to be created.
- `resource_group_prefix` (String) Prefix of resource groups associated with the cluster, such as cluster nodes. Changing this forces a new resource to be created.
- `snapshot_private_access_dns_zone_id` (String) Azure resource ID of the private DNS zone linked to the exocompute VNet, which will resolve private endpoints linked to snapshots. If empty, a new private DNS zone will be created in the Exocompute resource group. Changing this forces a new resource to be created.
- `sql_private_dns_zone_id` (String) Azure resource ID of the private DNS zone used to resolve the private endpoints of the Azure SQL servers RSC uses to back up Azure SQL Databases. The name of the zone must be `privatelink.database.windows.net` and the zone must be linked to the Exocompute VNet. RSC only uses the zone when the `REL_ENABLE_AZURE_SQL_DB_PRIVATE_ENDPOINT_AUTOMATION` feature flag is enabled for the RSC account and the Exocompute feature has the `PRIVATE_ENDPOINTS` permission group. If not set, the value in RSC is kept. Changing this forces a new resource to be created.
- `user_defined_routing` (Boolean) Enable user defined routing. This allows the route for the Exocompute egress traffic to be configured. Defaults to `false`. Changing this forces a new resource to be created.

## Import

To import an application exocompute configuration prepend `app-` to the ID of the configuration.

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_azure_exocompute.host
  id = "a9caddfd-25bd-4327-85f6-fa698ed898b6"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_azure_exocompute.host a9caddfd-25bd-4327-85f6-fa698ed898b6
```

