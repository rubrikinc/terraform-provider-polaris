data "polaris_role" "compliance_auditor" {
  name = "Compliance Auditor Role"
}

data "polaris_user" "compliance_auditor" {
  email = "auditor@example.org"
}

# Assign custom compliance auditor role to user.
resource "polaris_role_assignment" "compliance_auditor" {
  user_id = data.polaris_user.compliance_auditor.id

  role_ids = [
    data.polaris_role.compliance_auditor.id,
  ]
}

data "polaris_sso_group" "compliance_auditors" {
  name = "ComplianceAuditors"
}

# Assign role to SSO users using an SSO group.
resource "polaris_role_assignment" "compliance_auditor" {
  sso_group_id = data.polaris_sso_group.compliance_auditors.id

  role_ids = [
    data.polaris_role.compliance_auditor.id,
  ]
}
