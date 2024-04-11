data "polaris_gcp_permissions" "default" {
  features = [
    "CLOUD_NATIVE_PROTECTION",
  ]
}
