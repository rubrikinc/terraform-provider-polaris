---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_role_assignment Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
---

# polaris_role_assignment (Resource)



## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `role_id` (String) Role identifier.
- `user_email` (String) User email address.

### Read-Only

- `id` (String) The ID of this resource.

