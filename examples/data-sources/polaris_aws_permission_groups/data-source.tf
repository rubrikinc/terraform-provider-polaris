# Look up the latest permission groups available for a single RSC AWS feature.
data "polaris_aws_permission_groups" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"
}

# Use the result with the splat operator to feed permission group names into a
# polaris_aws_cnp_account feature block instead of hard-coding them.
output "cnp_permission_groups" {
  value = data.polaris_aws_permission_groups.cnp.permission_groups[*].name
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
