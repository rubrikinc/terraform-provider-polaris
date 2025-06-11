resource "polaris_aws_custom_tags" "tags" {
  custom_tags = {
    "app"    = "RSC"
    "vendor" = "Rubrik"
  }
}
