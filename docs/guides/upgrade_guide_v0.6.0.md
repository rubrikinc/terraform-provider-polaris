---
page_title: "Upgrade Guide: v0.6.0"
---

# Upgrade Guide v0.6.0

## RSC provider changes
The v0.6.0 release introduces breaking changes to the following resources:
* `polaris_azure_exocompute`

## How to upgrade
Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v0.6.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 0.6.0"
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
This will read the remote state of the resources and migrate the local Terraform state to the v0.6.0 version.

## Upgrade issues
When upgrading to the v0.6.0 release you may encounter one or more of the following issues.

### polaris_azure_exocompute
To update the resource remove the `polaris_managed` argument. I.e. if the resource configuration looked like this:
```terraform
resource "polaris_azure_exocompute" "default" {
  subscription_id = polaris_azure_subscription.default.id
  polaris_managed = false
  region          = "eastus2"
  subnet          = "/subscriptions/9318aeec-d357-11eb-9b37-5f4e9f79db5d/.../subnets/default"
}
```

It should look like this after the manual update:
```terraform
resource "polaris_azure_exocompute" "default" {
  subscription_id = polaris_azure_subscription.default.id
  region          = "eastus2"
  subnet          = "/subscriptions/9318aeec-d357-11eb-9b37-5f4e9f79db5d/.../subnets/default"
}
```

The `polaris_managed` argument is optional and may not be set in your configuration.
