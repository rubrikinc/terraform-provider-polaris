# RSC does not return the cloud type on import. Supply it in the identity block
# so the imported state is not mislabeled (omit it to default to PUBLIC).
import {
  to = polaris_azure_devops_organization.org
  identity = {
    id    = "a1b2c3d4-1234-4c5b-9abc-0123456789ab"
    cloud = "USGOV"
  }
}
