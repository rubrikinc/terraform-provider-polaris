variable "identity_provider_name" {
  type        = string
  description = "Name of the identity provider."
  default     = "My IdP"
}

variable "group_name" {
  type        = string
  description = "SSO group name."
  default     = "mygroup"
}

resource "polaris_custom_role" "viewer" {
  name        = "Cluster Viewer"
  description = "View clusters"

  permission {
    operation = "VIEW_CLUSTER"
    hierarchy {
      snappable_type = "AllSubHierarchyType"
      object_ids     = ["CLUSTER_ROOT"]
    }
  }
}

data "polaris_identity_provider" "example" {
  name = var.identity_provider_name
}

resource "polaris_sso_group" "example" {
  group_name     = var.group_name
  auth_domain_id = data.polaris_identity_provider.example.identity_provider_id

  role_ids = [
    polaris_custom_role.viewer.id,
  ]
}
