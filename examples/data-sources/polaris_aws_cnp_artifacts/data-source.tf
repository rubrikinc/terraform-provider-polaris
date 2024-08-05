# Permission groups defaults to BASIC.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"
  }
}

# Multiple permission groups. When permission groups are specified,
# the BASIC permission group must always be included.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
      "FILE_LEVEL_RECOVERY",
    ]
  }
}

# Multiple features with permission groups.
data "polaris_aws_cnp_artifacts" "artifacts" {
  feature {
    name = "CLOUD_NATIVE_ARCHIVAL"

    permission_groups = [
      "BASIC",
    ]
  }

  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
      "FILE_LEVEL_RECOVERY",
    ]
  }
}
