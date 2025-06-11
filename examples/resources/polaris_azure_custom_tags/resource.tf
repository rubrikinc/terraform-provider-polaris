
resource "polaris_azure_custom_tags" "tags" {
  custom_tags = {
    "app"    = "RSC"
    "vendor" = "Rubrik"
  }
}
