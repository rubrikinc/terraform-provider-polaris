# Create an AWS cloud cluster using RSC
resource "polaris_aws_cloud_cluster" "example" {
  cloud_account_id     = "12345678-1234-1234-1234-123456789012"
  region               = "us-west-2"
  use_placement_groups = true

  cluster_config {
    cluster_name            = "my-cloud-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    dns_search_domains      = ["example.com"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    bucket_name             = "my-s3-bucket"
    enable_immutability     = true
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version           = "9.4.0-p2-30507"
    instance_type         = "M6I_2XLARGE"
    instance_profile_name = "RubrikCloudClusterInstanceProfile"
    vpc_id                = "vpc-12345678"
    subnet_id             = "subnet-12345678"
    security_group_ids    = ["sg-12345678", "sg-45678901"]
  }
}

# Create an AWS cloud cluster with Multi-AZ resiliency
resource "polaris_aws_cloud_cluster" "multi_az" {
  cloud_account_id     = "12345678-1234-1234-1234-123456789012"
  region               = "us-west-2"
  is_az_resilient      = true
  use_placement_groups = false

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    bucket_name             = "my-s3-bucket"
    enable_immutability     = true
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version           = "9.4.0-p2-30507"
    instance_type         = "M6I_2XLARGE"
    instance_profile_name = "RubrikCloudClusterInstanceProfile"
    vpc_id                = "vpc-12345678"
    security_group_ids    = ["sg-12345678", "sg-45678901"]

    subnet_az_config {
      availability_zone = "us-west-2a"
      subnet            = "subnet-11111111"
    }

    subnet_az_config {
      availability_zone = "us-west-2b"
      subnet            = "subnet-22222222"
    }

    subnet_az_config {
      availability_zone = "us-west-2c"
      subnet            = "subnet-33333333"
    }
  }
}
