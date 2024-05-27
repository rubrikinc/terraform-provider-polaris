---
page_title: "Upgrade Guide: v0.3.0"
---

# RSC provider version v0.3.0
v0.3.0 introduces breaking changes to the following resources:
* `polaris_aws_account`
* `polaris_azure_subscription`
* `polaris_azure_service_principal`
* `polaris_gcp_project`

After the Terraform configuration files have been updated according to the instructions in this guide and the version
number has been bumped, update the RSC Terraform provider by running:
```bash
$ terraform init -upgrade
```

Next, validate the correctness of the Terraform configuration files by running:
```bash
$ terraform plan
```

If this doesn't produce any error, proceed by running:
```bash
$ terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to v0.3.0.

## AWS
### polaris_aws_account
To update the resource add a new `cloud_native_protection` block. Then move the `regions` argument from the resource
into the new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```hcl
resource "polaris_aws_account" "default" {
  profile = "default"

  regions = [
    "us-east-2",
  ]
}
```

It should look like this after the manual update:
```hcl
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```

## Azure
### polaris_azure_subscription
To update the resource add a new `cloud_native_protection` block. Then move the `regions` argument from the resource
into the new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```hcl
resource "polaris_azure_subscription" "default" {
  subscription_id = "1bb87eb6-2039-11ec-8a8a-3ba3fe58b590"
  tenant_domain   = "mydomain.onmicrosoft.com"

  regions = [
    "us-east-2",
  ]
}
```

It should look like this after the manual update:
```hcl
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
```hcl
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
}
```

It should look like this after the manual update:
```hcl
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}
```

## GCP
### polaris_gcp_project
To update the resource add a new `cloud_native_protection` block.

I.e. if the initial resource configuration looked like this:
```hcl
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-bf80e97f8c4e.json"
}
```

It should look like this after the manual update:
```hcl
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-bf80e97f8c4e.json"

  cloud_native_protection {
  }
}
```
