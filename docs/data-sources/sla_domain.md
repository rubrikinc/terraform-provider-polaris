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
- `azure_sql_database_config` (List of Object) Azure SQL Database continuous backups for point-in-time recovery. (see [below for nested schema](#nestedatt--azure_sql_database_config))
- `azure_sql_managed_instance_config` (List of Object) Azure SQL MI log backups. (see [below for nested schema](#nestedatt--azure_sql_managed_instance_config))
- `backup_location` (List of Object) Backup locations for AWS S3 object type. (see [below for nested schema](#nestedatt--backup_location))
- `daily_schedule` (List of Object) Daily schedule of the SLA Domain. (see [below for nested schema](#nestedatt--daily_schedule))
- `description` (String) SLA domain description.
- `first_full_snapshot` (List of Object) First full snapshot window. (see [below for nested schema](#nestedatt--first_full_snapshot))
- `hourly_schedule` (List of Object) Hourly schedule. (see [below for nested schema](#nestedatt--hourly_schedule))
- `local_retention` (List of Object) Local retention. (see [below for nested schema](#nestedatt--local_retention))
- `minute_schedule` (List of Object) Minute schedule. (see [below for nested schema](#nestedatt--minute_schedule))
- `monthly_schedule` (List of Object) Monthly schedule. (see [below for nested schema](#nestedatt--monthly_schedule))
- `object_types` (Set of String) Object types which can be protected by the SLA domain.
- `quarterly_schedule` (List of Object) Quarterly schedule. (see [below for nested schema](#nestedatt--quarterly_schedule))
- `replication_spec` (List of Object) Replicate snapshots to the specified region. (see [below for nested schema](#nestedatt--replication_spec))
- `retention_lock` (List of Object) Retention lock. (see [below for nested schema](#nestedatt--retention_lock))
- `snapshot_window` (List of Object) Snapshot window. (see [below for nested schema](#nestedatt--snapshot_window))
- `weekly_schedule` (List of Object) Weekly schedule. (see [below for nested schema](#nestedatt--weekly_schedule))
- `yearly_schedule` (List of Object) Yearly schedule. (see [below for nested schema](#nestedatt--yearly_schedule))

<a id="nestedatt--archival"></a>
### Nested Schema for `archival`

Read-Only:

- `archival_location_id` (String) Archival location ID (UUID).
- `threshold` (Number) Threshold specifies the time before archiving the snapshots at the managing location.
- `threshold_unit` (String) Threshold unit specifies the unit of threshold.


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


<a id="nestedatt--azure_sql_database_config"></a>
### Nested Schema for `azure_sql_database_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long, in days, the continuous backups are kept.


<a id="nestedatt--azure_sql_managed_instance_config"></a>
### Nested Schema for `azure_sql_managed_instance_config`

Read-Only:

- `log_retention` (Number) Log retention specifies for how long, in days, the log backups are kept.


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


<a id="nestedatt--first_full_snapshot"></a>
### Nested Schema for `first_full_snapshot`

Read-Only:

- `duration` (Number) Duration of the first full snapshot window in hours.
- `start_at` (String) Start time of the first full snapshot window.


<a id="nestedatt--hourly_schedule"></a>
### Nested Schema for `hourly_schedule`

Read-Only:

- `frequency` (Number) Frequency of snapshots (hours).
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--local_retention"></a>
### Nested Schema for `local_retention`

Read-Only:

- `retention` (Number) Retention specifies for how long the snapshots are kept.
- `retention_unit` (String) Retention unit specifies the unit of retention.


<a id="nestedatt--minute_schedule"></a>
### Nested Schema for `minute_schedule`

Read-Only:

- `frequency` (Number) Frequency (minutes).
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


<a id="nestedatt--monthly_schedule"></a>
### Nested Schema for `monthly_schedule`

Read-Only:

- `day_of_month` (String) Day of month.
- `frequency` (Number) Frequency.
- `retention` (Number) Retention.
- `retention_unit` (String) Retention unit.


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
- `replication_pair` (List of Object) Replication pairs specifying source and target clusters. (see [below for nested schema](#nestedatt--replication_spec--replication_pair))
- `retention` (Number) Retention specifies for how long the snapshots are kept.
- `retention_unit` (String) Retention unit specifies the unit of retention.

<a id="nestedatt--replication_spec--replication_pair"></a>
### Nested Schema for `replication_spec.replication_pair`

Read-Only:

- `source_cluster` (String) Source cluster ID (UUID).
- `target_cluster` (String) Target cluster ID (UUID).


<a id="nestedatt--retention_lock"></a>
### Nested Schema for `retention_lock`

Read-Only:

- `mode` (String) Retention lock mode.


<a id="nestedatt--snapshot_window"></a>
### Nested Schema for `snapshot_window`

Read-Only:

- `duration` (Number) Duration of the snapshot window in hours.
- `start_at` (String) Start time of the snapshot window.


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

