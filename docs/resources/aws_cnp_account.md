---
page_title: "polaris_aws_cnp_account Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_aws_cnp_account` resource adds an AWS account to RSC. To grant RSC
permissions to perform certain operations on the account, IAM roles needs to be
created and communicated to RSC using the `polaris_aws_cnp_attachment` resource.
The roles and permissions needed by RSC can be looked up using the
`polaris_aws_cnp_artifact` and `polaris_aws_cnp_permissions` data sources.

The `CLOUD_DISCOVERY` feature enables RSC to discover resources in the AWS
account without enabling protection. It is currently optional but will become
required when onboarding protection features. Once onboarded, it cannot be
removed unless all protection features are removed first.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

`CLOUD_DISCOVERY`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_ARCHIVAL`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_DYNAMODB_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_S3_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`EXOCOMPUTE`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.
  * `RSC_MANAGED_CLUSTER` - Represents the set of permissions required for the
    Rubrik-managed Exocompute cluster.

`RDS_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`KUBERNETES_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`SERVERS_AND_APPS`
  * `CLOUD_CLUSTER_ES` - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the `BASIC` permission group
   is always required except for the `SERVERS_AND_APPS` feature.

-> **Note:** To onboard an account using a CloudFormation stack instead of IAM
   roles, use the `polaris_aws_account` resource.

---

# polaris_aws_cnp_account (Resource)


The `polaris_aws_cnp_account` resource adds an AWS account to RSC. To grant RSC
permissions to perform certain operations on the account, IAM roles needs to be
created and communicated to RSC using the `polaris_aws_cnp_attachment` resource.
The roles and permissions needed by RSC can be looked up using the
`polaris_aws_cnp_artifact` and `polaris_aws_cnp_permissions` data sources.

The `CLOUD_DISCOVERY` feature enables RSC to discover resources in the AWS
account without enabling protection. It is currently optional but will become
required when onboarding protection features. Once onboarded, it cannot be
removed unless all protection features are removed first.

## Permission Groups
Following is a list of features and their applicable permission groups. These
are used when specifying the feature set.

`CLOUD_DISCOVERY`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_ARCHIVAL`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_DYNAMODB_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`CLOUD_NATIVE_S3_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`EXOCOMPUTE`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.
  * `RSC_MANAGED_CLUSTER` - Represents the set of permissions required for the
    Rubrik-managed Exocompute cluster.

`RDS_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`KUBERNETES_PROTECTION`
  * `BASIC` - Represents the basic set of permissions required to onboard the
    feature.

`SERVERS_AND_APPS`
  * `CLOUD_CLUSTER_ES` - Represents the basic set of permissions required to onboard the
    feature.

-> **Note:** When permission groups are specified, the `BASIC` permission group
   is always required except for the `SERVERS_AND_APPS` feature.

-> **Note:** To onboard an account using a CloudFormation stack instead of IAM
   roles, use the `polaris_aws_account` resource.



## Example Usage

```terraform
# Using hardcoded values.
resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"

  regions = [
    "us-east-2",
    "us-west-2",
  ]

  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "EXOCOMPUTE"
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]
  }
}

# Using variables for the account values and the features. The dynamic
# feature block could also be expanded from the polaris_aws_cnp_artifacts
# data source.
variable "name" {
  type        = string
  description = "AWS account name."
}

variable "native_id" {
  type        = string
  description = "AWS account ID."
}

variable "regions" {
  type        = set(string)
  description = "AWS regions to protect."
}

variable "features" {
  type = map(object({
    permission_groups = set(string)
  }))
  description = "RSC features with permission groups."
}

resource "polaris_aws_cnp_account" "account" {
  name      = var.name
  native_id = var.native_id
  regions   = var.regions

  dynamic "feature" {
    for_each = var.features
    content {
      name              = feature.key
      permission_groups = feature.value["permission_groups"]
    }
  }
}
```


## Schema

### Required

- `feature` (Block Set, Min: 1) RSC feature with permission groups. (see [below for nested schema](#nestedblock--feature))
- `native_id` (String) AWS account ID. Changing this forces a new resource to be created.
- `regions` (Set of String) Regions.

### Optional

- `cloud` (String) AWS cloud type. Possible values are `STANDARD`, `CHINA` and `GOV`. Default value is `STANDARD`. Changing this forces a new resource to be created.
- `delete_snapshots_on_destroy` (Boolean) Should snapshots be deleted when the resource is destroyed.
- `external_id` (String) External ID. Changing this forces a new resource to be created.
- `name` (String) Account name.

### Read-Only

- `id` (String) RSC cloud account ID (UUID).
- `trust_policies` (Set of Object) AWS IAM trust policies required by RSC. The `policy` field should be used with the `assume_role_policy` of the `aws_iam_role` resource. (see [below for nested schema](#nestedatt--trust_policies))

<a id="nestedblock--feature"></a>
### Nested Schema for `feature`

Required:

- `name` (String) RSC feature name. Possible values are `CLOUD_DISCOVERY`, `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, `CLOUD_NATIVE_DYNAMODB_PROTECTION`, `KUBERNETES_PROTECTION`, `SERVERS_AND_APPS`, `EXOCOMPUTE` and `RDS_PROTECTION`.
- `permission_groups` (Set of String) RSC permission groups for the feature. Possible values are `BASIC`, `CLOUD_CLUSTER_ES` and `RSC_MANAGED_CLUSTER`. For backwards compatibility, `[]` is interpreted as all applicable permission groups.

<a id="nestedatt--trust_policies"></a>
### Nested Schema for `trust_policies`

Read-Only:

- `policy` (String) RSC artifact key for the AWS role.
- `role_key` (String) AWS IAM trust policy.

## Import

If an `external_id` was specified when the account was onboarded, it must also be specified as part of the import ID.
This is done by appending the external ID to the account ID. E.g, to import an account onboarded with `external_id` set
to `ExternalID`:
```text
f503742e-0a15-4a53-8579-54c2f978e49d-ExternalID
```

If the wrong external ID is specified, the import will fail with an error similar to:
```text
Error: failed to get trust policies: Already a value is registered as an external id.
```

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_aws_cnp_account.account
  id = "3553bc74-7061-40e3-bac5-d2639e58bb7e-external-id"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_aws_cnp_account.account 3553bc74-7061-40e3-bac5-d2639e58bb7e-external-id
```

