---
page_title: "Upgrade Guide: v0.6.0 "
subcategory: "Upgrade"
---

# RSC provider version v0.6.0
v0.6.0 introduces breaking changes to the following resources:
* `polaris_azure_exocompute`

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
This will read the remote state of the resources and migrate the local Terraform state to v0.6.0.

## polaris_azure_exocompute
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
