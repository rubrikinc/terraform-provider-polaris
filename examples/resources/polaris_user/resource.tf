data "polaris_role" "compliance_auditor" {
  name = "Compliance Auditor Role"
}

resource "polaris_user" "auditor" {
  email    = "auditor@example.com"
  role_ids = [
    data.polaris_role.compliance_auditor.id
  ]
}
