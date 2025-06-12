variable "admin_password" {
  description = "Password for the cluster admin account."
  type        = string
  sensitive   = true
}

resource "polaris_cdm_registration" "cluster_registration" {
  admin_password          = var.admin_password
  cluster_name            = "my-cluster"
  cluster_node_ip_address = "10.0.100.101"
}
