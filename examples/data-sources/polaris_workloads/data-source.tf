data "polaris_workloads" "workloads" {}

# Create a map of workloads for easy lookup.
locals {
  workloads = {
    for w in data.polaris_workloads.workloads.workloads : w => w
  }
}

resource "polaris_custom_role" "azure_admin" {
  name        = "RSC Azure Admin"
  description = "Custom role for Azure admin."

  permission {
    operation = "VIEW_INVENTORY"
    hierarchy {
      # AllSubHierarchyType is the only valid snappable type for the
      # VIEW_INVENTORY operation.
      snappable_type = local.workloads.AllSubHierarchyType
      object_ids = [
        "GlobalResource"
      ]
    }
  }

  permission {
    operation = "VIEW_INVENTORY"
    hierarchy {
      # AwsNativeRdsInstance is a valid snappable type for the VIEW_INVENTORY
      # operation.
      snappable_type = local.workloads.AwsNativeRdsInstance
      object_ids = [
        "AWSNATIVE_ROOT"
      ]
    }
    hierarchy {
      # AllSubHierarchyType is a valid snappable type for the VIEW_INVENTORY
      # operation.
      snappable_type = local.workloads.AllSubHierarchyType
      object_ids = [
        "ORACLE_ROOT"
      ]
    }
  }
}
