# With service account key file
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-3f88757a02a4.json"
}

# Without service account key file
resource "polaris_gcp_project" "default" {
  project        = "my-project"
  project_number = 123456789012
}
