resource "polaris_gcp_service_account" "default" {
  credentials = "${path.module}/my-project-3f88757a02a4.json"
}
