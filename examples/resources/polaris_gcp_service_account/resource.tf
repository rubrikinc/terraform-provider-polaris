resource "polaris_gcp_service_account" "default" {
  credentials = "${path.module}/trinity-fdse-3f88757a02a4.json"
}
