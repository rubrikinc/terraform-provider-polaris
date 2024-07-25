# Using the archival location ID.
data "polaris_azure_archival_location" "archival_location" {
  id = "db34f042-79ea-48b1-bab8-c40dfbf2ab82"
}

# Using the archival location name.
data "polaris_azure_archival_location" "archival_location" {
  name = "my-archival-location"
}
