# With local user account
provider "polaris" {
  credentials = "my-account"
}

# With service account
provider "polaris" {
  credentials = "/path/to/service-account-credentials.json"
}
