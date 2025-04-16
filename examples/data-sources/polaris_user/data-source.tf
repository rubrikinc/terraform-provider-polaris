# Look up user by email address.
data "polaris_user" "admin" {
  email = "admin@example.org"
}

# Look up user by user ID.
data "polaris_user" "admin" {
  user_id = "auth0|700265c9583ef80078bb36b0"
}

# Look up SSO user by user ID.
data "polaris_user" "admin" {
  user_id = "samlp|...my-rubrik-account|admin@example.org"
}
