# Enable Cloud Native Protection in the us-east-2 region.
resource "polaris_aws_account" "account" {
  profile = "default"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }
}

# Enable Cloud Native Protection in teh us-east-2 and us-west-2 regions
# and Exocompute in the us-west-2 region. The Exocompute cluster will be
# managed by RSC.
resource "polaris_aws_account" "account" {
  profile = "default"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  exocompute {
    permission_groups = [
      "BASIC",
      "RSC_MANAGED_CLUSTER",
    ]

    regions = [
      "us-west-2",
    ]
  }
}
# Enable Cloud Native Protection and DSPM with Outpost.
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  dspm {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  outpost {
    outpost_account_id      = "123456789123"
    outpost_account_profile = "outpost"

    permission_groups = [
      "BASIC",
    ]
  }
}

# Enable Cloud Native Protection and Data Scanning with Outpost.
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }

  outpost {
    outpost_account_id      = "123456789123"
    outpost_account_profile = "outpost"

    permission_groups = [
      "BASIC",
    ]
  }
}

