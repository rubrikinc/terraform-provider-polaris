---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_aws_account Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
---

# polaris_aws_account (Resource)



## Example Usage

```terraform
resource "polaris_aws_account" "default" {
  profile = "default"
  regions = [
    "us-east-2",
    "us-west-2"
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **profile** (String) AWS named profile.
- **regions** (Set of String) Regions that Polaris will monitor for instances to automatically protect.

### Optional

- **delete_snapshots_on_destroy** (Boolean) Should snapshots be deleted when the resource is destroyed.
- **exocompute** (Block List, Max: 1) Enable the exocompute feature for the account. (see [below for nested schema](#nestedblock--exocompute))
- **id** (String) The ID of this resource.
- **name** (String) Account name in Polaris. If not given the name is taken from AWS Organizations or, if the required permissions are missing, is derived from the AWS account ID and the named profile.

<a id="nestedblock--exocompute"></a>
### Nested Schema for `exocompute`

Required:

- **regions** (Set of String) Regions to enable the exocompute feature in.

