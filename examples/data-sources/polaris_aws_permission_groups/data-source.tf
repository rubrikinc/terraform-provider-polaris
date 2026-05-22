# Look up the latest permission groups available for one or more RSC AWS
# features.
data "polaris_aws_permission_groups" "groups" {
  feature_names = [
    "CLOUD_NATIVE_PROTECTION",
    "RDS_PROTECTION",
  ]
}

# Feed the discovered permission group names into a polaris_aws_cnp_account
# feature block instead of hard-coding them.
locals {
  feature_groups = {
    for f in data.polaris_aws_permission_groups.groups.feature :
    f.name => [for pg in f.permission_group : pg.name]
  }
}

output "permission_groups_by_feature" {
  value = local.feature_groups
}
