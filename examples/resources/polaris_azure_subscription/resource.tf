# Enable Cloud Native Protection
resource "polaris_azure_subscription" "default" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "mydomain.onmicrosoft.com"

  cloud_native_protection {
    regions = [
      "eastus2",
    ]
  }
}

# Enable Cloud Native Protection and Exocompte. 
resource "polaris_azure_subscription" "default" {
  subscription_id = "31be1bb0-c76c-11eb-9217-afdffe83a002"
  tenant_domain   = "mydomain.onmicrosoft.com"

  cloud_native_protection {
    regions = [
      "eastus2",
      "westus2",
    ]
  }

  exocompute {
    regions = [
      "eastus2",
    ]
  }
}
