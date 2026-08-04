# Look up an onboarded GitHub organization by name.
data "polaris_github_organization" "org" {
  name = "my-org"
}

# Look up by native ID, GitHub's numeric organization ID.
data "polaris_github_organization" "by_native_id" {
  native_id = "54376070"
}

# Look up by RSC organization ID.
data "polaris_github_organization" "by_id" {
  id = "d7f3e5a0-1234-4c5b-9abc-0123456789ab"
}
