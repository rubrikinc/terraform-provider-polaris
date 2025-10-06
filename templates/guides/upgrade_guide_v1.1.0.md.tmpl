---
page_title: "Upgrade Guide: v1.1.0"
---

# Upgrade Guide v1.1.0

## Before Upgrading
Review the [changelog](changelog.md) to understand what has changed and what might cause an issue when upgrading the
provider. Note, deprecated resources and fields will be removed in a future release, please migrate your configurations
to use the recommended replacements as soon as possible.

## How to Upgrade
Make sure that the `version` field is configured in a way which allows Terraform to upgrade to the v1.1.0 release. One
way of doing this is by using the pessimistic constraint operator `~>`, which allows Terraform to upgrade to the latest
release within the same minor version:
```terraform
terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "~> 1.1.0"
    }
  }
}
```
Next, upgrade the provider to the new version by running:
```shell
% terraform init -upgrade
```
After the provider has been updated, validate the correctness of the Terraform configuration files by running:
```shell
% terraform plan
```
If you get an error or an unwanted diff, please see the _Significant Changes and New Features_ below for additional
instructions. Otherwise, proceed by running:
```shell
% terraform apply -refresh-only
```
This will read the remote state of the resources and migrate the local Terraform state to the v1.1.0 version.

## New Features

### Custom Tags
Support for custom tags has been added for AWS and Azure. The `polaris_aws_custom_tags` and `polaris_azure_custom_tags`
resources are used to manage the custom tags. Custom tags will be applied to all resources created in the cloud account
by RSC. Here's a simple example, showing how to add two custom AWS tags:
```terraform
resource "polaris_aws_custom_tags" "tags" {
  custom_tags = {
    "app"    = "RSC"
    "vendor" = "Rubrik"
  }
}
```

### SLA Domain Assignments and Tag Rules
Support for assigning SLA domains to workloads has been added. The `polaris_sla_domain_assignment` resource is used to
manage the assignment of an SLA domain to workloads. In addition, the `polaris_tag_rule` resource can be used to match
workloads based on tags to simplify the selection of workloads to protect. Here's a simple example, showing how to
assign the Bronze level SLA domain to all current and future Azure virtual machines in a specific cloud account tagged
with a certain tag:
```terraform
data "polaris_azure_subscription" "subscription" {
  name = "subscription"
}

data "polaris_sla_domain" "bronze" {
  name = "bronze"
}

resource "polaris_tag_rule" "rule" {
  name        = "azure-virtual-machines"
  object_type = "AZURE_VIRTUAL_MACHINE"
  tag_key     = "protect"
  tag_value   = "yes"

  cloud_account_ids = [
    data.polaris_azure_subscription.subscription.id,
  ]
}

resource "polaris_sla_domain_assignment" "assignment" {
  sla_domain_id = data.polaris_sla_domain.bronze.id

  object_ids = [
    polaris_tag_rule.rule.id,
  ]
}
```

### SSO Groups
Support for assigning roles to SSO groups has been added. The existing `polaris_role_assignment` resource is used to
manage the assignment of roles to SSO groups using the new `sso_group_id` field. Here's a simple example, showing how to
assign the administrator role to an SSO group:
```terraform
data "polaris_role" "admin" {
  name = "administrator"
}

data "polaris_sso_group" "group" {
  name = "group"
}

resource "polaris_role_assignment" "assignment" {
  sso_group_id = data.polaris_sso_group.group.id

  role_ids = [
    data.polaris_role.admin.id,
  ]
}
```

### CCES Registration
Support for automatically registering a CCES clusters with RSC has been added. The `polaris_cdm_registration` resource
is used to register a bootstrapped cluster with RSC. Note, the resource can only be used to register the cluster, it
cannot manage the registration. Here's a simple example, showing how to register a cluster with RSC:
```terraform
variable "admin_password" {
  description = "Password for the Rubrik Cloud Cluster admin account."
  type        = string
  sensitive   = true
}

resource "polaris_cdm_registration" "registration" {
  admin_password          = var.admin_password
  cluster_name            = "cluster"
  cluster_node_ip_address = "10.0.100.101"
}
```

## Significant Changes

