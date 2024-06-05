# Hardcoded values. Permission groups defaults to BASIC.
resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"

  regions = [
    "us-east-2",
    "us-west-2",
  ]

  feature {
    name = "CLOUD_NATIVE_ARCHIVAL"
  }

  feature {
    name = "CLOUD_NATIVE_PROTECTION"

    permission_groups = [
      "BASIC",
      "EXPORT_AND_RESTORE",
      "EXPORT_AND_RESTORE",
    ]
  }
}

# Using variables for the account values and the features. The dynamic
# feature block could also be expanded from the polaris_aws_cnp_artifacts
# data source.
resource "polaris_aws_cnp_account" "account" {
  cloud       = var.cloud
  external_id = var.external_id
  name        = var.name
  native_id   = var.native_id
  regions     = var.regions

  dynamic "feature" {
    for_each = var.features
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}
