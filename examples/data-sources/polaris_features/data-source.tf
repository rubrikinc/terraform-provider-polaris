# Output the features enabled for the RSC account.
data "polaris_features" "features" {}

output "features_enabled" {
  value = data.polaris_features.features
}
