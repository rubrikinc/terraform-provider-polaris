data "polaris_gcp_project" "project" {
  name = "my-gcp-project"
}

# Source region.
resource "polaris_gcp_archival_location" "archival_location" {
  cloud_account_id = data.polaris_gcp_project.project.id
  name             = "my-archival-location"
  bucket_prefix    = "my-bucket-prefix"
}

# Specific region.
resource "polaris_gcp_archival_location" "archival_location" {
  cloud_account_id = data.polaris_gcp_project.project.id
  name             = "my-archival-location"
  bucket_prefix    = "my-bucket-prefix"
  region           = "nam4"
}
