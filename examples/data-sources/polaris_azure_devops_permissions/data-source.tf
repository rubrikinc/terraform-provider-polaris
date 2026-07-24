# Look up the permissions RSC requires for an Azure DevOps feature.
data "polaris_azure_devops_permissions" "repo" {
  feature = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
  permission_groups = [
    "BASIC",
    "RECOVERY"
  ]
}

output "permissions" {
  value = jsondecode(data.polaris_azure_devops_permissions.repo.permissions)
}

output "permission_group_versions" {
  value = data.polaris_azure_devops_permissions.repo.permission_group_versions
}
