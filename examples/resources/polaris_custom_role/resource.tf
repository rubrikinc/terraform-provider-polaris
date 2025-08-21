# Manually defined role.
resource "polaris_custom_role" "auditor" {
  name        = "Compliance Auditor Role"
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

# From role template.
data "polaris_role_template" "auditor" {
  name = "Compliance Auditor"
}

resource "polaris_custom_role" "auditor" {
  name        = "Compliance Auditor Role"
  description = "Based on the ${data.polaris_role_template.auditor.name} template"

  dynamic "permission" {
    for_each = data.polaris_role_template.auditor.permission
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
