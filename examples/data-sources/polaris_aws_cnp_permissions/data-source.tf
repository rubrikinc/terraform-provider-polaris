data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_ARCHIVAL"

    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "CLOUD_NATIVE_ARCHIVAL_ENCRYPTION"

    permission_groups = [
      "BASIC",
      "ENCRYPTION",
    ]
  }

  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
    ]
  }
}

# Lookup the required permissions using the output from the
# artifacts data source.
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
