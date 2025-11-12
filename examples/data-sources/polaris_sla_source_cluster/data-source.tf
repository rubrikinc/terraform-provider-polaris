# Look up SLA source cluster by name.
data "polaris_sla_source_cluster" "cluster" {
  name = "my-cluster"
}

output "cluster_id" {
  value = data.polaris_sla_source_cluster.cluster.id
}

