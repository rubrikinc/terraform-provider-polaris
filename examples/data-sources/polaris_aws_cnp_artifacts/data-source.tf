# Single feature with permission groups.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"
    permission_groups = [
      "BASIC",
    ]
  }
}

# Single feature with multiple permission groups. When multiple permission
# groups are specified, the BASIC permission group must always be included.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "EXOCOMPUTE"
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]
  }
}

# Multiple features with permission groups.
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
