terraform {
  required_providers {
    polaris = {
      source  = "terraform.rubrik.com/rubrik/polaris"
      version = "~> 0.0.3"
    }
  }
}

provider "polaris" {
  account = "default"
}

resource "polaris_aws_account" "default" {
  profile = "default"
  regions = [
    "us-east-2"
  ]
}
