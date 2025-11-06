// Copyright 2024 Rubrik, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const dataSourceSLADomainDescription = `
The ´polaris_sla_domain´ data source is used to access information about RSC SLA
domains. A SLA domain is looked up using either the ID or the name.
`

// This data source uses a template for its documentation due to a bug in the TF
// docs generator. Remember to update the template if the documentation for any
// fields are changed.
func dataSourceSLADomain() *schema.Resource {
	return &schema.Resource{
		ReadContext: slaDomainRead,

		Description: description(dataSourceSLADomainDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyName},
				Description:  "SLA domain ID (UUID).",
				ValidateFunc: validation.IsUUID,
			},
			keyDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SLA domain description.",
			},
			keyName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{keyID},
				Description:  "SLA domain name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyObjectTypes: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Object types which can be protected by the SLA domain.",
			},
			keyArchival: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalLocationID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Archival location ID (UUID).",
						},
						keyThreshold: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Threshold specifies the time before archiving the snapshots at the managing location.",
						},
						keyThresholdUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Threshold unit specifies the unit of threshold.",
						},
					},
				},
				Computed:    true,
				Description: "Archive snapshots to the specified archival location.",
			},
			keyAWSDynamoDBConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyKMSAlias: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "KMS alias for primary backup.",
						},
					},
				},
				Computed:    true,
				Description: "AWS DynamoDB configuration.",
			},
			keyAWSRDSConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Log retention specifies for how long the backups are kept.",
						},
						keyLogRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Log retention unit specifies the unit of the log_retention field.",
						},
					},
				},
				Computed:    true,
				Description: "AWS RDS continuous backups for point-in-time recovery.",
			},
			keyAzureBlobConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalLocationID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Archival location ID (UUID).",
						},
					},
				},
				Computed:    true,
				Description: "Azure Blob Storage backup location for scheduled snapshots.",
			},
			keyAzureSQLDatabaseConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Log retention specifies for how long, in days, the continuous backups are kept.",
						},
					},
				},
				Computed:    true,
				Description: "Azure SQL Database continuous backups for point-in-time recovery.",
			},
			keyAzureSQLManagedInstanceConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Log retention specifies for how long, in days, the log backups are kept.",
						},
					},
				},
				Computed:    true,
				Description: "Azure SQL MI log backups.",
			},
			keyBackupLocation: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalGroupID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Archival group ID (UUID).",
						},
					},
				},
				Computed:    true,
				Description: "Backup locations for AWS S3 object type.",
			},
			keyDailySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency of snapshots (days).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention of snapshots.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Daily schedule of the SLA Domain.",
			},
			keyHourlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency of snapshots (hours).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Hourly schedule.",
			},
			keyMinuteSchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (minutes).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
					},
				},
				Computed:    true,
				Description: "Minute schedule.",
			},
			keyMonthlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (months).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of month.",
						},
					},
				},
				Computed:    true,
				Description: "Monthly schedule.",
			},
			keyQuarterlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (quarters).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfQuarter: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of quarter.",
						},
						keyQuarterStartMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Quarter start month.",
						},
					},
				},
				Computed:    true,
				Description: "Quarterly schedule.",
			},
			keyWeeklySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (weeks).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfWeek: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of week.",
						},
					},
				},
				Computed:    true,
				Description: "Weekly schedule.",
			},
			keyYearlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frequency (years).",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit.",
						},
						keyDayOfYear: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Day of year.",
						},
						keyYearStartMonth: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Year start month.",
						},
					},
				},
				Computed:    true,
				Description: "Yearly schedule.",
			},
			keyFirstFullSnapshot: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDuration: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Duration of the first full snapshot window in hours.",
						},
						keyStartAt: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Start time of the first full snapshot window.",
						},
					},
				},
				Computed:    true,
				Description: "First full snapshot window.",
			},
			keyLocalRetention: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention specifies for how long the snapshots are kept.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit specifies the unit of retention.",
						},
					},
				},
				Computed:    true,
				Description: "Local retention.",
			},
			keyReplicationSpec: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyAWSRegion: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "AWS region to replicate to.",
						},
						keyAWSCrossAccount: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Replication target (RSC cloud account ID) for cross account replication.",
						},
						keyAzureRegion: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Azure region to replicate to.",
						},
						keyRetention: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Retention specifies for how long the snapshots are kept.",
						},
						keyRetentionUnit: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention unit specifies the unit of retention.",
						},
					},
				},
				Computed:    true,
				Description: "Replicate snapshots to the specified region.",
			},
			keyRetentionLock: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyMode: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Retention lock mode.",
						},
					},
				},
				Computed:    true,
				Description: "Retention lock.",
			},
			keySnapshotWindow: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDuration: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Duration of the snapshot window in hours.",
						},
						keyStartAt: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Start time of the snapshot window.",
						},
					},
				},
				Computed:    true,
				Description: "Snapshot window.",
			},
		},
	}
}

func slaDomainRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "slaDomainRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	var slaDomain gqlsla.Domain
	if id := d.Get(keyID).(string); id != "" {
		id, err := uuid.Parse(id)
		if err != nil {
			return diag.FromErr(err)
		}
		slaDomain, err = sla.Wrap(client).DomainByID(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		slaDomain, err = sla.Wrap(client).DomainByName(ctx, d.Get(keyName).(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(keyName, slaDomain.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, slaDomain.Description); err != nil {
		return diag.FromErr(err)
	}

	objectTypes := &schema.Set{F: schema.HashString}
	for _, objectType := range slaDomain.ObjectTypes {
		objectTypes.Add(string(objectType))
	}
	if err := d.Set(keyObjectTypes, objectTypes); err != nil {
		return diag.FromErr(err)
	}

	// Set snapshot schedules
	if err := d.Set(keyDailySchedule, toDailySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyHourlySchedule, toHourlySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyMinuteSchedule, toMinuteSchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyMonthlySchedule, toMonthlySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyQuarterlySchedule, toQuarterlySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyWeeklySchedule, toWeeklySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyYearlySchedule, toYearlySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}

	// Set archival configuration - transform the archival specs
	var archivalSpecs []gqlsla.ArchivalSpec
	for _, archivalSpec := range slaDomain.ArchivalSpecs {
		groupID, err := uuid.Parse(archivalSpec.StorageSetting.ID)
		if err != nil {
			return diag.FromErr(err)
		}
		archivalSpecs = append(archivalSpecs, gqlsla.ArchivalSpec{
			GroupID:       groupID,
			Frequencies:   frequenciesFromSchedule(slaDomain.SnapshotSchedule),
			Threshold:     archivalSpec.Threshold,
			ThresholdUnit: archivalSpec.ThresholdUnit,
		})
	}
	archival, err := toArchival(archivalSpecs, d.Get(keyArchival).([]any))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyArchival, archival); err != nil {
		return diag.FromErr(err)
	}

	// Set cloud-specific configurations
	if err := d.Set(keyAWSDynamoDBConfig, toAWSDynamoDBConfig(slaDomain.ObjectSpecificConfigs.AWSDynamoDBConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyAWSRDSConfig, toAWSRDSConfig(slaDomain.ObjectSpecificConfigs.AWSRDSConfig)); err != nil {
		return diag.FromErr(err)
	}

	// Set Azure Blob config
	var azureBlobConfig []any
	if slaDomain.ObjectSpecificConfigs.AzureBlobConfig != nil {
		azureBlobConfig = []any{map[string]any{
			keyArchivalLocationID: slaDomain.ObjectSpecificConfigs.AzureBlobConfig.BackupLocationID.String(),
		}}
	}
	if err := d.Set(keyAzureBlobConfig, azureBlobConfig); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(keyAzureSQLDatabaseConfig, toAzureSQLConfig(slaDomain.ObjectSpecificConfigs.AzureSQLDatabaseDBConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyAzureSQLManagedInstanceConfig, toAzureSQLConfig(slaDomain.ObjectSpecificConfigs.AzureSQLManagedInstanceDBConfig)); err != nil {
		return diag.FromErr(err)
	}

	// Set backup locations
	backupLocations, err := toBackupLocations(slaDomain, d.Get(keyBackupLocation).([]any))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBackupLocation, backupLocations); err != nil {
		return diag.FromErr(err)
	}

	// Set snapshot window
	snapshotWindow, err := toSnapshotWindow(slaDomain.BackupWindows)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySnapshotWindow, snapshotWindow); err != nil {
		return diag.FromErr(err)
	}

	// Set first full snapshot
	firstFullSnapshot, err := toSnapshotWindow(slaDomain.FirstFullBackupWindows)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyFirstFullSnapshot, firstFullSnapshot); err != nil {
		return diag.FromErr(err)
	}

	// Set replication spec - transform the replication specs
	var replicationSpecs []gqlsla.ReplicationSpec
	for _, spec := range slaDomain.ReplicationSpecs {
		replicationSpecs = append(replicationSpecs, gqlsla.ReplicationSpec{
			AWSRegion:   spec.AWSRegion,
			AWSAccount:  spec.AWS.AccountID,
			AzureRegion: spec.AzureRegion,
			RetentionDuration: &gqlsla.RetentionDuration{
				Duration: spec.RetentionDuration.Duration,
				Unit:     spec.RetentionDuration.Unit,
			},
		})
	}
	if err := d.Set(keyReplicationSpec, toReplicationSpec(replicationSpecs)); err != nil {
		return diag.FromErr(err)
	}

	// Set local retention
	if slaDomain.LocalRetentionLimit != nil {
		if err := d.Set(keyLocalRetention, toLocalRetention(&gqlsla.RetentionDuration{
			Duration: slaDomain.LocalRetentionLimit.Duration,
			Unit:     slaDomain.LocalRetentionLimit.Unit,
		})); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(slaDomain.ID.String())
	return nil
}
