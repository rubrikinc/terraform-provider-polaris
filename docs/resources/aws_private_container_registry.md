---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_aws_private_container_registry Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  The polaris_aws_private_container_registry resource enables the private container
  registry (PCR) feature for the RSC customer account. This disables the standard
  Rubrik container registry.
  ~> Note: Even though the polaris_aws_private_container_registry resource ID
  is an RSC cloud account ID, there can only be a single PCR per RSC customer
  account.
  Exocompute Image Bundles
  The following GraphQL query can be used to retrieve information about the image
  bundles used by RSC for exocompute:
  
  query ExotaskImageBundle($input: GetExotaskImageBundleInput) {
    exotaskImageBundle(input: $input) {
      bundleImages {
        name
        sha
        tag
      }
      bundleVersion
      eksVersion
      repoUrl
    }
  }
  
  The repoUrl field holds the URL to the RSC container registry from where the RSC
  images can be pulled.
  The input is an object with the following structure:
  
  {
    "input": {
      "eksVersion": "1.29"
    }
  }
  
  Where eksVersion is the version of the customer's' EKS cluster. eksVersion is
  optional, if it's not specified it defaults to the latest EKS version supported by
  RSC.
  The following GraphQL mutation can be used to set the approved bundle version for
  the RSC customer account:
  
  mutation SetBundleApprovalStatus($input: SetBundleApprovalStatusInput!) {
    setBundleApprovalStatus(input: $input)
  }
  
  The input is an object with the following structure:
  
  {
    "input": {
      "approvalStatus": "ACCEPTED",
      "bundleVersion": "1.164",
      "bundleMetadata": {
        "eksVersion": "1.29"
      }
    }
  }
  
  Where approvalStatus can be either ACCEPTED or REJECTED. bundleVersion is
  the the bundle version being approved or rejected. eksVersion is the version
  of the customer's EKS cluster.
---

# polaris_aws_private_container_registry (Resource)

The `polaris_aws_private_container_registry` resource enables the private container
registry (PCR) feature for the RSC customer account. This disables the standard
Rubrik container registry.

~> **Note:** Even though the `polaris_aws_private_container_registry` resource ID
   is an RSC cloud account ID, there can only be a single PCR per RSC customer
   account.

## Exocompute Image Bundles
The following GraphQL query can be used to retrieve information about the image
bundles used by RSC for exocompute:
```graphql
query ExotaskImageBundle($input: GetExotaskImageBundleInput) {
  exotaskImageBundle(input: $input) {
    bundleImages {
      name
      sha
      tag
    }
    bundleVersion
    eksVersion
    repoUrl
  }
}
```
The `repoUrl` field holds the URL to the RSC container registry from where the RSC
images can be pulled.

The input is an object with the following structure:
```json
{
  "input": {
    "eksVersion": "1.29"
  }
}
```
Where `eksVersion` is the version of the customer's' EKS cluster. `eksVersion` is
optional, if it's not specified it defaults to the latest EKS version supported by
RSC.

The following GraphQL mutation can be used to set the approved bundle version for
the RSC customer account:
```graphql
mutation SetBundleApprovalStatus($input: SetBundleApprovalStatusInput!) {
  setBundleApprovalStatus(input: $input)
}
```
The input is an object with the following structure:
```json
{
  "input": {
    "approvalStatus": "ACCEPTED",
    "bundleVersion": "1.164",
    "bundleMetadata": {
      "eksVersion": "1.29"
    }
  }
}
```
Where `approvalStatus` can be either `ACCEPTED` or `REJECTED`. `bundleVersion` is
the the bundle version being approved or rejected. `eksVersion` is the version
of the customer's EKS cluster.

## Example Usage

```terraform
resource "polaris_aws_private_container_registry" "registry" {
  account_id = polaris_aws_account.account.id
  native_id  = "123456789012"
  url        = "234567890121.dkr.ecr.us-east-2.amazonaws.com"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `account_id` (String) RSC cloud account ID (UUID) of the AWS account hosting the Exocompute. Changing this forces a new resource to be created.
- `native_id` (String) AWS account ID of the AWS account that will pull images from the RSC container registry.
- `url` (String) URL for customer provided private container registry.

### Read-Only

- `id` (String) RSC cloud account ID (UUID).
