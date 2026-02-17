# Enable Cloud Native Protection in the us-east-2 and us-west-2 regions
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

# Enable Data Scanning and Outpost. Note, the Cyber Recovery Data
# Scanning, Data Scanning and DSPM features require the Outpost feature.
resource "polaris_aws_account" "account" {
  profile = "default"

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
    permission_groups = [
      "BASIC",
    ]
  }
}

# Enable the Outpost feature for one account, the Cyber Recovery Data
# Scanning and Data Scanning features for another and DSPM for a third.
# Note, the Outpost account must be onboarded first. Use depends_on to
# enforce the ordering.
resource "polaris_aws_account" "outpost" {
  profile = "outpost"

  outpost {
    permission_groups = [
      "BASIC",
    ]
  }
}

resource "polaris_aws_account" "account1" {
  profile = "account1"

  cyber_recovery_data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }

  data_scanning {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }

  depends_on = [
    polaris_aws_account.outpost,
  ]
}

resource "polaris_aws_account" "account2" {
  profile = "account1"

  dspm {
    permission_groups = [
      "BASIC",
    ]

    regions = [
      "us-east-2",
    ]
  }

  depends_on = [
    polaris_aws_account.outpost,
  ]
}
