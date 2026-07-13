# Basic daily SLA domain with snapshot windows
# - Daily backup schedule with 7-day retention
# - Snapshot window configuration (starts at 9 AM, 4-hour duration)
# - First full snapshot scheduling (Tuesday at 7 PM, 5-hour duration)
resource "polaris_sla_domain" "daily" {
  name         = "daily"
  description  = "Daily SLA Domain"
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


# Weekly SLA domain with Azure Blob archival
# - Weekly backup schedule (every Monday) with 4-week retention
# - Azure Blob-specific configuration with archival location
# - Using a data source to reference an existing archival location
data "polaris_azure_archival_location" "archival_location" {
  name = "my-archival-location"
}

resource "polaris_sla_domain" "weekly" {
  name         = "weekly"
  description  = "Weekly SLA Domain"
  object_types = ["AZURE_BLOB_OBJECT_TYPE"]
  weekly_schedule {
    day_of_week    = "MONDAY"
    frequency      = 1
    retention      = 4
    retention_unit = "WEEKS"
  }
  azure_blob_config {
    archival_location_id = data.polaris_azure_archival_location.archival_location.id
  }
}

# Advanced SLA domain with replication and cascading archival
# - Daily backup schedule with 7-day retention
# - Cross-cluster replication from mycluster2 to mycluster1
# - Local retention on the target cluster (7 days)
# - Cascading archival to a data center archival location after 7 days
# - Archival tiering with instant tiering to Azure Archive storage
# - Minimum accessible duration of 1 day (86400 seconds)
data "polaris_sla_source_cluster" "mycluster1" {
  name = "MY-CLUSTER-1"
}

data "polaris_sla_source_cluster" "mycluster2" {
  name = "MY-CLUSTER-2"
}

data "polaris_data_center_archival_location" "myarchivallocation" {
  cluster_id = data.polaris_sla_source_cluster.mycluster1.id
  name       = "My Archival Location"
}

resource "polaris_sla_domain" "with_cascading_archival" {
  name         = "with-cascading-archival"
  description  = "SLA Domain with replication and cascading archival"
  object_types = ["VSPHERE_OBJECT_TYPE"]

  daily_schedule {
    frequency      = 1
    retention      = 7
    retention_unit = "DAYS"
  }

  replication_spec {
    retention      = 7
    retention_unit = "DAYS"

    local_retention {
      retention      = 7
      retention_unit = "DAYS"
    }

    replication_pair {
      source_cluster = data.polaris_sla_source_cluster.mycluster2.id
      target_cluster = data.polaris_sla_source_cluster.mycluster1.id
    }

    cascading_archival {
      archival_location_id    = data.polaris_data_center_archival_location.myarchivallocation.id
      archival_threshold      = 7
      archival_threshold_unit = "DAYS"
      frequency               = ["DAYS"]

      archival_tiering {
        instant_tiering                    = true
        cold_storage_class                 = "AZURE_ARCHIVE"
        min_accessible_duration_in_seconds = 86400
        tier_existing_snapshots            = false
      }
    }
  }
}

# Azure SQL Database V2 (Rubrik-managed) SLA
# - Requires the CNP_AZURE_SQL_SLA_REVAMP feature
# - Rubrik-managed backups: a snapshot schedule plus a backup location
# - backup_location replaces the top-level archival block for Azure SQL
resource "polaris_sla_domain" "azure_sql_v2" {
  name         = "azure-sql-v2"
  description  = "Rubrik-managed Azure SQL Database SLA"
  object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

  hourly_schedule {
    frequency      = 1
    retention      = 1
    retention_unit = "DAYS"
  }

  azure_sql_database_config {
    log_retention = 7
  }

  backup_location {
    archival_group_id = data.polaris_azure_archival_location.archival_location.id
  }
}

# Azure SQL Database V1 (Azure-managed / long-term retention) SLA
# - Requires the CNP_AZURE_SQL_SLA_REVAMP feature
# - Azure-managed backups: ltr_config only, with no schedule and no backup location
resource "polaris_sla_domain" "azure_sql_v1" {
  name         = "azure-sql-v1"
  description  = "Azure-managed (LTR) Azure SQL Database SLA"
  object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

  azure_sql_database_config {
    log_retention = 7
    ltr_config {
      weekly_retention {
        retention      = 4
        retention_unit = "WEEKS"
      }
      monthly_retention {
        retention      = 12
        retention_unit = "MONTHS"
      }
      yearly_retention {
        retention      = 7
        retention_unit = "YEARS"
        week_of_year   = 1
      }
    }
  }
}