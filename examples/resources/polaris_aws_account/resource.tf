# Enable Cloud Native Protection
resource "polaris_aws_account" "default" {
  profile = "default"

  cloud_native_protection {
    regions = [
      "us-east-2",
    ]
  }
}

# Enable Cloud Native Protection and Exocompte.
resource "polaris_aws_account" "default" {
  profile = "default"

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

# The Couldformation stack ARN is available after creation
output "stack_arn" {
  value       = polaris_aws_account.default.exocompute[0].stack_arn
}
