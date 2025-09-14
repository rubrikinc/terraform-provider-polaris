# Create an AWS cloud cluster in RSC
resource "polaris_aws_cloud_cluster" "example" {
  cloud_account_id     = "12345678-1234-1234-1234-123456789012"
  region               = "us-west-2"
  is_es_type           = true
  use_placement_groups = true

  cluster_config {
    cluster_name         = "my-cloud-cluster"
    user_email           = "admin@example.com"
    admin_password       = "RubrikGoForward!"
    dns_name_servers     = ["8.8.8.8", "8.8.4.4"]
    dns_search_domain    = ["example.com"]
    ntp_servers          = ["pool.ntp.org"]
    num_nodes            = 3
    bucket_name          = "my-cluster-bucket"
    enable_immutability  = true
    should_create_bucket = true
    enable_object_lock   = true

    vm_config {
      instance_type         = "M6I_2XLARGE"
      instance_profile_name = "RubrikCloudClusterInstanceProfile"
      vpc                   = "vpc-12345678"
      subnet                = "subnet-12345678"
      security_groups       = ["sg-12345678", "sg-87654321"]
    }
  }
}
