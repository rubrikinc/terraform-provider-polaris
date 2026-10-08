---
page_title: "Azure SQL DB Protection: specific feature input is required"
subcategory: "Common Issues"
---

# Azure SQL DB Protection: specific feature input is required

## Symptom

When onboarding, updating, or re-adding the SQL DB Protection feature on an Azure subscription, `terraform apply` fails
with one of these errors:

```
INVALID_ARGUMENT: invalid request: specific_input is empty
```

```
INVALID_ARGUMENT: Invalid Request: resource group input needed for feature AZURE_SQL_DB_PROTECTION.
```

The corresponding RSC backend log line is:

```
Valid specific feature input is required for feature 'AZURE_SQL_DB_PROTECTION', can't be empty. input: <nil>, err: invalid request: specific_input is empty
```

The same configuration may succeed when the subscription is added through the RSC UI but fail through Terraform, and it
commonly shows up when a subscription is deleted and then re-added.

## Cause

Once the `CNP_AZURE_SQL_DB_TDE_CMK_SUPPORT` feature flag has been rolled out on your RSC account, SQL DB Protection
requires a user-assigned managed identity (UAMI) — together with the feature's resource group — as part of the
onboarding input. When the `sql_db_protection` block omits these fields, RSC rejects the request because the
feature-specific input is empty (`<nil>`).

## Resolution

Provide the resource group and the four user-assigned managed identity fields in the `sql_db_protection` block of the
`polaris_azure_subscription` resource. The UAMI fields are required together — set all four:

```hcl
resource "polaris_azure_subscription" "example" {
  # ...

  sql_db_protection {
    permissions           = data.polaris_azure_permissions.sql_db_protection.id
    regions               = ["eastus"]
    resource_group_name   = "my-resource-group"
    resource_group_region = "eastus"

    user_assigned_managed_identity_name                = "my-uami"
    user_assigned_managed_identity_principal_id        = "00000000-0000-0000-0000-000000000000"
    user_assigned_managed_identity_region              = "eastus"
    user_assigned_managed_identity_resource_group_name = "my-resource-group"
  }
}
```

The UAMI fields can be wired directly to an `azurerm_user_assigned_identity` resource so the identity is managed in the
same configuration:

```hcl
  sql_db_protection {
    # ...
    user_assigned_managed_identity_name                = azurerm_user_assigned_identity.rubrik.name
    user_assigned_managed_identity_principal_id        = azurerm_user_assigned_identity.rubrik.principal_id
    user_assigned_managed_identity_region              = azurerm_user_assigned_identity.rubrik.location
    user_assigned_managed_identity_resource_group_name = azurerm_user_assigned_identity.rubrik.resource_group_name
  }
```

## Notes

- **If you instead see `Unexpected attribute: An attribute named "user_assigned_managed_identity_…" is not expected
  here`,** your provider version predates these fields. Upgrade to a recent release of the provider. Once the feature
  flag is enabled there is no configuration-only workaround on an older provider, because the RSC API itself changes
  when the flag is turned on.
- **Do not set the UAMI fields before the feature flag is enabled.** Supplying them before
  `CNP_AZURE_SQL_DB_TDE_CMK_SUPPORT` is rolled out to your account produces a different error. Add them only once the
  flag is active.
- **Retrospective upgrades are supported.** Subscriptions already onboarded for SQL DB Protection are not blocked by
  this change and continue to function. You can add these fields to an existing configuration to bring it up to date;
  doing so re-onboards the SQL DB Protection feature.
