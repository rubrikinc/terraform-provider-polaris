# RSC managed Exocompute with security groups managed by RSC.
resource "polaris_aws_exocompute" "exocompute" {
  account_id = polaris_aws_account.account.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# RSC managed Exocompute with security groups managed by the customer.
resource "polaris_aws_exocompute" "exocompute" {
  account_id                = polaris_aws_account.account.id
  cluster_security_group_id = "sg-005656347687b8170"
  node_security_group_id    = "sg-00e147656785d7e2f"
  region                    = "us-east-2"
  vpc_id                    = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# Customer managed Exocompute.
resource "polaris_aws_exocompute" "exocompute" {
  account_id = polaris_aws_account.account.id
  region     = "us-east-2"
}

resource "polaris_aws_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-eks-cluster"
  exocompute_id = polaris_aws_exocompute.exocompute.id
}

# Using the exocompute resources shared by an Exocompute host.
resource "polaris_aws_exocompute" "exocompute" {
  account_id      = polaris_aws_account.account.id
  host_account_id = polaris_aws_account.host.id
}
