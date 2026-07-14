# List every Azure Native Resource Group RSC knows about, across all
# subscriptions it manages.
data "polaris_objects" "all_resource_groups" {
  object_type = "AzureNativeResourceGroup"
}

output "resource_group_names" {
  value = data.polaris_objects.all_resource_groups.objects[*].name
}

# Scope the search to a single subscription.
data "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
}

data "polaris_objects" "resource_groups_in_subscription" {
  object_type     = "AzureNativeResourceGroup"
  subscription_id = data.polaris_azure_subscription.subscription.id
}
