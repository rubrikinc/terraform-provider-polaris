---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_azure_service_principal Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  The polaris_azure_service_principal resource adds an Azure service principal to
  RSC. A service principal must be added for each Azure tenant before subscriptions
  for the tenants can be added to RSC.
  There are 3 ways to create a polaris_azure_service principal resource:
  Using the app_id, app_name, app_secret, tenant_id and tenant_domain
  fields.Using the credentials field which is the path to a custom service principal
  file. A description of the custom format can be found
  here https://github.com/rubrikinc/rubrik-polaris-sdk-for-go?tab=readme-ov-file#azure-credentials.Using the sdk_auth field which is the path to an Azure service principal
  created with the Azure SDK using the --sdk-auth parameter.
  Prefer to use option 1, as the app_name and the app_secret can be updated
  without replacing the service principal.
  ~> Note: Removing the last subscription from an RSC tenant will automatically
  remove the tenant, which also removes the service principal. If this happens,
  the service principal can be replaced using
  terraform apply -replace=<address-of-service-principal>.
  ~> Note: Destroying the polaris_azure_service_principal resource only updates
  the local state, it does not remove the service principal from RSC. However,
  creating another polaris_azure_service_principal resource for the same Azure
  tenant will overwrite the old service principal in RSC.
  -> Note: There is no way to verify if a service principal has been added to RSC
  using the UI. RSC tenants don't show up in the UI until the first subscription is
  added.
---

# polaris_azure_service_principal (Resource)

The `polaris_azure_service_principal` resource adds an Azure service principal to
RSC. A service principal must be added for each Azure tenant before subscriptions
for the tenants can be added to RSC.

There are 3 ways to create a `polaris_azure_service principal` resource:
  1. Using the `app_id`, `app_name`, `app_secret`, `tenant_id` and `tenant_domain`
     fields.
  2. Using the `credentials` field which is the path to a custom service principal 
     file. A description of the custom format can be found
     [here](https://github.com/rubrikinc/rubrik-polaris-sdk-for-go?tab=readme-ov-file#azure-credentials).
  3. Using the `sdk_auth` field which is the path to an Azure service principal
     created with the Azure SDK using the `--sdk-auth` parameter.

Prefer to use option 1, as the `app_name` and the `app_secret` can be updated
without replacing the service principal.

~> **Note:** Removing the last subscription from an RSC tenant will automatically
   remove the tenant, which also removes the service principal. If this happens,
   the service principal can be replaced using
   `terraform apply -replace=<address-of-service-principal>`.

~> **Note:** Destroying the `polaris_azure_service_principal` resource only updates
   the local state, it does not remove the service principal from RSC. However,
   creating another `polaris_azure_service_principal` resource for the same Azure
   tenant will overwrite the old service principal in RSC.

-> **Note:** There is no way to verify if a service principal has been added to RSC
   using the UI. RSC tenants don't show up in the UI until the first subscription is
   added.

## Example Usage

```terraform
# With custom service principal file.
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}

# With a service principal created using the Azure SDK and the
# --sdk-auth parameter.
resource "polaris_azure_service_principal" "default" {
  sdk_auth      = "${path.module}/sdk-service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}

# Without a service principal file.
resource "polaris_azure_service_principal" "default" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My App"
  app_secret    = "<my-apps-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `tenant_domain` (String) Azure tenant primary domain. Changing this forces a new resource to be created.

### Optional

- `app_id` (String) Azure app registration application ID. Also known as the client ID. Changing this forces a new resource to be created.
- `app_name` (String) Azure app registration display name. Changing this forces a new resource to be created.
- `app_secret` (String, Sensitive) Azure app registration client secret. Changing this forces a new resource to be created.
- `credentials` (String) Path to a custom service principal file. Changing this forces a new resource to be created.
- `permissions` (String, Deprecated) Permissions updated signal. When this field is updated, the provider will notify RSC that permissions has been updated. Use this field with the `polaris_azure_permissions` data source. **Deprecated:** use the `polaris_azure_subscription` resource's `permissions` fields instead.
- `permissions_hash` (String, Deprecated) Permissions updated signal. **Deprecated:** use `permissions` instead.
- `sdk_auth` (String) Path to an Azure service principal created with the Azure SDK using the `--sdk-auth` parameter. Changing this forces a new resource to be created.
- `tenant_id` (String) Azure tenant ID. Also known as the directory ID. Changing this forces a new resource to be created.

### Read-Only

- `id` (String) Azure app registration application ID (UUID). Also known as the client ID. Note, this might change in the future, use the `app_id` field to reference the application ID in configurations.
