# With service principal file.
resource "polaris_azure_service_principal" "default" {
  credentials = "${path.module}/service-principal.json"
}

# With service principal created with the Azure SDK using the --sdk-auth
# parameter
resource "polaris_azure_service_principal" "default" {
  sdk_auth = "${path.module}/sdk-service-principal.json"
}

# Without service principal file.
resource "polaris_azure_service_principal" "default" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My App"
  app_secret    = "H12w-c.g3mjk47iz1qyFltv~zqD9m4cpK_"
  tenant_domain = "my-domain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
}
