resource "polaris_azure_subscription" "default" {
    subscription_id   = "31be1bb0-c76c-11eb-9217-afdffe83a002"
    subscription_name = "My Subscription"
    tenant_domain     = "my-domain.onmicrosoft.com"
    regions           = [
        "eastus2",
        "westus2"
    ]
}
