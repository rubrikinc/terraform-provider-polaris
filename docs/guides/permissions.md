---
page_title: "Manage Permissions"
---

# Manage Permissions
RSC requires permissions to operate and as new features are added to RSC the set of required permissions changes. This
guide explains how Terraform can be used to keep this set of permissions up to date.

## AWS
For AWS this is managed through a CloudFormation stack. When the status of an account feature is `missing-permissions`
the CloudFormation stack must be updated for the feature to continue to function. This can be managed by setting the
`permissions` argument to `update`.
```hcl
resource "polaris_aws_account" "default" {
  profile     = "default"
  permissions = "update"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}
```
This will generate a diff when the status of at least one feature is `missing-permissions`. Applying the account
resource for this diff will update the CloudFormation stack. If the `permissions` argument is not specified the
provider will not attempt to update the CloudFormation stack.

## Azure
For Azure permissions are managed through a service principal. When the status of a subscription feature is
`missing-permissions` the permissions of the service principal must be updated for the feature to continue to
function. This can be managed by Terraform using the
[azurerm](https://registry.terraform.io/providers/hashicorp/azurerm/latest) provider:
```hcl
data "polaris_azure_permissions" "default" {
  features = [
    "cloud-native-protection",
    "exocompute",
  ]
}

resource "azurerm_role_definition" "default" {
  name  = "terraform"
  scope = data.azurerm_subscription.default.id

  permissions {
    actions          = data.polaris_azure_permissions.default.actions
    data_actions     = data.polaris_azure_permissions.default.data_actions
    not_actions      = data.polaris_azure_permissions.default.not_actions
    not_data_actions = data.polaris_azure_permissions.default.not_data_actions
  }
}

resource "azurerm_role_assignment" "default" {
  principal_id       = "9e7f3952-1fc1-11ec-b57a-972144d12d97"
  role_definition_id = azurerm_role_definition.default.role_definition_resource_id
  scope              = data.azurerm_subscription.default.id
}

resource "polaris_azure_service_principal" "default" {
  sdk_auth         = "${path.module}/sdk-service-principal.json"
  tenant_domain    = "mydomain.onmicrosoft.com"
  permissions_hash = data.polaris_azure_permissions.default.hash

  depends_on = [
    azurerm_role_definition.default,
    azurerm_role_assignment.default,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
role definition and service principal resources. Applying the diff will first update the permissions of the service
principal's role definition and then notify RSC about the update.

## GCP
For GCP permissions are managed through a service account. When the status of a project feature is `missing-permissions`
the permissions of the service account must be updated for the feature to continue to function. This can be managed by
Terraform using the [google](https://registry.terraform.io/providers/hashicorp/google/latest) provider.

### Project Service Account
When the service account is specified as part of the project resource:

```terraform
data "polaris_gcp_permissions" "default" {
  features = [
    "cloud-native-protection",
  ]
}

resource "google_project_iam_custom_role" "default" {
  role_id     = "terraform"
  title       = "Terraform"
  permissions = data.polaris_gcp_permissions.default.permissions
}

resource "google_project_iam_member" "default" {
  role   = google_project_iam_custom_role.default.id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"
}

resource "polaris_gcp_project" "default" {
  credentials      = "${path.module}//my-project-d978f94d6c4d.json"
  permissions_hash = data.polaris_gcp_permissions.default.hash

  cloud_native_protection {
  }

  depends_on = [
    google_project_iam_custom_role.default,
    google_project_iam_member.default,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
custom role and the project resources. Applying the diff will first update the permissions of the service account's
custom role and then notify RSC about the update.

### Default Service Account
When the service account is specified as part of the service account resource:
```terraform
data "polaris_gcp_permissions" "default" {
  features = [
    "cloud-native-protection",
  ]
}

resource "google_project_iam_custom_role" "default" {
  role_id     = "terraform"
  title       = "Terraform"
  permissions = data.polaris_gcp_permissions.default.permissions
}

resource "google_project_iam_member" "default" {
  role   = google_project_iam_custom_role.default.id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"
}

resource "polaris_gcp_service_account" "default" {
  credentials      = "${path.module}/my-project-d978f94d6c4d.json"
  permissions_hash = data.polaris_gcp_permissions.default.hash

  depends_on = [
    google_project_iam_custom_role.default,
    google_project_iam_member.default,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
custom role and the project resources. Applying the diff will first update the permissions of the service account's
custom role and then notify RSC about the update.
