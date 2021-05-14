resource "polaris_aws_account" "default" {
  name = "Trinity-AWS-FDSE"
  profile = "default"
  regions = [
    "us-east-2",
    "us-west-2"
  ]
}
