---
page_title: "Manage Permissions"
---

# Manage Permissions
RSC requires permissions to operate and as new features are added to RSC the set of required permissions changes. This
guide explains how Terraform can be used to keep this set of permissions up to date.

## Permission Groups
Most RSC features split their permissions into named permission groups, so that only the capabilities that are actually
needed have to be granted. RSC follows a least-privilege model: a permission group should be enabled only when its
capability is required. For example, the `BASIC` group grants the permissions needed for routine protection, while the
`RECOVERY` group grants the elevated permissions needed to perform recoveries and should be enabled only on accounts
that perform them.

The permissions data sources, and the account, subscription and project resources, take a `permission_groups` argument
per feature, and it should always be set. The permissions granted are the union of the selected groups. Some of these
resources and data sources (`polaris_azure_subscription`, `polaris_azure_permissions` and `polaris_gcp_permissions`)
mark the argument optional, but this is only for backwards compatibility: RSC rejects a request that does not specify
a feature's permission groups, so omitting it causes the apply to fail.

For AWS and Azure the `polaris_aws_permission_groups` and `polaris_azure_permission_groups` data sources return the
groups available for a single feature, so a configuration can discover them at plan time:
```terraform
data "polaris_aws_permission_groups" "rds" {
  feature = "RDS_PROTECTION"
}

output "rds_permission_groups" {
  value = [for group in data.polaris_aws_permission_groups.rds.permission_groups : group.name]
}
```
GCP does not have a dedicated discovery data source. The groups selected for a GCP feature are reflected by the
`permission_groups` attribute of the `polaris_gcp_permissions` data source.

For each feature, set the `permission_groups` argument on its permissions data source to the groups that feature
requires. This requires the singular `feature` argument; the deprecated `features` argument cannot be combined with
`permission_groups`:
```terraform
data "polaris_azure_permissions" "cnp" {
  feature           = "CLOUD_NATIVE_PROTECTION"
  permission_groups = ["BASIC", "FILE_LEVEL_RECOVERY"]
}
```
The same `permission_groups` argument must be set on the feature blocks of the `polaris_aws_account`,
`polaris_aws_cnp_account`, `polaris_azure_subscription` and `polaris_gcp_project` resources, and should match the
groups requested from the corresponding permissions data source. See the
[AWS CNP Account](aws_cnp_account.md) guide for a complete example that threads permission groups through
the AWS data sources and resources.

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
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }
}
```
This will generate a diff when the status of at least one feature is in the `MISSING_PERMISSIONS` state. Applying the
account resource for this diff will update the CloudFormation stack. If the `permissions` argument is not specified the
provider will not attempt to update the CloudFormation stack.

Each feature block, such as `cloud_native_protection`, requires a `permission_groups` argument that selects the
permissions granted through the stack, as described in the Permission Groups section above.

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
data "polaris_azure_permissions" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"

  permission_groups = [
    "BASIC",
  ]
}

resource "azurerm_role_definition" "subscription" {
  name  = "RSC - Subscription Level - CLOUD_NATIVE_PROTECTION"
  scope = data.azurerm_subscription.subscription.id

  permissions {
    actions          = data.polaris_azure_permissions.cnp.subscription_actions
    data_actions     = data.polaris_azure_permissions.cnp.subscription_data_actions
    not_actions      = data.polaris_azure_permissions.cnp.subscription_not_actions
    not_data_actions = data.polaris_azure_permissions.cnp.subscription_not_data_actions
  }
}

resource "azurerm_role_assignment" "subscription" {
  principal_id       = "9e7f3952-1fc1-11ec-b57a-972144d12d97"
  role_definition_id = azurerm_role_definition.subscription.role_definition_resource_id
  scope              = data.azurerm_subscription.subscription.id
}

resource "azurerm_role_definition" "resource_group" {
  name  = "RSC - Resource Group Level - CLOUD_NATIVE_PROTECTION"
  scope = data.azurerm_resource_group.resource_group.id

  permissions {
    actions          = data.polaris_azure_permissions.cnp.resource_group_actions
    data_actions     = data.polaris_azure_permissions.cnp.resource_group_data_actions
    not_actions      = data.polaris_azure_permissions.cnp.resource_group_not_actions
    not_data_actions = data.polaris_azure_permissions.cnp.resource_group_not_data_actions
  }
}

resource "azurerm_role_assignment" "resource_group" {
  principal_id       = "9e7f3952-1fc1-11ec-b57a-972144d12d97"
  role_definition_id = azurerm_role_definition.resource_group.role_definition_resource_id
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
    permissions           = data.polaris_azure_permissions.cnp.id
    permission_groups     = ["BASIC"]
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
definitions, then notify RSC about the update. The `permission_groups` for each feature must be set on both the
`polaris_azure_permissions` data source and the matching `polaris_azure_subscription` feature block, as described in the
Permission Groups section above.

## GCP
For GCP permissions are managed through a service account. When the status of a project feature is `missing-permissions`
the permissions of the service account must be updated for the feature to continue to function. This can be managed by
Terraform using the [google](https://registry.terraform.io/providers/hashicorp/google/latest) provider.

Set the `permission_groups` for each feature using the singular `feature` argument on the `polaris_gcp_permissions` data
source, and the matching `feature` block on the `polaris_gcp_project` resource. The deprecated `features` argument and
the deprecated `cloud_native_protection` block cannot carry permission groups, so use the `feature` form throughout, as
described in the Permission Groups section above.

The `polaris_gcp_permissions` data source returns the required permissions split into `without_conditions` and
`with_conditions`. Grant the `without_conditions` permissions through a custom role bound directly to the service
account. Grant the `with_conditions` permissions through a separate custom role bound with an IAM condition whose
expression is built from the `conditions` attribute. The conditional role is only needed when the selected features
have permissions with conditions.

### Project Service Account
When the service account is specified as part of the project resource:

```terraform
data "polaris_gcp_permissions" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"

  permission_groups = [
    "BASIC",
  ]
}

