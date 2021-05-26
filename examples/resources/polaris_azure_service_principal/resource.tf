resource "polaris_azure_service_principal" "default" {
    credentials = "${path.module}/service-principal.json"
}
