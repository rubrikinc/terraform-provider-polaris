---
page_title: "polaris_gcp_project Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
---

# polaris_gcp_project (Resource)




## Example Usage

```terraform
# With service account key file
resource "polaris_gcp_project" "project" {
  credentials = "${path.module}/my-project-3f88757a02a4.json"
}

# Without service account key file
resource "polaris_gcp_project" "project" {
  project        = "my-project"
  project_number = 123456789012
}
```


## Schema

### Required

- `cloud_native_protection` (Block List, Min: 1, Max: 1) Enable the Cloud Native Protection feature for the GCP project. (see [below for nested schema](#nestedblock--cloud_native_protection))

### Optional

- `credentials` (String) Path to GCP service account key file.
- `delete_snapshots_on_destroy` (Boolean) Should snapshots be deleted when the resource is destroyed.
- `organization_name` (String) Organization name.
- `permissions_hash` (String) Signals that the permissions has been updated.
- `project` (String) Project id.
- `project_name` (String) Project name.
- `project_number` (String) Project number.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--cloud_native_protection"></a>
### Nested Schema for `cloud_native_protection`

Read-Only:

- `status` (String) Status of the Cloud Native Protection feature.

## Import

Only projects using the global service account can be imported.

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_gcp_project.project
  id = "2689a6f0-41a5-4d7a-ba7f-ee591bb43e4a"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_gcp_project.project 2689a6f0-41a5-4d7a-ba7f-ee591bb43e4a
```