# Permissions without conditions are granted through a custom role bound
# directly to the service account.
resource "google_project_iam_custom_role" "without_conditions" {
  role_id     = "terraform"
  title       = "Terraform"
  permissions = data.polaris_gcp_permissions.cnp.without_conditions
}

resource "google_project_iam_member" "without_conditions" {
  role   = google_project_iam_custom_role.without_conditions.id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"
}

# Permissions with conditions are granted through a separate custom role bound
# with an IAM condition. Created only when the feature has such permissions.
resource "google_project_iam_custom_role" "with_conditions" {
  count = length(data.polaris_gcp_permissions.cnp.with_conditions) > 0 ? 1 : 0

  role_id     = "terraform_with_conditions"
  title       = "Terraform With Conditions"
  permissions = data.polaris_gcp_permissions.cnp.with_conditions
}

resource "google_project_iam_member" "with_conditions" {
  count = length(data.polaris_gcp_permissions.cnp.with_conditions) > 0 ? 1 : 0

  role   = google_project_iam_custom_role.with_conditions[0].id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"

  condition {
    title      = "Rubrik Condition"
    expression = join(" || ", data.polaris_gcp_permissions.cnp.conditions)
  }
}

resource "polaris_gcp_project" "default" {
  credentials = "${path.module}//my-project-d978f94d6c4d.json"

  feature {
    name              = "CLOUD_NATIVE_PROTECTION"
    permission_groups = ["BASIC"]
    permissions       = data.polaris_gcp_permissions.cnp.id
  }

  depends_on = [
    google_project_iam_member.without_conditions,
    google_project_iam_member.with_conditions,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
custom role and the project resources. Applying the diff will first update the permissions of the service account's
custom role and then notify RSC about the update.

### Default Service Account
When the service account is specified as part of the service account resource:
```terraform
data "polaris_gcp_permissions" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"

  permission_groups = [
    "BASIC",
  ]
}

# Permissions without conditions are granted through a custom role bound
# directly to the service account.
resource "google_project_iam_custom_role" "without_conditions" {
  role_id     = "terraform"
  title       = "Terraform"
  permissions = data.polaris_gcp_permissions.cnp.without_conditions
}

resource "google_project_iam_member" "without_conditions" {
  role   = google_project_iam_custom_role.without_conditions.id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"
}

# Permissions with conditions are granted through a separate custom role bound
# with an IAM condition. Created only when the feature has such permissions.
resource "google_project_iam_custom_role" "with_conditions" {
  count = length(data.polaris_gcp_permissions.cnp.with_conditions) > 0 ? 1 : 0

  role_id     = "terraform_with_conditions"
  title       = "Terraform With Conditions"
  permissions = data.polaris_gcp_permissions.cnp.with_conditions
}

resource "google_project_iam_member" "with_conditions" {
  count = length(data.polaris_gcp_permissions.cnp.with_conditions) > 0 ? 1 : 0

  role   = google_project_iam_custom_role.with_conditions[0].id
  member = "serviceAccount:terraform@my-project.iam.gserviceaccount.com"

  condition {
    title      = "Rubrik Condition"
    expression = join(" || ", data.polaris_gcp_permissions.cnp.conditions)
  }
}

resource "polaris_gcp_service_account" "default" {
  credentials = "${path.module}/my-project-d978f94d6c4d.json"
}

resource "polaris_gcp_project" "default" {
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012

  feature {
    name              = "CLOUD_NATIVE_PROTECTION"
    permission_groups = ["BASIC"]
    permissions       = data.polaris_gcp_permissions.cnp.id
  }

  depends_on = [
    google_project_iam_member.without_conditions,
    google_project_iam_member.with_conditions,
    polaris_gcp_service_account.default,
  ]
}
```
When the permissions for a feature changes the permissions data source will reflect this generating a diff for the
custom role and the project resources. Applying the diff will first update the permissions of the service account's
custom role and then notify RSC about the update.
