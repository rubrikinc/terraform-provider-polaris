# Create a compliance auditor user using the polaris_role data source.
resource "polaris_user" "auditor" {
  email    = "name@example.com"
  role_ids = [
    data.polaris_role.compliance_auditor.id
  ]
}
