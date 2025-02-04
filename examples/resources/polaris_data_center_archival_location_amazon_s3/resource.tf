data "polaris_data_center_aws_account" "archival" {
  name = "archival-account"
}

resource "polaris_data_center_archival_location_amazon_s3" "archival_location" {
  name             = "amazon-s3-archival-location"
  cluster_id       = "a501dae9-a27b-4ade-a604-fb103fba2fde"
  cloud_account_id = data.polaris_data_center_aws_account.archival.id
  region           = "us-east-2"
  bucket_name      = "archival-bucket"
  kms_master_key   = "aws/s3"
}
