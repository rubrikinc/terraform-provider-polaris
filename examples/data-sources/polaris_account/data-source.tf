# Output the features enabled for the RSC account.
data "polaris_account" "account" {}

output "features" {
  value = data.polaris_account.account.features
}

# Using the fqdn field from the deployment data source to create an Azure
# AD application.
data "polaris_deployment" "deployment" {}

resource "azuread_application" "app" {
  display_name = "Rubrik Security Cloud Integration"
  web {
    homepage_url = "https://${data.polaris_account.account.fqdn}/setup_azure"
  }
}
