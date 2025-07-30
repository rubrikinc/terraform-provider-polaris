terraform {
  required_providers {
    polaris = {
      source  = "rubrikinc/polaris"
      version = "0.9.0"
    }
  }
}

resource "polaris_aws_account" "default" {
  profile   = "default"

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
    outpost_account_id = "123456789123"
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
