list "polaris_azure_devops_organization" "all" {
  provider = polaris
}

list "polaris_azure_devops_organization" "by_native_id" {
  provider = polaris

  config {
    native_id = "my-org"
  }
}
