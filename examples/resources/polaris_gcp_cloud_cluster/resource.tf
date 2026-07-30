# Create a GCP cloud cluster (CCES) using RSC.
resource "polaris_gcp_cloud_cluster" "example" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  region           = "us-west1"
  zone             = "us-west1-a"

  cluster_config {
    cluster_name = "my-cloud-cluster"
    # admin_email and admin_password are write-only (requires Terraform v1.11.0+).
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    dns_search_domains      = ["example.com"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 1
    bucket_name             = "my-gcs-bucket"
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version      = "9.4.0-p2-30507"
    instance_type    = "N2_STANDARD_8"
    network          = "my-vpc"
    subnet           = "my-subnet"
    host_project     = "my-shared-vpc-host-project"
    service_accounts = ["cces-sa@my-project.iam.gserviceaccount.com"]
  }
}

# Create a Multi-AZ (availability-zone resilient) GCP cloud cluster. When
# az_resilient is true, omit `subnet` and provide one `subnet_az_config` block
# per zone. `zone` may be omitted and defaults to the first subnet_az_config
# availability zone.
resource "polaris_gcp_cloud_cluster" "multi_az" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  region           = "us-west1"
  az_resilient     = true

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8", "8.8.4.4"]
    ntp_servers             = ["pool.ntp.org"]
    num_nodes               = 3
    bucket_name             = "my-gcs-bucket"
    keep_cluster_on_failure = false
  }

  vm_config {
    cdm_version      = "9.4.0-p2-30507"
    instance_type    = "N2_STANDARD_8"
    network          = "my-vpc"
    host_project     = "my-shared-vpc-host-project"
    service_accounts = ["cces-sa@my-project.iam.gserviceaccount.com"]

    subnet_az_config {
      availability_zone = "us-west1-a"
      subnet            = "my-subnet"
    }
    subnet_az_config {
      availability_zone = "us-west1-b"
      subnet            = "my-subnet"
    }
    subnet_az_config {
      availability_zone = "us-west1-c"
      subnet            = "my-subnet"
    }
  }
}
