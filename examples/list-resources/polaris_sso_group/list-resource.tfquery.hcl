list "polaris_sso_group" "all" {
  provider = polaris
}

list "polaris_sso_group" "by_name" {
  provider = polaris

  config {
    name = "Engineering"
  }
}

list "polaris_sso_group" "by_name_and_domain" {
  provider = polaris

  config {
    name            = "Engineering"
    auth_domain_id  = "12345678-1234-1234-1234-123456789012"
  }
}
