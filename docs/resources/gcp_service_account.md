---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_gcp_service_account Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
---

# polaris_gcp_service_account (Resource)



## Example Usage

```terraform
resource "polaris_gcp_service_account" "default" {
  credentials = "${path.module}/my-project-3f88757a02a4.json"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **credentials** (String) Path to GCP service account key file.

### Optional

- **id** (String) The ID of this resource.
- **name** (String) Service account name in Polaris. If not given the name of the service account key file is used.

