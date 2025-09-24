---
page_title: "polaris_aws_cnp_account_trust_policy Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `aws_cnp_account_trust_policy` resource gets the AWS IAM trust policies
required by RSC. The `policy` field of `aws_cnp_account_trust_policy` resource
should be used with the `assume_role_policy` of the `aws_iam_role` resource.

~> **Note:** The `polaris_aws_cnp_account` resource can now be used to get the
   IAM trust policies for all role keys. The `aws_cnp_account_trust_policy`
   resource is no longer required and will be deprecated in a future version.

~> **Note:** Once `external_id` has been set it cannot be changed. Unless the
   cloud account is removed and onboarded again.

-> **Note:** The `features` field takes only the feature names and not the
   permission groups associated with the features.

---

# polaris_aws_cnp_account_trust_policy (Resource)


The `aws_cnp_account_trust_policy` resource gets the AWS IAM trust policies
required by RSC. The `policy` field of `aws_cnp_account_trust_policy` resource
should be used with the `assume_role_policy` of the `aws_iam_role` resource.

~> **Note:** The `polaris_aws_cnp_account` resource can now be used to get the
   IAM trust policies for all role keys. The `aws_cnp_account_trust_policy`
   resource is no longer required and will be deprecated in a future version.

~> **Note:** Once `external_id` has been set it cannot be changed. Unless the
   cloud account is removed and onboarded again.

-> **Note:** The `features` field takes only the feature names and not the
   permission groups associated with the features.



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

resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"
  regions = [
    "us-east-2",
    "us-west-2",
  ]

  dynamic "feature" {
    for_each = data.polaris_aws_cnp_artifacts.artifacts.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}

# Lookup the trust policies using the artifacts data source and the
# account resource.
resource "polaris_aws_cnp_account_trust_policy" "trust_policy" {
  for_each   = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  account_id = polaris_aws_cnp_account.account.id
  role_key   = each.key
}
```


## Schema

### Required

- `account_id` (String) RSC cloud account ID (UUID). Changing this forces a new resource to be created.
- `role_key` (String) RSC artifact key for the AWS role. Possible values are `CROSSACCOUNT`, `EXOCOMPUTE_EKS_MASTERNODE` and `EXOCOMPUTE_EKS_WORKERNODE`. Changing this forces a new resource to be created.

### Optional

- `external_id` (String) Trust policy external ID. If not specified, RSC will generate an external ID. Note, once the external ID has been set it cannot be changed. Changing this forces a new resource to be created.
- `features` (Set of String, Deprecated) RSC features. Possible values are `CLOUD_NATIVE_ARCHIVAL`, `CLOUD_NATIVE_PROTECTION`, `CLOUD_NATIVE_S3_PROTECTION`, `SERVERS_AND_APPS`, `EXOCOMPUTE` and `RDS_PROTECTION`. **Deprecated:** no longer used by the provider, any value set is ignored.

### Read-Only

- `id` (String) RSC cloud account ID (UUID) with the role key as a prefix.
- `policy` (String) AWS IAM trust policy.

## Import

When importing a trust policy both the `role_key` and the `account_id` must be specified as part of the import ID. E.g:
```text
CROSSACCOUNT-f503742e-0a15-4a53-8579-54c2f978e49d
```

If an `external_id` was specified when the account was onboarded, it must also be specified as part of the import ID.
This is done by appending the external ID to the account ID. E.g, to import an account onboarded with `external_id` set
to `ExternalID`:
```text
CROSSACCOUNT-f503742e-0a15-4a53-8579-54c2f978e49d-ExternalID
```

If the wrong external ID is specified, the import will fail with an error similar to:
```text
Error: failed to get trust policies: Already a value is registered as an external id.
```

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_aws_cnp_account_trust_policy.trust_policy
  id = "CROSSACCOUNT-acfd7b71-6259-45bc-b0c6-f067918c5cc7"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_aws_cnp_account_trust_policy.trust_policy CROSSACCOUNT-acfd7b71-6259-45bc-b0c6-f067918c5cc7
```

