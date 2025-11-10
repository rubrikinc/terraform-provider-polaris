# With service account private key.
resource "google_service_account" "service_account" {
  account_id = "rubrik-service-account"
}

resource "google_service_account_key" "service_account" {
  service_account_id = google_service_account.service_account.name
}

resource "polaris_gcp_project" "project" {
  credentials    = google_service_account_key.service_account.private_key
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012
}

# With the RSC global service account key.
resource "polaris_gcp_project" "project" {
  project        = "my-project"
  project_name   = "My Project"
  project_number = 123456789012
}
