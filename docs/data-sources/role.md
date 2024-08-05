---
page_title: "polaris_role Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_role` data source is used to access information about RSC roles.

---

# polaris_role (Data Source)


The `polaris_role` data source is used to access information about RSC roles.



## Example Usage

```terraform
data "polaris_role" "compliance_auditor" {
  name = "Compliance Auditor Role"
}
```


## Schema

### Required

- `name` (String) Role name.

### Read-Only

- `description` (String) Role description.
- `id` (String) Role ID (UUID).
- `is_org_admin` (Boolean) True if the role is the organization administrator.
- `permission` (Set of Object) Role permission. (see [below for nested schema](#nestedatt--permission))

<a id="nestedatt--permission"></a>
### Nested Schema for `permission`

Read-Only:

- `hierarchy` (Set of Object) Snappable hierarchy. (see [below for nested schema](#nestedobjatt--permission--hierarchy))
- `operation` (String) Operation allowed on object IDs under the snappable hierarchy.

<a id="nestedobjatt--permission--hierarchy"></a>
### Nested Schema for `permission.hierarchy`

Read-Only:

- `object_ids` (Set of String) Object/workload identifiers.
- `snappable_type` (String) Snappable/workload type.
