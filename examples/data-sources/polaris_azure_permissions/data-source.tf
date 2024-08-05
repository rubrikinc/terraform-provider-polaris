# Permissions required for the Cloud Native Protection RSC feature.
data "polaris_azure_permissions" "cloud_native_protection" {
  feature = "CLOUD_NATIVE_PROTECTION"
}

# Permissions required for the Exocompute RSC feature. The subscription
# is set up to notify RSC when the permissions are updated for the feature.
data "polaris_azure_permissions" "exocompute" {
  feature = "EXOCOMPUTE"
}

resource "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  exocompute {
    permissions = data.polaris_azure_permissions.exocompute.id
    regions = [
      "eastus2",
    ]
    resource_group_name   = "my-east-resource-group"
    resource_group_region = "eastus2"
  }
}
