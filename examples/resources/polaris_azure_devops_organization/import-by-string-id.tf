# The plain string ID form defaults cloud to PUBLIC. Use import-by-identity.tf
# instead to onboard a non-public organization.
import {
  to = polaris_azure_devops_organization.org
  id = "a1b2c3d4-1234-4c5b-9abc-0123456789ab"
}
