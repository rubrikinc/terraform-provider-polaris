list "polaris_custom_role" "all" {
  provider = polaris
}

list "polaris_custom_role" "filtered" {
  provider = polaris

  config {
    name = "Compliance Auditor"
  }
}
