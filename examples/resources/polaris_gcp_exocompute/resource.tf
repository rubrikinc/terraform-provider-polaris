resource "polaris_gcp_exocompute" "exocompute" {
  cloud_account_id     = polaris_gcp_project.project.id
  trigger_health_check = true

  regional_config {
    region      = "us-west1"
    subnet_name = "my-vpc-subnet-01"
    vpc_name    = "my-vpc-01"
  }

  regional_config {
    region      = "us-east1"
    subnet_name = "my-vpc-subnet-02"
    vpc_name    = "my-vpc-02"
  }
}
