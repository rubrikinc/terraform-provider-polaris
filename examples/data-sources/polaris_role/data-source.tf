# Look up role by name.
data "polaris_role" "owner" {
  name = "Owner"
}

# Look up role by ID.
data "polaris_role" "owner" {
  role_id = "00000000-0000-0000-0000-000000000009"
}
