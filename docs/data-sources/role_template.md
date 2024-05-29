---
page_title: "polaris_role_template Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
---

# polaris_role_template (Data Source)




## Example Usage

```terraform
data "polaris_role_template" "compliance_auditor" {
  name = "Compliance Auditor"
}
```


## Schema

### Required

- `name` (String) Role name.

### Read-Only

- `description` (String) Role description.
- `id` (String) The ID of this resource.
- `permission` (Set of Object) Role permission. (see [below for nested schema](#nestedatt--permission))

<a id="nestedatt--permission"></a>
### Nested Schema for `permission`

Read-Only:

- `hierarchy` (Set of Object) Snappable hierarchy. (see [below for nested schema](#nestedobjatt--permission--hierarchy))
- `operation` (String) Operation allowed on object ids under the snappable hierarchy.

<a id="nestedobjatt--permission--hierarchy"></a>
### Nested Schema for `permission.hierarchy`

Read-Only:

- `object_ids` (Set of String) Object/workload identifiers.
- `snappable_type` (String) Snappable/workload type.