### Azure SQL DB Protection Resource Group
The Azure SQL DB Protection RSC feature has been updated to use a resource group. To support this in the provider, the
`sql_db_protection` field of the `polaris_azure_subscription` resource has been updated to include the
`resource_group_name`, `resource_group_region` and `resource_group_tags` optional fields. The resource group specified
using the fields must already exist in Azure. This update is being rolled out over time to RSC accounts. When the update
is rolled out to an RSC account a diff similar to this will be seen:
```console
# azurerm_role_definition.resource_group["AZURE_SQL_DB_PROTECTION"] will be updated in-place
~ resource "azurerm_role_definition" "resource_group" {
    id                          = "/subscriptions/e64456f3-7e4f-4aa4-9e6d-097e552ddf42/providers/Microsoft.Authorization/roleDefinitions/b2243415-7c8c-f023-8e94-745666937a2f|/subscriptions/e64456f3-7e4f-4aa4-9e6d-097e552ddf42/resourceGroups/ja-sqldb-test-rg"
    name                        = "Terraform - Azure Permissions Example Resource Group Level - AZURE_SQL_DB_PROTECTION"
    # (5 unchanged attributes hidden)

  + permissions {
      + actions     = [
          + "Microsoft.Sql/servers/databases/delete",
          + "Microsoft.Sql/servers/delete",
          + "Microsoft.Sql/servers/firewallRules/read",
          + "Microsoft.Sql/servers/firewallRules/write",
          + "Microsoft.Sql/servers/privateEndpointConnectionsApproval/action",
          + "Microsoft.Sql/servers/write",
        ]
      + not_actions = []
    }
}

# azurerm_role_definition.subscription["AZURE_SQL_DB_PROTECTION"] will be updated in-place
~ resource "azurerm_role_definition" "subscription" {
    id                          = "/subscriptions/e64456f3-7e4f-4aa4-9e6d-097e552ddf42/providers/Microsoft.Authorization/roleDefinitions/4d1ac739-a1ff-7586-62c0-a76406dacb29|/subscriptions/e64456f3-7e4f-4aa4-9e6d-097e552ddf42"
    name                        = "Terraform - Azure Permissions Example Subscription Level - AZURE_SQL_DB_PROTECTION"
    # (5 unchanged attributes hidden)

  ~ permissions {
      ~ actions          = [
          - "Microsoft.Logic/workflows/read",
          - "Microsoft.Logic/workflows/runs/read",
          - "Microsoft.Logic/workflows/triggers/run/action",
            "Microsoft.Resources/subscriptions/resourceGroups/read",
            # (21 unchanged elements hidden)
            "Microsoft.Sql/servers/read",
          - "Microsoft.Web/connections/read",
        ]
        # (3 unchanged attributes hidden)
    }
}

# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
    id                          = "daa810ba-f749-49a2-9fcf-3e26c382c979"
    # (4 unchanged attributes hidden)

  ~ sql_db_protection {
      ~ permissions       = "eb400595a152970804b87af2e0615e81fc4b945460be92b510fe556124d8c194" -> "8a3f91c8d159e409224efec447fce29812496b1f5dbefb06a94da415d1820fe3"
        # (3 unchanged attributes hidden)
    }
}
```
Terraform wants to add permissions to the resource group level in Azure, remove permission from the subscription level
in Azure and update the subscription in RSC. Before applying this diff, upgrade the provider to version `1.1.0` and add
the `resource_group_name` and `resource_group_region` fields to the `polaris_azure_subscription` resource. The
`resource_group_tags` field is optional. Note, when the resource group has been set, it cannot be changed unless the
feature is re-onboarded.

Applying the diff without upgrading the provider or providing a resource group name and region will result in an error
similar to this:
```text
polaris_azure_subscription.subscription: Modifying... [id=daa810ba-f749-49a2-9fcf-3e26c382c979]
╷
│ Error: failed to update permissions: failed to request upgradeAzureCloudAccountPermissionsWithoutOauthWithPermissionGroups: graphql response body is an error (status code 200): UNKNOWN: Error RBK30300021: Resource group input not given for feature AZURE_SQL_DB_PROTECTION for subscription ID e64456f3-7e4f-4aa4-9e6d-097e552ddf42. Possible cause: Resource group input not given for feature AZURE_SQL_DB_PROTECTION for subscription ID e64456f3-7e4f-4aa4-9e6d-097e552ddf42. Possible remedy: Provide resource group input for feature AZURE_SQL_DB_PROTECTION for subscription ID e64456f3-7e4f-4aa4-9e6d-097e552ddf42. (code: 500, traceId: zBsFyulf40YlgfHtA5qtJw==)
│
│   with polaris_azure_subscription.subscription,
│   on azure_sql_db.tf line 150, in resource "polaris_azure_subscription" "subscription":
│  150: resource "polaris_azure_subscription" "subscription" {
│
╵
```
Note, even though the Terraform apply failed, the `permissions` field of the `polaris_azure_subscription` resource has
been updated and any plans following will not show the diff, even though the Azure SQL DB Protection feature has not
been properly updated. To force an update of the subscription, temporarily change the `permissions` field of the
`sql_db_protection` field of the `polaris_azure_subscription` resource to some string value, e.g:
```terraform
resource "polaris_azure_subscription" "subscription" {
  # ...
  sql_db_protection {
    # ...
    permissions           = "force-update"
    resource_group_name   = "<resource-group-name>"
    resource_group_region = "<resource-group-region>"
    # ...
  }
}
```
Apply the configuration and change the value back and re-apply. This will trigger an update of the feature.

