data "polaris_sla_source_cluster" "mycluster1" {
  name = "MY-CLUSTER-1"
}

# Look up an archival location by cluster ID and name.
data "polaris_data_center_archival_location" "archival_location" {
  cluster_id = data.polaris_sla_source_cluster.mycluster1.id
  name       = "my-archival-location"
}

