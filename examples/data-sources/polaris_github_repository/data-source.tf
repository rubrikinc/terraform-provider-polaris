# Look up a GitHub repository by name, scoped to an organization.
data "polaris_github_repository" "repo" {
  name   = "my-repo"
  org_id = "d7f3e5a0-1234-4c5b-9abc-0123456789ab"
}

# Look up by RSC repository ID.
data "polaris_github_repository" "by_id" {
  id = "f1e2d3c4-1234-4c5b-9abc-0123456789ab"
}
