data "polaris_role" "compliance_auditor" {
  name = "Compliance Auditor Role"
}

data "polaris_user" "compliance_auditor" {
  email = "auditor@example.org"
}

# Assign role to user using data sources.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id = data.polaris_role.compliance_auditor.id
  user_id = data.polaris_user.compliance_auditor.id
}

# Assign role to user using email address.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id    = data.polaris_role.compliance_auditor.id
  user_email = "auditor@example.org"
}

# Assign custom compliance auditor role to user.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id = polaris_custom_role.compliance_auditor.id
  user_id = data.polaris_user.compliance_auditor.id
}

data "polaris_sso_group" "compliance_auditors" {
  name = "ComplianceAuditors"
}

# Assign role to SSO users using an SSO group.
resource "polaris_role_assignment" "compliance_auditor" {
  role_id      = data.polaris_role.compliance_auditor.id
  sso_group_id = data.polaris_sso_group.compliance_auditors.id
}
