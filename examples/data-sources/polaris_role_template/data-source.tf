# Look up role template by name.
data "polaris_role_template" "compliance_auditor" {
  name = "Compliance Auditor"
}

# Look up role template by ID.
data "polaris_role_template" "compliance_auditor" {
  role_template_id = "00000000-0000-0000-0000-000000000007"
}
