# Look up SSO group by name.
data "polaris_sso_group" "admins" {
  name = "Administrators"
}

# Look up SSO group by ID.
data "polaris_sso_group" "admins" {
  sso_group_id = "samlpgroup|...my-rubrik-account|Administrators"
}
