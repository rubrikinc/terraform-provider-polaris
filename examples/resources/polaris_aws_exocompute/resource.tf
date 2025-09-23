data "polaris_aws_account" "host" {
  name = "host-account"
}

# RSC managed Exocompute and security groups.
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
  vpc_id     = "vpc-4859acb9"

  subnets = [
    "subnet-ea67b67b",
    "subnet-ea43ec78"
  ]
}

# RSC managed Exocompute and customer managed security groups.
resource "polaris_aws_exocompute" "host" {
  account_id                = data.polaris_aws_account.host.id
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
resource "polaris_aws_exocompute" "host" {
  account_id = data.polaris_aws_account.host.id
  region     = "us-east-2"
}

resource "polaris_aws_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-eks-cluster"
  exocompute_id = polaris_aws_exocompute.host.id
}

data "polaris_aws_account" "application" {
  name = "application-account"
}

# Application Exocompute.
resource "polaris_aws_exocompute" "application" {
  account_id      = data.polaris_aws_account.application.id
  host_account_id = data.polaris_aws_account.host.id
}
