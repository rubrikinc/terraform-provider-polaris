resource "polaris_aws_account" "default" {
  profile = "default"
  regions = [
    "us-east-2",
    "us-west-2"
  ]
}
