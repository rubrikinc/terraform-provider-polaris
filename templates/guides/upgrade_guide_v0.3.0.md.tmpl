---
page_title: "Upgrade Guide: v0.3.0"
---

# Upgrade Guide v0.3.0

## RSC provider changes
The v0.3.0 release introduces breaking changes to the following resources:
* `polaris_aws_account`
* `polaris_azure_subscription`
* `polaris_azure_service_principal`
* `polaris_gcp_project`

## How to upgrade
Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v0.3.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 0.3.0"
    }
  }
}
```
Next, upgrade the Terraform provider to the new version by running:
```shell
% terraform init -upgrade
```
After the Terraform provider has been updated, validate the correctness of the Terraform configuration files by running:
```shell
% terraform plan
```
If this doesn't produce an error or unwanted diff, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v0.3.0 version.

## Upgrade issues
When upgrading to the v0.3.0 release you may encounter one or more of the following issues.

### polaris_aws_account
To update the resource add a new `cloud_native_protection` block. Then move the `regions` argument from the resource
into the new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```terraform
resource "polaris_aws_account" "default" {
  profile = "default"

  regions = [
    "us-east-2",
  ]
}
```

It should look like this after the manual update:
```terraform
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```

### polaris_azure_subscription
To update the resource add a new `cloud_native_protection` block. Then move the `regions` argument from the resource
into the new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```terraform
resource "polaris_azure_subscription" "default" {
  subscription_id = "1bb87eb6-2039-11ec-8a8a-3ba3fe58b590"
  tenant_domain   = "mydomain.onmicrosoft.com"

  regions = [
    "us-east-2",
  ]
}
```

It should look like this after the manual update:
```terraform
resource "polaris_azure_subscription" "default" {
  subscription_id = "1bb87eb6-2039-11ec-8a8a-3ba3fe58b590"
  tenant_domain   = "mydomain.onmicrosoft.com"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```

### polaris_azure_service_principal
To update the resource add a new `tenant_domain` argument. The value of this argument can be found in the credentials
file, as either `tenant_domain` or `tenantDomain`.

I.e. if the initial resource configuration looked like this:
```terraform
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
}
```

It should look like this after the manual update:
```terraform
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}
```

### polaris_gcp_project
To update the resource add a new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```terraform
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-bf80e97f8c4e.json"
}
```

It should look like this after the manual update:
```terraform
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-bf80e97f8c4e.json"

  cloud_native_protection {
  }
}
```
