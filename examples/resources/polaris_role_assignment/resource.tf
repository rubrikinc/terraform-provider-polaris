# Assign a role to a user using the polaris_role data source.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id = data.polaris_role.compliance_auditor.id
  user_email = "name@example.com"
}

# Assign a role to a user using the polaris_custom_role resource.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id = polaris_custom_role.compliance_auditor.id
  user_email = "name@example.com"
}
