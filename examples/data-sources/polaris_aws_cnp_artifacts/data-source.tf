data "polaris_aws_cnp_artifacts" "artifacts" {
  cloud    = "STANDARD"
  features = ["CLOUD_NATIVE_PROTECTION"]
}
