# Enable Cloud Native Protection
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}

# Enable Cloud Native Protection and Exocompte. Generates a diff when
# permissions are missing for the enabled features.
resource "polaris_aws_account" "default" {
  permissions = "update"
  profile     = "default"

  cloud_native_protection {
    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  exocompute {
    regions = [
      "us-west-2",
    ]
  }
}
