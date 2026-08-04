# Look up the GCP regions and their availability zones that RSC supports for a
# cloud account.
data "polaris_gcp_regions" "example" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
}

# Resolve a single region and its (sorted) zones.
locals {
  region = one([for r in data.polaris_gcp_regions.example.regions : r if r.name == "us-west1"])
  zones  = sort(tolist(local.region.zones))
}

# Drive a Multi-AZ cluster's subnet_az_config from the region's zones with
# for_each, instead of hard-coding the availability zones.
resource "polaris_gcp_cloud_cluster" "multi_az" {
  cloud_account_id = "12345678-1234-1234-1234-123456789012"
  region           = local.region.name
  az_resilient     = true

  cluster_config {
    cluster_name            = "my-multi-az-cluster"
    admin_email             = "admin@example.com"
    admin_password          = "RubrikGoForward!"
    dns_name_servers        = ["8.8.8.8"]
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

    dynamic "subnet_az_config" {
      for_each = local.zones
      content {
        availability_zone = subnet_az_config.value
        subnet            = "my-subnet"
      }
    }
  }
}
