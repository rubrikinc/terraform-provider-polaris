---
page_title: "polaris_aws_cnp_permissions Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_aws_cnp_permissions` data source is used to access information about the
permissions required by RSC for a specified feature set.

## Permission Groups
Following is a list of features and their applicable permission groups. These are used
when specifying the feature set.

### CLOUD_NATIVE_ARCHIVAL
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

### CLOUD_NATIVE_PROTECTION
 * `BASIC` - Represents the basic set of permissions required to onboard the feature.
 * `EXPORT_AND_RESTORE` - Represents the set of permissions required for export and
   restore operations.
 * `FILE_LEVEL_RECOVERY` - Represents the set of permissions required for file-level
   recovery operations.
 * `SNAPSHOT_PRIVATE_ACCESS` - Represents the set of permissions required for private
   access to disk snapshots.

### CLOUD_NATIVE_S3_PROTECTION
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

### EXOCOMPUTE
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.
  * `PRIVATE_ENDPOINTS` - Represents the set of permissions required for usage of private
    endpoints.
  * `RSC_MANAGED_CLUSTER` - Represents the set of permissions required for the Rubrik-
    managed Exocompute cluster.

### RDS_PROTECTION
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

-> **Note:** When permission groups are specified, the `BASIC` permission group must
   always be included.

---

# polaris_aws_cnp_permissions (Data Source)


The `polaris_aws_cnp_permissions` data source is used to access information about the
permissions required by RSC for a specified feature set.

## Permission Groups
Following is a list of features and their applicable permission groups. These are used
when specifying the feature set.

### CLOUD_NATIVE_ARCHIVAL
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

### CLOUD_NATIVE_PROTECTION
 * `BASIC` - Represents the basic set of permissions required to onboard the feature.
 * `EXPORT_AND_RESTORE` - Represents the set of permissions required for export and
   restore operations.
 * `FILE_LEVEL_RECOVERY` - Represents the set of permissions required for file-level
   recovery operations.
 * `SNAPSHOT_PRIVATE_ACCESS` - Represents the set of permissions required for private
   access to disk snapshots.

### CLOUD_NATIVE_S3_PROTECTION
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

### EXOCOMPUTE
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.
  * `PRIVATE_ENDPOINTS` - Represents the set of permissions required for usage of private
    endpoints.
  * `RSC_MANAGED_CLUSTER` - Represents the set of permissions required for the Rubrik-
    managed Exocompute cluster.

### RDS_PROTECTION
  * `BASIC` - Represents the basic set of permissions required to onboard the feature.

-> **Note:** When permission groups are specified, the `BASIC` permission group must
   always be included.



## Example Usage

```terraform
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_ARCHIVAL"

    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
    ]
  }
}

# Lookup the required permissions using the output from the
# artifacts data source.
data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  cloud    = data.polaris_aws_cnp_artifacts.artifacts.cloud
  role_key = each.key

  dynamic "feature" {
    for_each = data.polaris_aws_cnp_artifacts.artifacts.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}
```


## Schema

### Required

- `feature` (Block Set, Min: 1) RSC feature with optional permission groups. (see [below for nested schema](#nestedblock--feature))
- `role_key` (String) Role key.

### Optional

- `cloud` (String) AWS cloud type. Possible values are `STANDARD`, `CHINA` and `GOV`. Default value is `STANDARD`.
- `ec2_recovery_role_path` (String) AWS EC2 recovery role path.

### Read-Only

- `customer_managed_policies` (List of Object) Customer managed policies. (see [below for nested schema](#nestedatt--customer_managed_policies))
- `id` (String) SHA-256 hash of the customer managed policies and the managed policies.
- `managed_policies` (List of String) Managed policies.

<a id="nestedblock--feature"></a>
### Nested Schema for `feature`

Required:

- `name` (String) RSC feature name. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_ARCHIVAL_ENCRYPTION`, `CLOUD_NATIVE_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, `EXOCOMPUTE` and `RDS_PROTECTION`.
- `permission_groups` (Set of String) RSC permission groups for the feature. Possible values are `BASIC`, `ENCRYPTION`, `EXPORT_AND_RESTORE`, `SNAPSHOT_PRIVATE_ACCESS`, `PRIVATE_ENDPOINT` and `RSC_MANAGED_CLUSTER`. Default value is `BASIC`.


<a id="nestedatt--customer_managed_policies"></a>
### Nested Schema for `customer_managed_policies`

Read-Only:

- `feature` (String) RSC feature name.
- `name` (String) Policy name.
- `policy` (String) AWS policy.
