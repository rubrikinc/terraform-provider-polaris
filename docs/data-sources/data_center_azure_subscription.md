---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "polaris_data_center_azure_subscription Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  The polaris_data_center_azure_subscription data source is used to access
  information about an Azure data center subscription added to RSC. A data center
  subscription is looked up using the name.
  -> Note: Data center subscriptions and cloud native subscriptions are
  different and cannot be used interchangeably.
  -> Note: The name is the name of the data center subscription as it appears
  in RSC.
---

# polaris_data_center_azure_subscription (Data Source)

The `polaris_data_center_azure_subscription` data source is used to access
information about an Azure data center subscription added to RSC. A data center
subscription is looked up using the name.

-> **Note:** Data center subscriptions and cloud native subscriptions are
   different and cannot be used interchangeably.

-> **Note:** The name is the name of the data center subscription as it appears
   in RSC.

## Example Usage

```terraform
data "polaris_data_center_azure_subscription" "archival" {
  name = "archival-subscription"
}

output "cloud_account_id" {
  value = data.polaris_data_center_azure_subscription.archival.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Data center subscription name.

### Read-Only

- `connection_status` (String) Connection status.
- `description` (String) Data center subscription description.
- `id` (String) RSC data center cloud account ID (UUID).
- `subscription_id` (String) Azure subscription ID (UUID).
- `tenant_id` (String) Azure tenant ID (UUID).
