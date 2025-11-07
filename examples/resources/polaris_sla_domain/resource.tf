resource "polaris_sla_domain" "daily" {
  name = "daily"
  description = "Daily SLA Domain"
  object_types = ["AWS_EC2_EBS_OBJECT_TYPE"]
  daily_schedule {
    frequency = 1
    retention = 7
  }
  snapshot_window {
    start_at = "09:00"
    duration = 4
  }
  first_full_snapshot {
    start_at = "Tue, 19:00"
    duration = 5
  }
}


data "polaris_azure_archival_location" "archival_location" {
  name = "my-archival-location"
}

resource "polaris_sla_domain" "weekly" {
  name = "weekly"
  description = "Weekly SLA Domain"
  object_types = ["AZURE_BLOB_OBJECT_TYPE"]
  weekly_schedule {
    day_of_week = "MONDAY"
    frequency = 1
    retention = 4
    retention_unit = "WEEKS"
  }
  azure_blob_config {
    archival_location_id = data.polaris_azure_archival_location.archival_location.id
  }
}



