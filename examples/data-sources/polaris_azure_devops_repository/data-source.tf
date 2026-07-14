# Look up an Azure DevOps repository by name.
data "polaris_azure_devops_repository" "repo" {
  name = "my-repo"
}

# Look up by RSC repository ID.
data "polaris_azure_devops_repository" "by_id" {
  id = "f1e2d3c4-1234-4c5b-9abc-0123456789ab"
}
