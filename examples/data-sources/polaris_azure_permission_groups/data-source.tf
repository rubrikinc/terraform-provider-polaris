# Look up the latest permission groups available for a single RSC Azure feature.
data "polaris_azure_permission_groups" "cnp" {
  feature = "CLOUD_NATIVE_PROTECTION"
}

# Splat over permission_groups to get just the group names — feed straight
# into a polaris_azure_subscription feature block instead of hard-coding them.
output "cnp_permission_groups" {
  value = data.polaris_azure_permission_groups.cnp.permission_groups[*].name
}

# Chain into polaris_azure_permissions to get the policy-shaped permission
# lists that drive an Azure role definition for this feature.
data "polaris_azure_permissions" "cnp" {
  feature           = data.polaris_azure_permission_groups.cnp.feature
  permission_groups = data.polaris_azure_permission_groups.cnp.permission_groups[*].name
}

# Look up several features at once with for_each.
data "polaris_azure_permission_groups" "all" {
  for_each = toset([
    "CLOUD_NATIVE_PROTECTION",
    "AZURE_SQL_DB_PROTECTION",
    "EXOCOMPUTE",
  ])

  feature = each.key
}

output "permission_groups_by_feature" {
  value = {
    for f, d in data.polaris_azure_permission_groups.all :
    f => d.permission_groups[*].name
  }
}
