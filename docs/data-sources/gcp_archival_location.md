---
page_title: "polaris_gcp_archival_location Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_gcp_archival_location` data source is used to access information
about a GCP archival location. An archival location is looked up using either
the ID or the name.

---

# polaris_gcp_archival_location (Data Source)


The `polaris_gcp_archival_location` data source is used to access information
about a GCP archival location. An archival location is looked up using either
the ID or the name.



## Example Usage

```terraform
# Using the ID.
data "polaris_gcp_archival_location" "location" {
  id = "9e90a8bb-0578-43dc-9330-57f86a9ae1e6"
}

# Using the name.
data "polaris_gcp_archival_location" "location" {
  name = "my-archival-location"
}
```


## Schema

### Optional

- `id` (String) Cloud native archival location ID (UUID).
- `name` (String) Name of the cloud native archival location.

### Read-Only

- `bucket_labels` (Map of String) GCP bucket labels.
- `bucket_prefix` (String) GCP bucket prefix. Note, `rubrik-` will always be prepended to the prefix.
- `cloud_account_id` (String) RSC cloud account ID (UUID).
- `connection_status` (String) Connection status of the archival location.
- `customer_managed_key` (Set of Object) Customer managed storage encryption. For `SPECIFIC_REGION`, a customer managed key for the specified region will be returned. For `SOURCE_REGION`, a customer managed key for each specified region will be returned, for other regions, data will be encrypted using platform managed keys. (see [below for nested schema](#nestedatt--customer_managed_key))
- `location_template` (String) RSC location template. If a region was specified, it will be `SPECIFIC_REGION`, otherwise `SOURCE_REGION`.
- `region` (String) GCP region to store the snapshots in (`SPECIFIC_REGION`). If not specified, the snapshots will be stored in the same region as the workload (`SOURCE_REGION`).
- `storage_class` (String) GCP bucket storage class. Possible values are `ARCHIVE`, `COLDLINE`, `NEARLINE`, `STANDARD` and `DURABLE_REDUCED_AVAILABILITY`.

<a id="nestedatt--customer_managed_key"></a>
### Nested Schema for `customer_managed_key`

Read-Only:

- `name` (String) Key name
- `region` (String) The region in which the key will be used.
- `ring_name` (String) Key ring name.
