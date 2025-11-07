---
page_title: "polaris_gcp_project Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_gcp_project` data source is used to access information about a GCP
project added to RSC. A GCP project is looked up using either the GCP project
ID, the GCP project number, the RSC cloud account ID or the name.

-> **Note:** The project name is the name of the GCP project as it appears in
   RSC.

---

# polaris_gcp_project (Data Source)


The `polaris_gcp_project` data source is used to access information about a GCP
project added to RSC. A GCP project is looked up using either the GCP project
ID, the GCP project number, the RSC cloud account ID or the name.

-> **Note:** The project name is the name of the GCP project as it appears in
   RSC.



## Example Usage

```terraform
data "polaris_gcp_project" "project" {
  name = "example"
}
```


## Schema

### Optional

- `id` (String) RSC cloud account ID (UUID).
- `name` (String) GCP project name.
- `project_id` (String) GCP project ID.
- `project_number` (String) GCP project number.

### Read-Only

- `feature` (Set of Object) RSC feature with permission groups and status. (see [below for nested schema](#nestedatt--feature))
- `organization_name` (String) GCP organization name.

<a id="nestedatt--feature"></a>
### Nested Schema for `feature`

Read-Only:

- `name` (String) Feature name.
- `permission_groups` (Set of String) Permission groups for the feature.
- `status` (String) Status of the feature.