If the Azure SQL DB Protection feature has already been updated using the RSC UI, a diff similar to this will be seen:
```console
# polaris_azure_subscription.subscription will be updated in-place
~ resource "polaris_azure_subscription" "subscription" {
    id                          = "daa810ba-f749-49a2-9fcf-3e26c382c979"
    # (4 unchanged attributes hidden)

  ~ sql_db_protection {
      - resource_group_name   = "ja-sqldb-test-rg" -> null
      - resource_group_region = "eastus2" -> null
        # (5 unchanged attributes hidden)
    }
}
```
Where `resource_group_name` and `resource_group_region` are the values specified in the RSC UI when the feature was
updated. To resolve the diff, copy the values from the diff and add them to the configuration for the
`sql_db_protection` field of the `polaris_azure_subscription` resource.

### User Changes
The `id` field of the `polaris_user` resource has changed, it now holds the user ID instead of the user email address.
This is a breaking change if a configuration expects the `id` field to be an email address. To work around this issue,
use the `email` field of the `polaris_user` instead of the `id` field.

When creating a user, RSC automatically convert all letters in the user's email address to lower case, this will cause a
diff to be generated if the email address in the configuration is specified using upper case letters. The `email` field
of the `polaris_user` resource now validates that all letters of the email address is lower case. If an email address
contains upper case letters, Terraform will report an error similar to this:
```text
╷
│ Error: invalid value for email (letters must be lower case)
│
│   with polaris_user.user,
│   on main.tf line 6, in resource "polaris_user" "user":
│    6:   email = "User@example.org"
│
╵
```

A new data source has been added which allows a user to be looked up by the user's email address. The `polaris_user`
data source can be used to look up a user by the user's email address.

### Role Assignment Changes
The `id` field of the `polaris_role_assignment` resource has changed, it now holds the user ID or the SSO group ID
instead of the hash of the user email address and the role ID. This is not expected to be a breaking change since the
old value was a hash.

The `role_id` and `user_email` fields of the `polaris_role_assignment` resource have been deprecated. The `role_id`
field has been replaced with the `role_ids` field and the `user_email` field has been replaced with the `user_id` field.

To get the user ID for a role assignment of a user managed outside of Terraform, use the new `polaris_user` data source:
```terraform
data "polaris_role" "admin" {
  name = "administrator"
}

data "polaris_user" "user" {
  email = "user@example.org"
}

resource "polaris_role_assignment" "assignment" {
  user_id = data.polaris_user.user.id

  role_ids = [
    data.polaris_role.role.id,
  ]
}
```

To get the SSO group ID for a role assignment of an SSO group, use the new `polaris_sso_group` data source:
```terraform
data "polaris_role" "admin" {
  name = "administrator"
}

data "polaris_sso_group" "group" {
  name = "group"
}

resource "polaris_role_assignment" "assignment" {
  sso_group_id = data.polaris_sso_group.group.id

  role_ids = [
    data.polaris_role.role.id,
  ]
}
```

### Azure Service Principal Changes
The behavior of the `sdk_auth` field of the `polaris_azure_service_principal` resource has changed. The Azure app name
is no longer looked up using the Azure AD Graph API. Instead, the app name is generated in a consistent way using the
Azure app and tenant IDs. This change is due to the deprecation of the Azure AD Graph API by Microsoft.

### CDM Bootstrap Backwards Compatibility
Extra fields have been added to the `polaris_cdm_bootstrap`, `polaris_cdm_bootstrap_cces_aws` and
`polaris_cdm_bootstrap_cces_azure` resource to increase backwards compatibility with the older Rubrik (CDM) Terraform
provider, making it possible to migrate from using the old provider. See issue
[20](https://github.com/rubrikinc/terraform-aws-rubrik-cloud-cluster-elastic-storage/issues/20) more information.
