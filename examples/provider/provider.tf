# With local user account
provider "polaris" {
  credentials = "my-account-name"
}

# With service account
provider "polaris" {
  credentials = "${path.module}/polaris-service-account.json"
}
