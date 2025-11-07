resource "polaris_gcp_custom_labels" "labels" {
  custom_labels = {
    "app"    = "RSC"
    "vendor" = "Rubrik"
  }
}
