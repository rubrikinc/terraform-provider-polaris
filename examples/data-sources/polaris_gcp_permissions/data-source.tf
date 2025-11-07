data "polaris_gcp_permissions" "cloud_native_archival" {
  feature = "CLOUD_NATIVE_ARCHIVAL"
  permission_groups = [
    "BASIC",
    "ENCRYPTION",
  ]
}
