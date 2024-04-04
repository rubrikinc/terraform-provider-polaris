# Permissions required for the Cloud Native Protection RSC feature.
data "polaris_azure_permissions" "default" {
  features = [
    "CLOUD_NATIVE_PROTECTION",
  ]
}

# Permissions required for the Cloud Native Protection and Exocompute
# RSC features. The polaris_azure_service_principal is set up to notify
# RSC when the permissions are updated.
data "polaris_azure_permissions" "default" {
  features = [
    "CLOUD_NATIVE_PROTECTION",
    "EXOCOMPUTE"
  ]
}

resource "polaris_azure_service_principal" "default" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My App"
  app_secret    = "<my-app-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
  permissions   = data.polaris_azure_permissions.default.id
}
