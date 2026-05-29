# Look up the latest permission groups available for a single RSC AWS feature.
data "polaris_aws_permission_groups" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"
}

# Splat over permission_groups to get just the group names — feed straight
# into a polaris_aws_cnp_account feature block instead of hard-coding them.
output "cnp_permission_groups" {
  value = data.polaris_aws_permission_groups.cnp.permission_groups[*].name
}

# Splat across groups to get every IAM action required by the feature.
output "cnp_actions" {
  value = flatten(data.polaris_aws_permission_groups.cnp.permission_groups[*].statements[*].name)
}

# Look up several features at once with for_each.
data "polaris_aws_permission_groups" "all" {
  for_each = toset([
    "CLOUD_NATIVE_PROTECTION",
    "EXOCOMPUTE",
    "RDS_PROTECTION",
  ])

  feature = each.key
}

output "permission_groups_by_feature" {
  value = {
    for f, d in data.polaris_aws_permission_groups.all :
    f => d.permission_groups[*].name
  }
}
