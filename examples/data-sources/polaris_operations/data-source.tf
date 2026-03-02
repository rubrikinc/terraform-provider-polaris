data "polaris_operations" "op" {}

# Create a map of operations for easy lookup.
locals {
  operations = {
    for op in data.polaris_operations.op.operations : op => op
  }
}

resource "polaris_custom_role" "azure_admin" {
  name        = "RSC Azure Admin"
  description = "Custom role for Azure admin."

  permission {
    # VIEW_INVENTORY is a valid operation. 
    operation = local.operations.VIEW_INVENTORY
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids = [
        "GlobalResource"
      ]
    }
  }

  permission {
    # RESTORE_TO_ORIGIN is a valid operation.
    operation = local.operations.RESTORE_TO_ORIGIN
    hierarchy {
      snappable_type = "AwsNativeRdsInstance"
      object_ids = [
        "AWSNATIVE_ROOT"
      ]
    }
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids = [
        "ORACLE_ROOT"
      ]
    }
  }
}
