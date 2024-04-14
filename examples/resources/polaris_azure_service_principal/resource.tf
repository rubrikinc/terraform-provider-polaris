# With custom service principal file.
resource "polaris_azure_service_principal" "default" {
  credentials   = "${path.module}/service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}

# With a service principal created using the Azure SDK and the
# --sdk-auth parameter.
resource "polaris_azure_service_principal" "default" {
  sdk_auth      = "${path.module}/sdk-service-principal.json"
  tenant_domain = "mydomain.onmicrosoft.com"
}

# Without a service principal file.
resource "polaris_azure_service_principal" "default" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My App"
  app_secret    = "<my-apps-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
}

# Using the polaris_azure_permissions data source to inform RSC
# about permission updates.
data "polaris_azure_permissions" "cnp" {
  features = [
    "CLOUD_NATIVE_PROTECTION",
  ]
}

resource "polaris_azure_service_principal" "default" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My App"
  app_secret    = "<my-app-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
  permissions   = data.polaris_azure_permissions.cnp.id
}
