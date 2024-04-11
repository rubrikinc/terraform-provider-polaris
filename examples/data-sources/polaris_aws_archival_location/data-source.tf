# Using the archival location ID.
data "polaris_aws_archival_location" "location" {
  archival_location_id = "db34f042-79ea-48b1-bab8-c40dfbf2ab82"
}

# Using the name.
data "polaris_aws_archival_location" "location" {
  name = "my-archival-location"
}
