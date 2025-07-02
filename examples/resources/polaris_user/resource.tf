data "polaris_role" "auditor" {
  name = "Compliance Auditor Role"
}

resource "polaris_user" "auditor" {
  email = "auditor@example.org"

  role_ids = [
    data.polaris_role.auditor.id
  ]
}
