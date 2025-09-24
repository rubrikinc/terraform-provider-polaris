data "polaris_azure_permissions" "cloud_native_protection" {
  feature = "CLOUD_NATIVE_PROTECTION"
  permission_groups = [
    "BASIC",
    "EXPORT_AND_RESTORE",
    "FILE_LEVEL_RECOVERY",
  ]
}


data "polaris_azure_permissions" "exocompute" {
  feature = "EXOCOMPUTE"
  permission_groups = [
    "BASIC",
  ]
}

resource "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  cloud_native_protection {
    permissions           = data.polaris_azure_permissions.cloud_native_protection.id
    permission_groups     = data.polaris_azure_permissions.cloud_native_protection.permission_groups
    resource_group_name   = "my-cloud-native-protection-rg"
    resource_group_region = "eastus2"

    regions = [
      "eastus2",
    ]
  }

  exocompute {
    permissions           = data.polaris_azure_permissions.exocompute.id
    permission_groups     = data.polaris_azure_permissions.exocompute.permission_groups
    resource_group_name   = "my-exocompute-rg"
    resource_group_region = "eastus2"

    regions = [
      "eastus2",
    ]
  }
}
