# With security groups managed by RSC.
resource "polaris_aws_exocompute" "default" {
  account_id = polaris_aws_account.default.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# With security groups managed by the user.
resource "polaris_aws_exocompute" "default" {
  account_id                = polaris_aws_account.default.id
  cluster_security_group_id = "sg-005656347687b8170"
  node_security_group_id    = "sg-00e147656785d7e2f"
  region                    = "us-east-2"
  vpc_id                    = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# Using the exocompute resources shared by an exocompute host.
resource "polaris_aws_exocompute" "default" {
  account_id      = polaris_aws_account.app.id
  host_account_id = polaris_aws_account.host.id
}
