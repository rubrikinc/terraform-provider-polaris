data "polaris_gcp_project" "shared_vpc_host" {
  name = "my-shared-vpc-host-project"
}

resource "polaris_gcp_exocompute" "exocompute" {
  cloud_account_id     = polaris_gcp_project.project.id
  trigger_health_check = true

  regional_config {
    region          = "us-west1"
    subnet_name     = "my-shared-vpc-subnet-01"
    vpc_name        = "my-shared-vpc-01"
    host_project_id = data.polaris_gcp_project.shared_vpc_host.project_id
  }

  regional_config {
    region          = "us-east1"
    subnet_name     = "my-shared-vpc-subnet-02"
    vpc_name        = "my-shared-vpc-02"
    host_project_id = data.polaris_gcp_project.shared_vpc_host.project_id
  }
}
