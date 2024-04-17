# Output the IP addresses used by the RSC deployment.
data "polaris_deployment" "deployment" {}

output "ip_addresses" {
  value = data.polaris_deployment.deployment.ip_addresses
}

# Using the account_fqdn field from the deployment data source to create
# an Azure AD application.
data "polaris_deployment" "deployment" {}

resource "azuread_application" "app" {
  display_name = "Rubrik Security Cloud Integration"
  web {
    homepage_url = "https://${data.polaris_deployment.deployment.account_fqdn}/setup_azure"
  }
}
