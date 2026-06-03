# List every Azure resource group visible to RSC across every managed
# subscription.
data "polaris_azure_resource_groups" "all" {}

# Narrow the list to a specific set of subscriptions. Resource group names are
# only unique within a subscription, so keying on name should always also branch
# on subscription_id.
data "polaris_azure_resource_groups" "filtered" {
  subscription_ids = [
    "83ae73a0-4cd2-4f3f-a30d-e56af28caedc",
  ]
}

# Look up a single resource group by exact name within a subscription.
# The provider sends `name` to RSC as a substring filter for server-side
# narrowing, then keeps only entries with an exact name match before returning,
# so this yields zero or one result.
data "polaris_azure_resource_groups" "by_name" {
  subscription_ids = [
    "83ae73a0-4cd2-4f3f-a30d-e56af28caedc",
  ]
  name = "terraform-test"
}

output "terraform_test_rg_id" {
  value = one(data.polaris_azure_resource_groups.by_name.resource_groups[*].id)
}

# Pull just the names of the filtered resource groups.
output "filtered_resource_group_names" {
  value = data.polaris_azure_resource_groups.filtered.resource_groups[*].name
}

# Group resource group names by their parent subscription.
output "resource_groups_by_subscription" {
  value = {
    for rg in data.polaris_azure_resource_groups.all.resource_groups :
    rg.subscription_id => rg.name...
  }
}
