# Enable Cloud Native Protection
resource "polaris_aws_account" "default" {
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

# Enable Cloud Native Protection and Exocompute.
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

resource "polaris_aws_account" "default" {
  profile = "default"

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

    regions = [
      "us-east-2",
      "us-west-2",
    ]
  }
}

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

  outpost {
    outpost_account_id      = "123456789123"
    outpost_account_profile = "outpost"

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
}



# The Couldformation stack ARN is available after creation
output "stack_arn" {
  value = polaris_aws_account.default.exocompute[0].stack_arn
}
