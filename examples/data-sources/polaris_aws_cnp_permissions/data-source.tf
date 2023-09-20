data "polaris_aws_cnp_permissions" "permissions" {
  for_each = data.polaris_aws_cnp_artifacts.artifacts.role_keys
  cloud    = data.polaris_aws_cnp_artifacts.artifacts.cloud
  features = data.polaris_aws_cnp_artifacts.artifacts.features
  role_key = each.key
}
