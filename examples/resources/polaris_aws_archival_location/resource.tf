data "polaris_aws_account" "archival" {
  name = "archival-account"
}

# Source region.
resource "polaris_aws_archival_location" "archival_location" {
  account_id     = data.polaris_aws_account.archival.id
  name           = "my-archival-location"
  bucket_prefix  = "e089osn2qn"
}

# Specific region.
resource "polaris_aws_archival_location" "archival_location" {
  account_id     = data.polaris_aws_account.archival.id
  name           = "my-archival-location"
  bucket_prefix  = "f48wad7flz"
  region         = "us-east-2"
}
