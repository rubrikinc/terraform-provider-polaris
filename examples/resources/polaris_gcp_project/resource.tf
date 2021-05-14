# With service account
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/trinity-fdse-3f88757a02a4.json"
  project     = "trinity-fdse"
}

# Without service account
resource "polaris_gcp_project" "default" {
  organization_name = "Trinity Organization"
  project = "trinity-fdse"
  project_name = "Trinity FDSE"
  project_number = 994761414559
}
