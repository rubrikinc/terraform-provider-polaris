resource "polaris_cdm_bootstrap_cces_azure" "default" {
  admin_email            = "admin@example.org"
  admin_password         = "password"
  cluster_name           = "my-cluster"
  cluster_nodes          = {
    "my-cluster-node-1" = "10.1.100.100",
    "my-cluster-node-2" = "10.1.100.101",
    "my-cluster-node-3" = "10.1.100.102",
  }
  container_name         = "my-cluster"
  dns_search_domain      = ["example.org"]
  dns_name_servers       = ["10.1.150.100", "10.1.150.200"]
  enable_immutability    = true
  management_gateway     = "10.1.100.1"
  management_subnet_mask = "255.255.255.0"
  ntp_server1_name       = "10.1.200.100"
  ntp_server2_name       = "10.1.200.200"
}
