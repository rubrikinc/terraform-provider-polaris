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
    ]
  }
}

resource "polaris_aws_cnp_account" "account" {
  name      = "My Account"
  native_id = "123456789123"
  regions = [
    "us-east-2",
    "us-west-2",
  ]

  dynamic "feature" {
    for_each = data.polaris_aws_cnp_artifacts.artifacts.feature
    content {
      name              = feature.value["name"]
      permission_groups = feature.value["permission_groups"]
    }
  }
}

# Lookup the trust policies using the artifacts data source and the
# account resource.
resource "polaris_aws_cnp_account_trust_policy" "trust_policy" {
  for_each   = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  account_id = polaris_aws_cnp_account.account.id
  role_key   = each.key
}
