# Look up an Azure DevOps project by name.
data "polaris_azure_devops_project" "project" {
  name = "my-project"
}

# Look up by RSC project ID.
data "polaris_azure_devops_project" "by_id" {
  id = "a1b2c3d4-1234-4c5b-9abc-0123456789ab"
}
