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

# Lookup the required permissions using the output from the
# polaris_aws_cnp_artifacts data source.
data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  cloud    = data.polaris_aws_cnp_artifacts.artifacts.cloud
  role_key = each.key

  dynamic "feature" {
    for_each = data.polaris_aws_cnp_artifacts.artifacts.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}
