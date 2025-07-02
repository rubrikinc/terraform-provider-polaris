resource "polaris_azure_exocompute_cluster_attachment" "cluster" {
  cluster_name  = "my-aks-cluster"
  exocompute_id = polaris_azure_exocompute.host.id
}
