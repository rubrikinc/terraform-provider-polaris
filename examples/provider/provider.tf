# Service account from the current environment.
provider "polaris" {
}

# Service account from file.
provider "polaris" {
  credentials = "/path/to/service-account-credentials.json"
}

# Local user account.
provider "polaris" {
  credentials = "my-account"
}
