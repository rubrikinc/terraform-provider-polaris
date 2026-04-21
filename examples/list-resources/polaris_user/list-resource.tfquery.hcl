list "polaris_user" "all" {
  provider = polaris
}

list "polaris_user" "filtered" {
  provider = polaris

  config {
    email = "john.doe@example.com"
  }
}
