# Look up user by email address.
data "polaris_user" "admin" {
  email = "admin@example.org"
}

# Look up user by email address and user domain.
data "polaris_user" "admin" {
  email  = "admin@example.org"
  domain = "SSO"
}

# Look up user by user ID.
data "polaris_user" "admin" {
  user_id = "<id>"
}
