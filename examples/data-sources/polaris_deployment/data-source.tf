# Using the account_fqdn field from the deployment data source to create
# an Azure AD application.
data "polaris_deployment" "deployment" {}

resource "azuread_application" "app" {
  display_name = "Rubrik Security Cloud Integration"
  web {
    homepage_url = "https://${data.polaris_deployment.deployment.account_fqdn}/setup_azure"
  }
}
