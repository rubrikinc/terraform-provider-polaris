---
page_title: "Manage Permissions"
---

# Manage Permissions
RSC requires permissions to operate and as new features are added to RSC the set of required permissions changes. This
guide explains how Terraform can be used to keep this set of permissions up to date.

## AWS
There are two ways to onboard AWS accounts to RSC, using a CloudFormation stack or not. Depending on the way an account
is onboarded, permissions are managed in different ways.

### Using a CloudFormation Stack
When an account is onboarded using a CloudFormation stack, the permissions are managed through the stack. When the
status of an account feature is `MISSING_PERMISSIONS` the CloudFormation stack must be updated for the RSC feature to
continue to function. This can be managed by setting the `permissions` argument to `update`.
```terraform
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
This will generate a diff when the status of at least one feature is in the `MISSING_PERMISSIONS` state. Applying the
account resource for this diff will update the CloudFormation stack. If the `permissions` argument is not specified the
provider will not attempt to update the CloudFormation stack.

### Not Using a CloudFormation Stack
When an account is onboarded without using a CloudFormation stack, the permissions can be managed using the
`polaris_aws_cnp_artifacts` and `polaris_aws_cnp_permissions` data sources and the
[aws](https://registry.terraform.io/providers/hashicorp/aws/latest) provider, using IAM roles. Please see the
[AWS CNP Account](aws_cnp_account.md) guide for more information on how create IAM roles using the data sources.

## Azure
For Azure permissions are managed through the subscription. When the status of a subscription feature is
`MISSING_PERMISSIONS` the permissions must be updated for the feature to continue to function. This can be managed by
Terraform using the [azurerm](https://registry.terraform.io/providers/hashicorp/azurerm/latest) provider:
```terraform
variable "features" {
  type        = set(string)
  description = "List of RSC features to enable for subscription."
}

data "polaris_azure_permissions" "features" {
  for_each = var.features
  feature  = each.key
}

resource "azurerm_role_definition" "subscription" {
  for_each = data.polaris_azure_permissions.features
  name     = "RSC - Subscription Level - ${each.value.feature}"
  scope    = data.azurerm_subscription.subscription.id

  permissions {
    actions          = each.value.subscription_actions
    data_actions     = each.value.subscription_data_actions
    not_actions      = each.value.subscription_not_actions
    not_data_actions = each.value.subscription_not_data_actions
  }
}

resource "azurerm_role_assignment" "subscription" {
  for_each           = data.polaris_azure_permissions.features
  principal_id       = "9e7f3952-1fc1-11ec-b57a-972144d12d97"
  role_definition_id = azurerm_role_definition.subscription[each.key].role_definition_resource_id
  scope              = data.azurerm_subscription.subscription.id
}

resource "azurerm_role_definition" "resource_group" {
  for_each = data.polaris_azure_permissions.features
  name     = "RSC - Resource Group Level - ${each.value.feature}"
  scope    = data.azurerm_resource_group.resource_group.id

  permissions {
    actions          = each.value.resource_group_actions
    data_actions     = each.value.resource_group_data_actions
    not_actions      = each.value.resource_group_not_actions
    not_data_actions = each.value.resource_group_not_data_actions
  }
}

resource "azurerm_role_assignment" "resource_group" {
  for_each           = data.polaris_azure_permissions.features
  principal_id       = "9e7f3952-1fc1-11ec-b57a-972144d12d97"
  role_definition_id = azurerm_role_definition.resource_group[each.key].role_definition_resource_id
  scope              = data.azurerm_resource_group.resource_group.id
}

resource "polaris_azure_service_principal" "service_principal" {
  ...
}

resource "polaris_azure_subscription" "subscription" {
  subscription_id   = data.azurerm_subscription.subscription.subscription_id
  subscription_name = data.azurerm_subscription.subscription.display_name
  tenant_domain     = polaris_azure_service_principal.service_principal.tenant_domain

  cloud_native_protection {
    permissions           = data.polaris_azure_permissions.features["CLOUD_NATIVE_PROTECTION"].id
    resource_group_name   = data.azurerm_resource_group.resource_group.name
    resource_group_region = data.azurerm_resource_group.resource_group.location
    regions               = ["eastus2"]
  }

  ...

  depends_on = [
    azurerm_role_definition.subscription,
    azurerm_role_definition.resource_group,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
role definitions and subscription resources. Applying the diff will first update the permissions of the role
definitions, then notify RSC about the update.

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
