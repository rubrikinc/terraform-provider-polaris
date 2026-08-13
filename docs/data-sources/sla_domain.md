---
page_title: "polaris_sla_domain Data Source - terraform-provider-polaris"
subcategory: ""
description: |-
  
The `polaris_sla_domain` data source is used to access information about RSC SLA
domains. A SLA domain is looked up using either the ID or the name.

---

# polaris_sla_domain (Data Source)


The `polaris_sla_domain` data source is used to access information about RSC SLA
domains. A SLA domain is looked up using either the ID or the name.



## Example Usage

```terraform
# Using SLA domain ID.
data "polaris_sla_domain" "sla_domain" {
  id = "3c1a891a-340c-4b8a-a1ca-adec4d5914e4"
}

# Using SLA domain name.
data "polaris_sla_domain" "sla_domain" {
  name = "bronze"
}
```



## Schema

### Optional

- `id` (String) SLA domain ID (UUID).
- `name` (String) SLA domain name.

### Read-Only

- `archival` (List of Object) Archive snapshots to the specified archival location. (see [below for nested schema](#nestedatt--archival))
- `aws_dynamodb_config` (List of Object) AWS DynamoDB configuration. (see [below for nested schema](#nestedatt--aws_dynamodb_config))
- `aws_rds_config` (List of Object) AWS RDS continuous backups for point-in-time recovery. (see [below for nested schema](#nestedatt--aws_rds_config))
- `azure_blob_config` (List of Object) Azure Blob Storage backup location for scheduled snapshots. (see [below for nested schema](#nestedatt--azure_blob_config))
- `azure_postgres_flexible_server_config` (List of Object) Azure Postgres Flexible Server configuration. (see [below for nested schema](#nestedatt--azure_postgres_flexible_server_config))
- `azure_sql_database_config` (List of Object) Azure SQL Database continuous backups for point-in-time recovery. (see [below for nested schema](#nestedatt--azure_sql_database_config))
- `azure_sql_managed_instance_config` (List of Object) Azure SQL MI log backups. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config))
- `backup_location` (List of Object) Backup locations for AWS S3 object type. (see [below for nested schema](#nestedatt--backup_location))
- `backup_type` (String) Identifies which system manages the SLA's Azure SQL backups: `NATIVE` for a V1 (Azure-managed / long-term retention) SLA, or the Rubrik-managed value for a V2 SLA.
- `daily_schedule` (List of Object) Daily schedule of the SLA Domain. (see [below for nested schema](#nestedatt--daily_schedule))
- `db2_config` (List of Object) Db2 database configuration. (see [below for nested schema](#nestedatt--db2_config))
- `description` (String) SLA domain description.
- `first_full_snapshot` (List of Object) First full snapshot window. (see [below for nested schema](#nestedatt--first_full_snapshot))
- `gcp_cloud_sql_config` (List of Object) GCP Cloud SQL configuration. (see [below for nested schema](#nestedatt--gcp_cloud_sql_config))
- `hourly_schedule` (List of Object) Hourly schedule. (see [below for nested schema](#nestedatt--hourly_schedule))
- `informix_config` (List of Object) Informix database configuration. (see [below for nested schema](#nestedatt--informix_config))
- `local_retention` (List of Object) Local retention. (see [below for nested schema](#nestedatt--local_retention))
- `managed_volume_config` (List of Object) Managed Volume configuration. (see [below for nested schema](#nestedatt--managed_volume_config))
- `minute_schedule` (List of Object) Minute schedule. (see [below for nested schema](#nestedatt--minute_schedule))
- `mongo_config` (List of Object) MongoDB database configuration. (see [below for nested schema](#nestedatt--mongo_config))
- `monthly_schedule` (List of Object) Monthly schedule. (see [below for nested schema](#nestedatt--monthly_schedule))
- `mssql_config` (List of Object) SQL Server database configuration. (see [below for nested schema](#nestedatt--mssql_config))
- `mysqldb_config` (List of Object) MySQL database configuration. (see [below for nested schema](#nestedatt--mysqldb_config))
- `ncd_config` (List of Object) NAS Cloud Direct configuration. (see [below for nested schema](#nestedatt--ncd_config))
- `object_types` (Set of String) Object types which can be protected by the SLA domain.
- `oracle_config` (List of Object) Oracle database configuration. (see [below for nested schema](#nestedatt--oracle_config))
- `postgres_db_cluster_config` (List of Object) Postgres DB Cluster configuration. (see [below for nested schema](#nestedatt--postgres_db_cluster_config))
- `quarterly_schedule` (List of Object) Quarterly schedule. (see [below for nested schema](#nestedatt--quarterly_schedule))
- `replication_spec` (List of Object) Replicate snapshots to the specified region. (see [below for nested schema](#nestedatt--replication_spec))
- `retention_lock` (List of Object) Retention lock. (see [below for nested schema](#nestedatt--retention_lock))
- `sap_hana_config` (List of Object) SAP HANA database configuration. (see [below for nested schema](#nestedatt--sap_hana_config))
- `snapshot_window` (List of Object) Snapshot window. (see [below for nested schema](#nestedatt--snapshot_window))
- `vmware_vm_config` (List of Object) VMware vSphere VM log backups. (see [below for nested schema](#nestedatt--vmware_vm_config))
- `weekly_schedule` (List of Object) Weekly schedule. (see [below for nested schema](#nestedatt--weekly_schedule))
- `yearly_schedule` (List of Object) Yearly schedule. (see [below for nested schema](#nestedatt--yearly_schedule))

<a id="nestedatt--archival"></a>
### Nested Schema for `archival`

Read-Only:

- `archival_location_id` (String) Archival location ID (UUID).
- `archival_location_to_cluster_mapping` (List of Object) Mapping between archival location and Rubrik cluster. (see [below for nested schema](#nestedatt--archival--archival_location_to_cluster_mapping))
- `archival_tiering` (List of Object) (see [below for nested schema](#nestedobjatt--archival--archival_tiering))
- `frequency` (Set of String) Effective snapshot frequencies being archived.
- `threshold` (Number) Threshold specifies the time before archiving the snapshots at the managing location.
- `threshold_unit` (String) Threshold unit specifies the unit of threshold.

<a id="nestedatt--archival--archival_location_to_cluster_mapping"></a>
### Nested Schema for `archival.archival_location_to_cluster_mapping`

Read-Only:

- `archival_location_id` (String) Archival location ID (UUID).
- `cluster_id` (String) Cluster ID (UUID).
- `cluster_name` (String) Cluster name.
- `name` (String) Archival location name.


<a id="nestedobjatt--archival--archival_tiering"></a>
### Nested Schema for `archival.archival_tiering`

Read-Only:

- `cold_storage_class` (String) Cold storage class for tiering.
- `instant_tiering` (Boolean) Enable instant tiering to cold storage.
- `min_accessible_duration_in_seconds` (Number) Minimum duration in seconds that data must remain accessible before tiering.
- `tier_existing_snapshots` (Boolean) Whether to tier existing snapshots to cold storage.



<a id="nestedatt--aws_dynamodb_config"></a>
### Nested Schema for `aws_dynamodb_config`

Read-Only:

- `kms_alias` (String) KMS alias for primary backup.


<a id="nestedatt--aws_rds_config"></a>
### Nested Schema for `aws_rds_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long the backups are kept.
- `log_retention_unit` (String) Log retention unit specifies the unit of the log_retention field.


<a id="nestedatt--azure_blob_config"></a>
### Nested Schema for `azure_blob_config`

Read-Only:

- `archival_location_id` (String) Archival location ID (UUID).


<a id="nestedatt--azure_postgres_flexible_server_config"></a>
### Nested Schema for `azure_postgres_flexible_server_config`

Read-Only:

- `backup_retention_in_days` (Number) Point-in-time restore retention, in days, enforced on the source flexible server.


<a id="nestedatt--azure_sql_database_config"></a>
### Nested Schema for `azure_sql_database_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long, in days, the continuous backups are kept.
- `ltr_config` (List of Object) Long-term retention (LTR) configuration for a V1 (Azure-managed) Azure SQL SLA. (see [below for nested schema](#nestedatt--azure_sql_database_config--ltr_config))

<a id="nestedatt--azure_sql_database_config--ltr_config"></a>
### Nested Schema for `azure_sql_database_config.ltr_config`

Read-Only:

- `monthly_retention` (List of Object) The monthly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_database_config--ltr_config--monthly_retention))
- `weekly_retention` (List of Object) The weekly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_database_config--ltr_config--weekly_retention))
- `yearly_retention` (List of Object) The yearly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_database_config--ltr_config--yearly_retention))

<a id="nestedatt--azure_sql_database_config--ltr_config--monthly_retention"></a>
### Nested Schema for `azure_sql_database_config.ltr_config.monthly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.


<a id="nestedatt--azure_sql_database_config--ltr_config--weekly_retention"></a>
### Nested Schema for `azure_sql_database_config.ltr_config.weekly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.


<a id="nestedatt--azure_sql_database_config--ltr_config--yearly_retention"></a>
### Nested Schema for `azure_sql_database_config.ltr_config.yearly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.
- `week_of_year` (Number) The week of the year (1-52) retained as the yearly backup.



<a id="nestedatt--azure_sql_managed_instance_config"></a>
### Nested Schema for `azure_sql_managed_instance_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long, in days, the log backups are kept.
- `ltr_config` (List of Object) Long-term retention (LTR) configuration for a V1 (Azure-managed) Azure SQL SLA. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config--ltr_config))

<a id="nestedatt--azure_sql_managed_instance_config--ltr_config"></a>
### Nested Schema for `azure_sql_managed_instance_config.ltr_config`

Read-Only:

- `monthly_retention` (List of Object) The monthly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config--ltr_config--monthly_retention))
- `weekly_retention` (List of Object) The weekly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config--ltr_config--weekly_retention))
- `yearly_retention` (List of Object) The yearly Azure SQL long-term retention. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config--ltr_config--yearly_retention))

<a id="nestedatt--azure_sql_managed_instance_config--ltr_config--monthly_retention"></a>
### Nested Schema for `azure_sql_managed_instance_config.ltr_config.monthly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.


<a id="nestedatt--azure_sql_managed_instance_config--ltr_config--weekly_retention"></a>
### Nested Schema for `azure_sql_managed_instance_config.ltr_config.weekly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.


<a id="nestedatt--azure_sql_managed_instance_config--ltr_config--yearly_retention"></a>
### Nested Schema for `azure_sql_managed_instance_config.ltr_config.yearly_retention`

Read-Only:

- `retention` (Number) Retention value in the configured retention unit.
- `retention_unit` (String) Unit for the retention value. One of DAYS, WEEKS, MONTHS or YEARS.
- `week_of_year` (Number) The week of the year (1-52) retained as the yearly backup.



<a id="nestedatt--backup_location"></a>
### Nested Schema for `backup_location`

Read-Only:

- `archival_group_id` (String) Archival group ID (UUID).


<a id="nestedatt--daily_schedule"></a>
### Nested Schema for `daily_schedule`

Read-Only:

- `frequency` (Number) Frequency of snapshots (days).
- `retention` (Number) Retention of snapshots.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--db2_config"></a>
### Nested Schema for `db2_config`

Read-Only:

- `differential_frequency` (Number) Differential backup frequency.
- `differential_frequency_unit` (String) Differential frequency unit.
- `incremental_frequency` (Number) Incremental backup frequency.
- `incremental_frequency_unit` (String) Incremental frequency unit.
- `log_archival_method` (String) Log archival method.
- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--first_full_snapshot"></a>
### Nested Schema for `first_full_snapshot`

Read-Only:

- `duration` (Number) Duration of the first full snapshot window in hours.
- `start_at` (String) Start time of the first full snapshot window.


<a id="nestedatt--gcp_cloud_sql_config"></a>
### Nested Schema for `gcp_cloud_sql_config`

Read-Only:

- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--hourly_schedule"></a>
### Nested Schema for `hourly_schedule`

Read-Only:

- `frequency` (Number) Frequency of snapshots (hours).
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--informix_config"></a>
### Nested Schema for `informix_config`

Read-Only:

- `frequency` (Number) Log backup frequency.
- `frequency_unit` (String) Frequency unit.
- `incremental_frequency` (Number) Incremental backup frequency.
- `incremental_frequency_unit` (String) Incremental frequency unit.
- `incremental_retention` (Number) Incremental backup retention duration.
- `incremental_retention_unit` (String) Incremental retention unit.
- `retention` (Number) Log retention duration.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--local_retention"></a>
### Nested Schema for `local_retention`

Read-Only:

- `retention` (Number) Retention specifies for how long the snapshots are kept.
- `retention_unit` (String) Retention unit specifies the unit of retention.


<a id="nestedatt--managed_volume_config"></a>
### Nested Schema for `managed_volume_config`

Read-Only:

- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--minute_schedule"></a>
### Nested Schema for `minute_schedule`

Read-Only:

- `frequency` (Number) Frequency (minutes).
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--mongo_config"></a>
### Nested Schema for `mongo_config`

Read-Only:

- `frequency` (Number) Log backup frequency.
- `frequency_unit` (String) Frequency unit.
- `retention` (Number) Log retention duration.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--monthly_schedule"></a>
### Nested Schema for `monthly_schedule`

Read-Only:

- `day_of_month` (String) Day of month.
- `frequency` (Number) Frequency.
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--mssql_config"></a>
### Nested Schema for `mssql_config`

Read-Only:

- `frequency` (Number) Log backup frequency.
- `frequency_unit` (String) Frequency unit.
- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--mysqldb_config"></a>
### Nested Schema for `mysqldb_config`

Read-Only:

- `frequency` (Number) Log backup frequency.
- `frequency_unit` (String) Frequency unit.
- `retention` (Number) Log retention duration.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--ncd_config"></a>
### Nested Schema for `ncd_config`

Read-Only:

- `daily_backup_locations` (List of String) Target location UUIDs for daily schedule backups.
- `hourly_backup_locations` (List of String) Target location UUIDs for hourly schedule backups.
- `minutely_backup_locations` (List of String) Target location UUIDs for per-minute schedule backups.
- `monthly_backup_locations` (List of String) Target location UUIDs for monthly schedule backups.
- `quarterly_backup_locations` (List of String) Target location UUIDs for quarterly schedule backups.
- `weekly_backup_locations` (List of String) Target location UUIDs for weekly schedule backups.
- `yearly_backup_locations` (List of String) Target location UUIDs for yearly schedule backups.


<a id="nestedatt--oracle_config"></a>
### Nested Schema for `oracle_config`

Read-Only:

- `frequency` (Number) Log backup frequency.
- `frequency_unit` (String) Frequency unit.
- `host_log_retention` (Number) Host log retention duration for archived redo logs.
- `host_log_retention_unit` (String) Host log retention unit.
- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--postgres_db_cluster_config"></a>
### Nested Schema for `postgres_db_cluster_config`

Read-Only:

- `log_retention` (Number) Log retention duration for Write-Ahead Logging (WAL) logs.
- `log_retention_unit` (String) Log retention unit.


<a id="nestedatt--quarterly_schedule"></a>
### Nested Schema for `quarterly_schedule`

Read-Only:

- `day_of_quarter` (String) Day of quarter.
- `frequency` (Number) Frequency.
- `quarter_start_month` (String) Quarter start month.
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--replication_spec"></a>
### Nested Schema for `replication_spec`

Read-Only:

- `aws_cross_account` (String) Replication target (RSC cloud account ID) for cross account replication.
- `aws_region` (String) AWS region to replicate to.
- `azure_region` (String) Azure region to replicate to.
- `cascading_archival` (List of Object) Cascading archival specifications for replication. (see [below for nested schema](#nestedatt--replication_spec--cascading_archival))
- `local_retention` (List of Object) Local retention on replication target. (see [below for nested schema](#nestedatt--replication_spec--local_retention))
- `replication_pair` (List of Object) Replication pairs specifying source and target clusters. (see [below for nested schema](#nestedatt--replication_spec--replication_pair))
- `retention` (Number) Retention specifies for how long the snapshots are kept.
- `retention_unit` (String) Retention unit specifies the unit of retention.

<a id="nestedatt--replication_spec--cascading_archival"></a>
### Nested Schema for `replication_spec.cascading_archival`

Read-Only:

- `archival_location_id` (String) Archival location ID (UUID) for cascading archival.
- `archival_threshold` (Number) Archival threshold specifies when to archive replicated snapshots.
- `archival_threshold_unit` (String) Archival threshold unit.
- `archival_tiering` (List of Object) Archival tiering specification for cold storage. (see [below for nested schema](#nestedatt--replication_spec--cascading_archival--archival_tiering))
- `frequency` (Set of String) Frequencies for cascading archival.

<a id="nestedatt--replication_spec--cascading_archival--archival_tiering"></a>
### Nested Schema for `replication_spec.cascading_archival.archival_tiering`

Read-Only:

- `cold_storage_class` (String) Cold storage class for tiering.
- `instant_tiering` (Boolean) Enable instant tiering to cold storage.
- `min_accessible_duration_in_seconds` (Number) Minimum duration in seconds that data must remain accessible before tiering.
- `tier_existing_snapshots` (Boolean) Whether to tier existing snapshots to cold storage.


<a id="nestedatt--replication_spec--local_retention"></a>
### Nested Schema for `replication_spec.local_retention`

Read-Only:

- `retention` (Number) Local retention on replication target specifies for how long the snapshots are kept on the replication target before being archived.
- `retention_unit` (String) Local retention unit.

<a id="nestedatt--replication_spec--replication_pair"></a>
### Nested Schema for `replication_spec.replication_pair`

Read-Only:

- `source_cluster` (String) Source cluster ID (UUID).
- `target_cluster` (String) Target cluster ID (UUID).


<a id="nestedatt--retention_lock"></a>
### Nested Schema for `retention_lock`

Read-Only:

- `mode` (String) Retention lock mode.


<a id="nestedatt--sap_hana_config"></a>
### Nested Schema for `sap_hana_config`

Read-Only:

- `differential_frequency` (Number) Differential backup frequency.
- `differential_frequency_unit` (String) Differential frequency unit.
- `incremental_frequency` (Number) Incremental backup frequency.
- `incremental_frequency_unit` (String) Incremental frequency unit.
- `log_retention` (Number) Log retention duration.
- `log_retention_unit` (String) Log retention unit.
- `storage_snapshot_config` (List of Object) SAP HANA storage snapshot configuration. (see [below for nested schema](#nestedobjatt--sap_hana_config--storage_snapshot_config))

<a id="nestedobjatt--sap_hana_config--storage_snapshot_config"></a>
### Nested Schema for `sap_hana_config.storage_snapshot_config`

Read-Only:

- `frequency` (Number) Storage snapshot frequency.
- `frequency_unit` (String) Frequency unit.
- `retention` (Number) Storage snapshot retention.
- `retention_unit` (String) Retention unit.



<a id="nestedatt--snapshot_window"></a>
### Nested Schema for `snapshot_window`

Read-Only:

- `duration` (Number) Duration of the snapshot window in hours.
- `start_at` (String) Start time of the snapshot window.


<a id="nestedatt--vmware_vm_config"></a>
### Nested Schema for `vmware_vm_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long, in seconds, the log backups are kept.


<a id="nestedatt--weekly_schedule"></a>
### Nested Schema for `weekly_schedule`

Read-Only:

- `day_of_week` (String) Day of week.
- `frequency` (Number) Frequency.
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--yearly_schedule"></a>
### Nested Schema for `yearly_schedule`

Read-Only:

- `day_of_year` (String) Day of year.
- `frequency` (Number) Frequency.
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.
- `year_start_month` (String) Year start month.

