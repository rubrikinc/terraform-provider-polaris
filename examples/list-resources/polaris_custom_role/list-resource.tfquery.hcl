list "polaris_custom_role" "all" {
  provider = polaris
}

list "polaris_custom_role" "by_name" {
  provider = polaris

  config {
    name = "Compliance Auditor"
  }
}
