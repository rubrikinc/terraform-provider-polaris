resource "polaris_azure_subscription" "default" {
    subscription_id   = "8fa81a5e-a236-4a73-8e28-e1dcf863c56d"
    subscription_name = "Trinity-FDSE"
    tenant_domain     = "rubriktrinity.onmicrosoft.com"
    regions           = ["eastus2"]
}
