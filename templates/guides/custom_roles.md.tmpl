---
page_title: "Custom Roles"
---

# Managing RSC roles using Terraform
In v0.5.0, support for custom roles has been added through two new resources and two new data sources:
 * `polaris_custom_role` _(Resource)_
 * `polaris_role_assignment` _(Resource)_
 * `polaris_role` _(Data Source)_
 * `polaris_role_template` _(Data Source)_

The `polaris_custom_role` resource is used to define custom roles. The `polaris_role_assignment` resource is used to
assign a role, custom or builtin, to a user. The `polaris_role` data source is used to refer to a role, custom or
builtin, by name. And finally, the `polaris_role_template` data source is used to refer to a builtin RSC role template
by name.

## Creating a custom role
Custom roles can be created in two different ways, either by specifying the permissions of the role manually or by
getting them from an existing RSC role template.

### Manual permissions
Here we create a custom role from scratch by entering the permissions the role should have. Valid values for `operation`
and `snappable_type` can be found in the RSC GraphQL API docs
[here](https://rubrikinc.github.io/rubrik-api-documentation/schema/reference/operation.doc.html) and
[here](https://rubrikinc.github.io/rubrik-api-documentation/schema/reference/workloadlevelhierarchy.doc.html).
```terraform
resource "polaris_custom_role" "compliance_auditor" {
  name = "Compliance Auditor Role"
  description = "Compliance Auditor"

  permission {
    operation = "EXPORT_DATA_CLASS_GLOBAL"
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids = [
        "GlobalResource"
      ]
    }
  }

  permission {
    operation = "VIEW_DATA_CLASS_GLOBAL"
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids = [
        "GlobalResource"
      ]
    }
  }
}
```

### From a role template
Here we make use of the `polaris_role_template` data source to refer to an RSC role template by name. The role templates
available in RSC can be found in the UI, under _Settings / Users and Access / Roles / Create Role_.
```terraform
data "polaris_role_template" "compliance_auditor" {
  name = "Compliance Auditor"
}

resource "polaris_custom_role" "compliance_auditor" {
  name        = "Compliance Auditor Role"
  description = "Based on the ${data.polaris_role_template.compliance_auditor.name} template"

  dynamic "permission" {
    for_each = data.polaris_role_template.compliance_auditor.permission
    content {
      operation = permission.value["operation"]

      dynamic "hierarchy" {
        for_each = permission.value["hierarchy"]
        content {
          snappable_type = hierarchy.value["snappable_type"]
          object_ids     = hierarchy.value["object_ids"]
        }
      }
    }
  }
}
```

## Assigning a role to a user
Assigning a role to a user is done using the `polaris_role_assignment` resource. The resource take to arguments, the id
of the role and the email address of the user. For builtin roles or roles being defined elsewhere the `polaris_role`
data source can be used to refer to the role by name.

## Assigning a role
Here we have a custom role defined in the same Terraform configuration with the label `compliance_auditor` which we
refer to.
```terraform
resource "polaris_role_assignment" "compliance_auditor" {
  role_id = polaris_custom_role.compliance_auditor.id
  user_email = "name@example.com"
}
```

## Assigning a role defined elsewhere
Here we make use of the `polaris_role` data source to refer to an RSC role by name. This named role can be builtin,
defined in the UI or by another Terraform configuration.
```terraform
data "polaris_role" "compliance_auditor" {
    name = "Compliance Auditor"
}

resource "polaris_role_assignment" "compliance_auditor" {
  role_id = data.polaris_role.compliance_auditor.id
  user_email = "name@example.com"
}
```
