# Enable the Cloud Native Protection feature for the EastUS2 region.
resource "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  cloud_native_protection {
    regions = [
      "eastus2",
    ]
    resource_group_name   = "my-resource-group"
    resource_group_region = "eastus2"
  }
}

# Enable the Cloud Native Protection feature for the EastUS2 and the
# WestUS2 regions and the Exocompute feature for the EastUS2 region.
resource "polaris_azure_subscription" "subscription" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  cloud_native_protection {
    regions = [
      "eastus2",
      "westus2",
    ]
    resource_group_name   = "my-west-resource-group"
    resource_group_region = "westus2"
    resource_group_tags = {
      environment = "production"
    }
  }

  exocompute {
    regions = [
      "eastus2",
    ]
    resource_group_name   = "my-east-resource-group"
    resource_group_region = "eastus2"
  }
}

# Using the polaris_azure_permissions data source to inform RSC about
# permission updates for the feature.
data "polaris_azure_permissions" "exocompute" {
  feature = "EXOCOMPUTE"
}

resource "polaris_azure_subscription" "default" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "my-domain.onmicrosoft.com"

  exocompute {
    permissions = data.polaris_azure_permissions.exocompute.id
    regions = [
      "eastus2",
    ]
    resource_group_name   = "my-resource-group"
    resource_group_region = "eastus2"
  }
}
