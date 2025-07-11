---
page_title: "polaris_user Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_user` data source is used to access information about an RSC user.
Information for both local and SSO users can be accessed. A user is looked up
using either the ID or the email address.

-> **Note:** RSC allows the same email address to be used, at the same time, by
   both local and SSO users. Use the `domain` field to specify in which domain
   to look for a user.

-> **Note:** The `status` field will always be `UNKNOWN` for SSO users.

---

# polaris_user (Data Source)


The `polaris_user` data source is used to access information about an RSC user.
Information for both local and SSO users can be accessed. A user is looked up
using either the ID or the email address.

-> **Note:** RSC allows the same email address to be used, at the same time, by
   both local and SSO users. Use the `domain` field to specify in which domain
   to look for a user.

-> **Note:** The `status` field will always be `UNKNOWN` for SSO users.



## Example Usage

```terraform
# Look up user by email address.
data "polaris_user" "admin" {
  email = "admin@example.org"
}

# Look up user by email address and user domain.
data "polaris_user" "admin" {
  email  = "admin@example.org"
  domain = "SSO"
}

# Look up user by user ID.
data "polaris_user" "admin" {
  user_id = "<id>"
}
```


## Schema

### Optional

- `domain` (String) The domain in which to look for a user when an email address is specified. Possible values are `LOCAL` and `SSO`.
- `email` (String) User email address.
- `user_id` (String) User ID.

### Read-Only

- `id` (String) User ID.
- `is_account_owner` (Boolean) True if the user is the account owner.
- `roles` (Set of Object) Roles assigned to the user. (see [below for nested schema](#nestedatt--roles))
- `status` (String) User status.

<a id="nestedatt--roles"></a>
### Nested Schema for `roles`

Read-Only:

- `id` (String) Role ID (UUID).
- `name` (String) Role name.
