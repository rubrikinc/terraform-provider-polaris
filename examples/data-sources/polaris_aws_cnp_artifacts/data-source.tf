# Single feature with one permission group.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
    ]
  }
}

# Single feature with multiple permission groups. BASIC must always be
# included, except for the SERVERS_AND_APPS feature which only supports
# the CLOUD_CLUSTER_ES permission group.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "EXOCOMPUTE"
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]
  }
}

# Multiple features with multiple permission groups.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "EXOCOMPUTE"
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]
  }
}

# Using a variable for the features.
variable "features" {
  type = map(object({
    permission_groups = set(string)
  }))
  description = "RSC features with permission groups."
}

data "polaris_aws_cnp_artifacts" "artifacts" {
  dynamic "feature" {
    for_each = var.features
    content {
      name              = feature.key
      permission_groups = feature.value["permission_groups"]
    }
  }
}
