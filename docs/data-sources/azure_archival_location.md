---
page_title: "polaris_azure_archival_location Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_azure_archival_location` data source is used to access information about
an Azure archival location. An archival location is looked up using either the ID or
the name.

---

# polaris_azure_archival_location (Data Source)


The `polaris_azure_archival_location` data source is used to access information about
an Azure archival location. An archival location is looked up using either the ID or
the name.



## Example Usage

```terraform
# Using the archival location ID.
data "polaris_azure_archival_location" "archival_location" {
  id = "db34f042-79ea-48b1-bab8-c40dfbf2ab82"
}

# Using the archival location name.
data "polaris_azure_archival_location" "archival_location" {
  name = "my-archival-location"
}
```


## Schema

### Optional

- `archival_location_id` (String, Deprecated) Cloud native archival location ID (UUID). **Deprecated:** use `id` instead.
- `id` (String) Cloud native archival location ID (UUID).
- `name` (String) Name of the cloud native archival location.

### Read-Only

- `connection_status` (String) Connection status of the cloud native archival location.
- `container_name` (String) Azure storage container name.
- `customer_managed_key` (Set of Object) Customer managed storage encryption. Specify the regions and their respective encryption details. For other regions, data will be encrypted using platform managed keys. (see [below for nested schema](#nestedatt--customer_managed_key))
- `location_template` (String) RSC location template. If a storage account region was specified, it will be `SPECIFIC_REGION`, otherwise `SOURCE_REGION`.
- `redundancy` (String) Azure storage redundancy. Possible values are `GRS`, `GZRS`, `LRS`, `RA_GRS`, `RA_GZRS` and `ZRS`. Default value is `LRS`.
- `storage_account_name_prefix` (String) Azure storage account name prefix. The storage account name prefix cannot be longer than 14 characters and can only consist of numbers and lower case letters.
- `storage_account_region` (String) Azure region to store the snapshots in. If not specified, the snapshots will be stored in the same region as the workload.
- `storage_account_tags` (Map of String) Azure storage account tags. Each tag will be added to the storage account created by RSC.
- `storage_tier` (String) Azure storage tier. Possible values are `COOL` and `HOT`. Default value is `COOL`.

<a id="nestedatt--customer_managed_key"></a>
### Nested Schema for `customer_managed_key`

Read-Only:

- `name` (String) Key name.
- `region` (String) The region in which the key will be used. Regions without customer managed keys will use platform managed keys.
- `vault_name` (String) Key vault name.
