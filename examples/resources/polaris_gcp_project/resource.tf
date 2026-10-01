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

# BigQuery protection. The datasets to protect are in one project, and RSC
# runs the backup and recovery jobs on a BigQuery slot reservation in a
# dedicated reservation project. Only one project per RSC account can be the
# reservation project.
resource "polaris_gcp_project" "bigquery" {
  project        = "my-bigquery-project"
  project_name   = "My BigQuery Project"
  project_number = 123456789012

  feature {
    name              = "GCP_BIGQUERY_PROTECTION"
    permission_groups = ["BASIC", "EXPORT_AND_RESTORE"]
  }
}

resource "polaris_gcp_project" "bigquery_reservation" {
  project        = "my-reservation-project"
  project_name   = "My Reservation Project"
  project_number = 210987654321

  feature {
    name              = "GCP_BIGQUERY_RESERVATION"
    permission_groups = ["BASIC"]
  }
}
