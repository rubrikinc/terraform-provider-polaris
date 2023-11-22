# Source region.
resource "polaris_aws_archival_location" "archival_location" {
  account_id     = polaris_aws_cnp_account.account.id
  name           = "my-archival-location"
  bucket_prefix  = "e089osn2qn"
}

# Specific region.
resource "polaris_aws_archival_location" "archival_location" {
  account_id     = polaris_aws_cnp_account.account.id
  name           = "my-archival-location"
  bucket_prefix  = "f48wad7flz"
  region         = "us-east-2"
}
