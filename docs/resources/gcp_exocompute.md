---
page_title: "polaris_gcp_exocompute Resource - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_gcp_exocompute` resource creates an RSC Exocompute configuration
for GCP workloads. This resource should only be used with customer managed
networking. Customer managed networking is used when the `EXOCOMPUTE` feature
of the GCP project was onboarded without the `AUTOMATED_NETWORKING_SETUP`
permission group. If the GCP project was onboarded with the
`AUTOMATED_NETWORKING_SETUP` permission group, RSC will automatically create
and manage the networking resources for Exocompute.

---

# polaris_gcp_exocompute (Resource)


The `polaris_gcp_exocompute` resource creates an RSC Exocompute configuration
for GCP workloads. This resource should only be used with customer managed
networking. Customer managed networking is used when the `EXOCOMPUTE` feature
of the GCP project was onboarded without the `AUTOMATED_NETWORKING_SETUP`
permission group. If the GCP project was onboarded with the
`AUTOMATED_NETWORKING_SETUP` permission group, RSC will automatically create
and manage the networking resources for Exocompute.



## Example Usage

```terraform
resource "polaris_gcp_exocompute" "exocompute" {
  cloud_account_id     = polaris_gcp_project.project.id
  trigger_health_check = true

  regional_config {
    region      = "us-west1"
    subnet_name = "my-vpc-subnet-01"
    vpc_name    = "my-vpc-01"
  }

  regional_config {
    region      = "us-east1"
    subnet_name = "my-vpc-subnet-02"
    vpc_name    = "my-vpc-02"
  }
}
```


## Schema

### Required

- `cloud_account_id` (String) RSC cloud account ID. This is the ID of the `polaris_gcp_project` resource for which the Exocompute service runs. Changing this forces a new resource to be created.
- `regional_config` (Block Set, Min: 1) Regional configuration for the Exocompute service. (see [below for nested schema](#nestedblock--regional_config))

### Optional

- `trigger_health_check` (Boolean) Trigger a health check for the Exocompute configuration. Defaults to `false`.

### Read-Only

- `id` (String) RSC Cloud Account ID (UUID).

<a id="nestedblock--regional_config"></a>
### Nested Schema for `regional_config`

Optional:

- `region` (String) GCP region to run the exocompute service in. Should be specified in the standard GCP style, e.g. `us-east1`.
- `subnet_name` (String) Name of the GCP subnet to run the exocompute service in.
- `vpc_name` (String) Name of the GCP VPC to run the exocompute service in.

## Import

To import the resource, you need to provide the ID of the RSC cloud account for which the Exocompute service is
configured.

Import is supported using the following syntax:


In Terraform v1.5.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `id` attribute, for example:

```terraform
import {
  to = polaris_gcp_exocompute.exocompute
  id = "3084e4c8-dbc0-43a9-97d6-80c5ba2c51d6"
}
```



The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```terraform
% terraform import polaris_gcp_exocompute.exocompute 3084e4c8-dbc0-43a9-97d6-80c5ba2c51d6
```

