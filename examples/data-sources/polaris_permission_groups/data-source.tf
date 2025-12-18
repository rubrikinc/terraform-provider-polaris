# Look up the permission groups required for AWS features.
data "polaris_permission_groups" "aws" {
  cloud_provider = "AWS"
}

# Filter for specific features using a local variable.
locals {
  desired_features = ["CLOUD_NATIVE_PROTECTION", "EXOCOMPUTE"]

  filtered_features = [
    for feature in data.polaris_permission_groups.aws.features :
    feature if contains(local.desired_features, feature.name)
  ]
}

# Output the filtered features with their required permission groups.
output "filtered_features" {
  value = local.filtered_features
}

