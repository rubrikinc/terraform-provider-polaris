list "polaris_user" "all" {
  provider = polaris
}

list "polaris_user" "by_email" {
  provider = polaris

  config {
    email = "auditor@example.org"
  }
}
