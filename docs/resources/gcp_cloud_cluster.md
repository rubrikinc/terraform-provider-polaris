---
page_title: "polaris_gcp_cloud_cluster Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_gcp_cloud_cluster` resource creates a GCP cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in GCP using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual GCP infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The GCP project must be onboarded to RSC with the `SERVERS_AND_APPS`
   feature enabled before creating a cloud cluster.

~> **Note:** This resource requires **Terraform v1.11.0 or later** due to the use of write-only attributes for
   `admin_email` and `admin_password`.

---

# polaris_gcp_cloud_cluster (Resource)


The `polaris_gcp_cloud_cluster` resource creates a GCP cloud cluster using RSC.

This resource creates a Rubrik Cloud Data Management (CDM) cluster with elastic storage
in GCP using the specified configuration. The cluster will be deployed with the specified
number of nodes, instance types, and network configuration.

~> **Note:** This resource creates actual GCP infrastructure. Destroying the
   resource will attempt to clean up the created resources, but manual cleanup
   may be required.

~> **Note:** The GCP project must be onboarded to RSC with the `SERVERS_AND_APPS`
   feature enabled before creating a cloud cluster.

~> **Note:** This resource requires **Terraform v1.11.0 or later** due to the use of write-only attributes for
   `admin_email` and `admin_password`.



## Example Usage

```terraform
# Create a GCP cloud cluster (CCES) using RSC.
resource "polaris_gcp_cloud_cluster" "example" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  region           = "us-west1"
  zone             = "us-west1-a"

  cluster_config {
    cluster_name = "my-cloud-cluster"
    # admin_email and admin_password are write-only (requires Terraform v1.11.0+).
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    dns_search_domains      = ["example.com"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 1
    bucket_name             = "my-gcs-bucket"
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version      = "9.4.0-p2-30507"
    instance_type    = "N2_STANDARD_8"
    network          = "my-vpc"
    subnet           = "my-subnet"
    host_project     = "my-shared-vpc-host-project"
    service_accounts = ["cces-sa@my-project.iam.gserviceaccount.com"]
  }
}

# Create a Multi-AZ (availability-zone resilient) GCP cloud cluster. When
# az_resilient is true, omit `subnet` and provide one `subnet_az_config` block
# per zone. `zone` may be omitted and defaults to the first subnet_az_config
# availability zone.
resource "polaris_gcp_cloud_cluster" "multi_az" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  region           = "us-west1"
  az_resilient     = true

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    bucket_name             = "my-gcs-bucket"
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version      = "9.4.0-p2-30507"
    instance_type    = "N2_STANDARD_8"
    network          = "my-vpc"
    host_project     = "my-shared-vpc-host-project"
    service_accounts = ["cces-sa@my-project.iam.gserviceaccount.com"]

    subnet_az_config {
      availability_zone = "us-west1-a"
      subnet            = "my-subnet"
    }
    subnet_az_config {
      availability_zone = "us-west1-b"
      subnet            = "my-subnet"
    }
    subnet_az_config {
      availability_zone = "us-west1-c"
      subnet            = "my-subnet"
    }
  }
}
```

## Schema

### Required

- `cloud_account_id` (String) RSC cloud account ID (UUID).
- `cluster_config` (Block List, Min: 1, Max: 1) Configuration for the cloud cluster. (see [below for nested schema](#nestedblock--cluster_config))
- `region` (String) GCP region to deploy the cluster in. Changing this forces a new resource to be created.
- `vm_config` (Block List, Min: 1, Max: 1) VM configuration for the cluster nodes. Changing this forces a new resource to be created. (see [below for nested schema](#nestedblock--vm_config))

### Optional

- `az_resilient` (Boolean) Whether to deploy the cluster across multiple availability zones for AZ resiliency. When enabled, `subnet_az_config` blocks must be specified in `vm_config` and `subnet` must be omitted. Requires at least three nodes and a region with at least three zones. Changing this forces a new resource to be created.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `zone` (String) GCP zone to deploy the cluster in. Required when `az_resilient` is false. When `az_resilient` is true it may be omitted and defaults to the first `subnet_az_config` availability zone. Changing this forces a new resource to be created.

### Read-Only

- `id` (String) Cloud cluster ID (UUID).

<a id="nestedblock--cluster_config"></a>
### Nested Schema for `cluster_config`

Required:

- `admin_email` (String) Email address for the cluster admin user. Changing this value will have no effect on the cluster.
- `admin_password` (String, Sensitive) Password for the cluster admin user. Changing this value will have no effect on the cluster.
- `bucket_name` (String) Name of the GCS bucket to use for the cluster. Changing this forces a new resource to be created.
- `cluster_name` (String) Unique name to assign to the cloud cluster.
- `dns_name_servers` (Set of String) DNS name servers for the cluster.
- `keep_cluster_on_failure` (Boolean) Whether to keep the cluster on failure (can be useful for troubleshooting). Changing this forces a new resource to be created.
- `ntp_servers` (Set of String) NTP servers for the cluster.
- `num_nodes` (Number) Number of nodes in the cluster. Must be at least 3 when `az_resilient` is true. Changing this forces a new resource to be created.

Optional:

- `dns_search_domains` (Set of String) DNS search domains for the cluster.
- `force_cluster_delete_on_destroy` (Boolean) Whether to force delete the cluster on destroy.
- `location` (String) Location for the cluster. This is free text, RSC will map it to the closest possible location e.g. Palo Alto, CA.
- `timezone` (String) Timezone for the cluster using IANA standard format e.g. America/Los_Angeles, Europe/Paris, etc.


<a id="nestedblock--vm_config"></a>
### Nested Schema for `vm_config`

Required:

- `cdm_version` (String) CDM version to use. Changing this forces a new resource to be created.
- `instance_type` (String) GCP instance type for the cluster nodes. Changing this forces a new resource to be created. Possible values are `N2_STANDARD_8`, `N2_STANDARD_16`, `N2_HIGHMEM_16`, `N2D_STANDARD_8`, `N2D_STANDARD_16` and `N2D_HIGHMEM_16`. The set of instance types actually available depends on the selected CDM version.
- `network` (String) GCP network name for the cluster nodes. Changing this forces a new resource to be created.
- `service_accounts` (Set of String) GCP service account emails for the cluster nodes. Changing this forces a new resource to be created.

Optional:

- `delete_protection` (Boolean) Whether to enable delete protection on the GCP instances. Changing this forces a new resource to be created.
- `host_project` (String) GCP host project for shared VPC. Changing this forces a new resource to be created.
- `subnet` (String) GCP subnet name for the cluster nodes. Required when `az_resilient` is false; omit it and use `subnet_az_config` when `az_resilient` is true. Changing this forces a new resource to be created.
- `subnet_az_config` (Block List) Subnet and availability zone pairs for Multi-AZ deployments. Required when `az_resilient` is true. Each block specifies a subnet and its availability zone; the network and host project are taken from the `network` and `host_project` fields. Changing this forces a new resource to be created. (see [below for nested schema](#nestedblock--vm_config--subnet_az_config))

Read-Only:

- `cdm_product` (String) CDM Product Code. This is a read-only field and computed based on the CDM version.

<a id="nestedblock--vm_config--subnet_az_config"></a>
### Nested Schema for `vm_config.subnet_az_config`

Required:

- `availability_zone` (String) Availability zone name, e.g. `us-west1-a`.
- `subnet` (String) GCP subnet name for this availability zone.


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
