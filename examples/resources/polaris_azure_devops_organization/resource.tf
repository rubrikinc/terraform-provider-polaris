# Register the customer application for the Azure DevOps use case. A tenant that
# also uses cloud native protection declares a second service principal with the
# default use case.
resource "polaris_azure_service_principal" "devops" {
  app_id        = "25c2b42a-c76b-11eb-9767-6ff6b5b7e72b"
  app_name      = "My DevOps App"
  app_secret    = "<my-apps-secret>"
  tenant_domain = "mydomain.onmicrosoft.com"
  tenant_id     = "2bfdaef8-c76b-11eb-8d3d-4706c14a88f0"
  use_case      = "AZURE_DEVOPS"
}

# Look up the permissions RSC requires for the feature. Wire its id into the
# feature's permissions field to track permission changes.
data "polaris_azure_devops_permissions" "repo" {
  feature           = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
  permission_groups = ["BASIC", "RECOVERY"]
}

# Generate the onboarding script for the organization.
data "polaris_azure_devops_script" "onboard" {
  tenant_domain  = polaris_azure_service_principal.devops.tenant_domain
  org_native_ids = ["my-org"]

  feature {
    name              = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
  }
}

# Run the generated script against the organization out of band before applying
# the resource below — the provider does not run it. See the
# polaris_azure_devops_script data source for how to run it.

# Onboard the organization to RSC using Rubrik-hosted exocompute and Rubrik
# Cloud Vault storage. cloud is optional and defaults to PUBLIC.
resource "polaris_azure_devops_organization" "org" {
  native_id            = "my-org"
  tenant_domain        = polaris_azure_service_principal.devops.tenant_domain
  exocompute_host_type = "RUBRIK_HOST"
  storage_type         = "RCV"
  exocompute_region    = "eastus"

  feature {
    name              = "AZURE_DEVOPS_REPOSITORY_PROTECTION"
    permission_groups = ["BASIC", "RECOVERY"]
    permissions       = data.polaris_azure_devops_permissions.repo.id
  }

  depends_on = [polaris_azure_service_principal.devops]
}
