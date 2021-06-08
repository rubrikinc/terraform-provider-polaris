# With GCP service account key file
resource "polaris_gcp_project" "default" {
  credentials = "${path.module}/my-project-3f88757a02a4.json"
  project     = "my-project"
}

# Without GCP service account key file
resource "polaris_gcp_project" "default" {
  organization_name = "My Organization"
  project = "my-project"
  project_name = "My Project"
  project_number = 123456789012
}
