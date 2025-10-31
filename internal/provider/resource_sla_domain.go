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
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"
	gqlazure "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/azure"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/sla"
)

const resourceSLADomainDescription = `
The ´polaris_sla_domain´ resource is used to manage RSC global SLA Domains. SLA
Domain defines how you want to take snapshots of objects like virtual machines,
databases, SaaS apps and cloud objects. An SLA Domain defines frequency,
retention, archival and replication.

-> Enabling Instant Archive can increase bandwidth usage and archival storage
   requirements.

-> The hourly retention for snapshots of cloud-native workloads must be a
   multiple of 24.

-> For workloads backed up on a Rubrik cluster, snapshots are scheduled using
   the time zone of that Rubrik cluster. For workloads backed up in the cloud,
   snapshots are scheduled using the UTC time zone.

---



Frequency

This defines when and how often snapshots are taken. This could be interval-based (days, hours, minutes) or calendar-based (a day of each month).

Retention

This defines how long the snapshot is kept on the Rubrik cluster.

Before You Start: To archive snapshots, make sure you’ve added archival locations so that they’re available for selection.

To avoid early deletion fees, retain snapshots in cool tier archival locations for at least 30 days.

Retention lock: https://docs.rubrik.com/en-us/saas/saas/retention_locked_sla_domain.html

---

For Azure SQL Database:
	"For Azure SQL Database, archival is mandatory and the backups will be instantly archived. " +
	"These frequencies and retentions apply to archived snapshots of the Azure SQL database. " +
	"You can configure continuous backups in the next step. " +
	"To avoid early deletion fees, retain snapshots in cool tier archival locations for at least 30 days. " +
	"Archiving starts immediately. The archival location retains snapshots for ",
`

func resourceSLADomain() *schema.Resource {
	return &schema.Resource{
		CreateContext: newSLADomainMutator("create"),
		ReadContext:   readSLADomain,
		UpdateContext: newSLADomainMutator("update"),
		DeleteContext: deleteSLADomain,

		Description: description(resourceSLADomainDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SLA Domain ID (UUID).",
			},
			keyApplyChangesToExistingSnapshots: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Apply changes to existing snapshots when updating the SLA domain.",
			},
			keyApplyChangesToNonPolicySnapshots: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Apply changes to non-policy snapshots when updating the SLA domain.",
			},
			keyArchival: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalLocationID: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Archival location ID (UUID).",
							ValidateFunc: validation.IsUUID,
						},
						keyThreshold: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
							Description: "Threshold specifies the time before archiving the snapshots at the " +
								"managing location. The archival location retains the snapshots according to the SLA " +
								"Domain schedule.",
							ValidateFunc: validation.IntAtLeast(0),
						},
						keyThresholdUnit: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlsla.Days),
							Description: "Threshold unit specifies the unit of `threshold`. Possible values are " +
								"`DAYS`, `WEEKS`, `MONTHS` and `YEARS`. Default value is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional: true,
				Description: "Archive snapshots to the specified archival location. Note, if `instant_archive` is " +
					"enabled, `threshold` and `threshold_unit` are ignored.",
			},
			keyAWSRDSConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention specifies for how long the backups are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlsla.Days),
							Description: "Log retention unit specifies the unit of the `log_retention` field. " +
								"Possible values are `DAYS`, `WEEKS`, `MONTHS` and `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional: true,
				MaxItems: 1,
				Description: "AWS RDS continuous backups for point-in-time recovery. If continuous backup isn't " +
					"specified, AWS provides 1 day of continuous backup by default for Aurora databases, which can " +
					"be changed but not disable.",
			},
			keyAzureBlobConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalLocationID: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Archival location ID (UUID).",
							ValidateFunc: validation.IsUUID,
						},
					},
				},
				Optional: true,
				MaxItems: 1,
				Description: "Azure Blob Storage backup location for scheduled snapshots. To avoid early deletion " +
					"fees, retain snapshots in cool tier archival locations for at least 30 days.",
			},
			keyAzureSQLDatabaseConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:     schema.TypeInt,
							Required: true,
							Description: "Log retention specifies for how long, in days, the continuous backups are " +
								"kept.",
							ValidateFunc: validation.IntBetween(1, 35),
						},
					},
				},
				Optional: true,
				MaxItems: 1,
				Description: "Azure SQL Database continuous backups for point-in-time recovery. Continuous " +
					"backups are stored in the source database. Note, the changes will be applied during the next " +
					"maintenance window.",
			},
			keyAzureSQLManagedInstanceConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention specifies for how long, in days, the log backups are kept.",
							ValidateFunc: validation.IntBetween(1, 35),
						},
					},
				},
				Optional: true,
				MaxItems: 1,
				Description: "Azure SQL MI log backups. Note, the changes will be applied during the next " +
					"maintenance window.",
			},
			keyBackupLocation: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyArchivalGroupID: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Archival group ID (UUID).",
							ValidateFunc: validation.IsUUID,
						},
					},
				},
				Optional: true,
			},
			keyDailySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in days.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlsla.Days),
							Description: "Retention unit specifies the unit of the `retention` field. Possible " +
								"values are `DAYS`, `WEEKS` and `MONTHS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyHourlySchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in days.",
			},
			keyDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "SLA Domain description.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyFirstFullSnapshot: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDuration: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Duration of snapshot window in hours.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyStartAt: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Start of the snapshot window. Should be given as `DAY, HH:MM`, e.g: " +
								"`Mon, 15:30`.",
							ValidateFunc: validateStartAt(true),
						},
					},
				},
				Optional: true,
				Description: "Specifies the snapshot window where the first full snapshot will be taken. If not " +
					"specified it will be at first opportunity.",
			},
			keyHourlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in hours.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlsla.Days),
							Description: "Retention unit specifies the unit of the `retention` field. Possible " +
								"values are `HOURS`, `DAYS` and `WEEKS`. Default value is `DAYS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in hours.",
			},
			keyMonthlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDayOfMonth: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Day of month. Possible values are `FIRST_DAY`, `FIFTEENTH` and `LAST_DAY`.",
							ValidateFunc: validation.StringInSlice([]string{
								gqlsla.FirstDay,
								string(gqlsla.FifteenthDay),
								gqlsla.LastDay,
							}, false),
						},
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in months.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Minute),
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyHourlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in months.",
			},
			keyName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "SLA Domain name.",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			keyObjectTypes: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(gqlsla.AllObjectTypesAsStrings(), false),
				},
				Required: true,
				Description: "Object types which can be protected by the SLA Domain. Possible values are " +
					"`AWS_EC2_EBS_OBJECT_TYPE`, `AWS_RDS_OBJECT_TYPE`, `AWS_S3_OBJECT_TYPE`, `AZURE_OBJECT_TYPE`, " +
					"`AZURE_SQL_DATABASE_OBJECT_TYPE`, `AZURE_SQL_MANAGED_INSTANCE_OBJECT_TYPE`, " +
					"`AZURE_BLOB_OBJECT_TYPE` and `GCP_OBJECT_TYPE`. Note, `AZURE_SQL_DATABASE_OBJECT_TYPE` cannot " +
					"be provided at the same time as other object types.",
			},
			keyQuarterlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDayOfQuarter: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Day of quarter. Possible values are `FIRST_DAY` and `LAST_DAY`.",
							ValidateFunc: validation.StringInSlice([]string{
								gqlsla.FirstDay,
								gqlsla.LastDay,
							}, false),
						},
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in quarters.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyQuarterStartMonth: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Quarter start month. Possible values are `JANUARY`, `FEBRUARY`, " +
								"`MARCH`, `APRIL`, `MAY`, `JUNE`, `JULY`, `AUGUST`, `SEPTEMBER`, `OCTOBER`, " +
								"`NOVEMBER` and `DECEMBER`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.January),
								string(gqlsla.February),
								string(gqlsla.March),
								string(gqlsla.April),
								string(gqlsla.May),
								string(gqlsla.June),
								string(gqlsla.July),
								string(gqlsla.August),
								string(gqlsla.September),
								string(gqlsla.October),
								string(gqlsla.November),
								string(gqlsla.December),
							}, false),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Minute),
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyHourlySchedule,
					keyMonthlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in quarters.",
			},
			keyReplicationSpec: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyAWSRegion: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "AWS region to replicate to. Should be specified in the standard AWS " +
								"style, e.g. `us-west-2`.",
							ValidateFunc: validation.StringInSlice(gqlaws.AllRegionNames(), false),
						},
						keyAWSCrossAccount: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Replication targetRSC cloud account ID) for cross account replication. Set to empyt string for same account replication.",
						},
						keyAzureRegion: {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Azure region to replicate to. Should be specified in the standard " +
								"Azure style, e.g. `eastus`.",
							ValidateFunc: validation.StringInSlice(gqlazure.AllRegionNames(), false),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional:    true,
				Description: "Replication specification for the SLA Domain. ",
			},
			keyRetentionLock: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyMode: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Retention lock mode. Possible values are `COMPLIANCE` and `GOVERNANCE`.",
							ValidateFunc: validation.StringInSlice([]string{
								"COMPLIANCE",
								"GOVERNANCE",
							}, false),
						},
					},
				},
				Optional: true,
				MaxItems: 1,
				Description: "Enable retention lock. Retention lock prevents data from being accidentally or " +
					"maliciously modified or deleted during the retention period",
			},
			keySnapshotWindow: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDuration: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Duration of the snapshot window in hours.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyStartAt: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Start of the snapshot window. Should be given as `HH:MM`, e.g: " +
								"`15:30`.",
							// Snapshot windows with day of week are accepted by the API but not used by RSC
							// causing inaccurate diffs if allowed.
							ValidateFunc: validateStartAt(false),
						},
					},
				},
				Optional:    true,
				Description: "Specifies an optional snapshot window.",
			},
			keyLocalRetention: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(gqlsla.Days),
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Minute),
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "",
			},
			keyWeeklySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDayOfWeek: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Day of week. Possible values are `MONDAY`, `TUESDAY`, `WEDNESDAY`, " +
								"`THURSDAY`, `FRIDAY`, `SATURDAY` and `SUNDAY`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Monday),
								string(gqlsla.Tuesday),
								string(gqlsla.Wednesday),
								string(gqlsla.Thursday),
								string(gqlsla.Friday),
								string(gqlsla.Saturday),
								string(gqlsla.Sunday),
							}, false),
						},
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in weeks.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Minute),
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyHourlySchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in weeks.",
			},
			keyYearlySchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyDayOfYear: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Day of year. Possible values are `FIRST_DAY` and `LAST_DAY`.",
							ValidateFunc: validation.StringInSlice([]string{"FIRST_DAY", "LAST_DAY"}, false),
						},
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency (years).",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Retention specifies for how long the snapshots are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Retention unit specifies the unit of `retention`. Possible values are " +
								"`MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Minute),
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
								string(gqlsla.Quarters),
								string(gqlsla.Years),
							}, false),
						},
						keyYearStartMonth: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Year start month. Possible values are `JANUARY`, `FEBRUARY`, " +
								"`MARCH`, `APRIL`, `MAY`, `JUNE`, `JULY`, `AUGUST`, `SEPTEMBER`, `OCTOBER`, " +
								"`NOVEMBER` and `DECEMBER`.",
							ValidateFunc: validation.StringInSlice([]string{
								"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE", "JULY", "AUGUST", "SEPTEMBER",
								"OCTOBER", "NOVEMBER", "DECEMBER",
							}, false),
						},
					},
				},
				Optional: true,
				ForceNew: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyHourlySchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in years.",
			},
		},
	}
}

// fromArchival returns a slice of ArchivalSpec structs holding the archival
// configuration.
func fromArchival(d *schema.ResourceData, schedule gqlsla.SnapshotSchedule) ([]gqlsla.ArchivalSpec, error) {
	var archivalSpecs []gqlsla.ArchivalSpec
	for _, archival := range d.Get(keyArchival).([]any) {
		archival := archival.(map[string]any)

		groupID, err := uuid.Parse(archival[keyArchivalLocationID].(string))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %s", keyArchivalLocationID, err)
		}

		archivalSpecs = append(archivalSpecs, gqlsla.ArchivalSpec{
			GroupID:       groupID,
			Frequencies:   frequenciesFromSchedule(schedule),
			Threshold:     archival[keyThreshold].(int),
			ThresholdUnit: gqlsla.RetentionUnit(archival[keyThresholdUnit].(string)),
		})
	}

	return archivalSpecs, nil
}

// toArchival returns a slice holding the archival configuration.
func toArchival(archivalSpecs []gqlsla.ArchivalSpec, existing []any) ([]any, error) {
	blocks := make(map[string]map[string]any)
	for _, spec := range archivalSpecs {
		id := spec.GroupID.String()
		if blocks[id] != nil {
			return nil, fmt.Errorf("archival location %q used multiple times", spec.GroupID.String())
		}
		blocks[id] = map[string]any{
			keyArchivalLocationID: id,
			keyThreshold:          spec.Threshold,
			keyThresholdUnit:      string(spec.ThresholdUnit),
		}
	}

	// Preserve order from existing, then add new ones to the end.
	var sorted []any
	for _, old := range existing {
		id := old.(map[string]any)[keyArchivalLocationID].(string)
		if block, ok := blocks[id]; ok {
			sorted = append(sorted, block)
			delete(blocks, id)
		}
	}

	// Add remaining blocks in the order they appear in archivalSpecs.
	for _, spec := range archivalSpecs {
		id := spec.GroupID.String()
		if _, ok := blocks[id]; !ok {
			continue
		}
		sorted = append(sorted, blocks[id])
	}
	return sorted, nil
}

// fromReplicationSpec returns a slice of ReplicationSpec structs holding the
// replication configuration.
func fromReplicationSpec(d *schema.ResourceData) ([]gqlsla.ReplicationSpec, error) {
	var replicationSpecs []gqlsla.ReplicationSpec
	for _, spec := range d.Get(keyReplicationSpec).([]any) {
		spec := spec.(map[string]any)

		retention := &gqlsla.RetentionDuration{
			Duration: spec[keyRetention].(int),
			Unit:     gqlsla.RetentionUnit(spec[keyRetentionUnit].(string)),
		}
		var awsRegion gqlaws.Region
		var awsCrossAccount string
		if name := spec[keyAWSRegion].(string); name != "" {
			awsRegion = gqlaws.RegionFromName(name)
			if awsRegion == gqlaws.RegionUnknown {
				return nil, fmt.Errorf("unknown AWS region: %s", name)
			}
			awsCrossAccount = spec[keyAWSCrossAccount].(string)
			if awsCrossAccount == "" {
				awsCrossAccount = "SAME"
			}

		}
		var azureRegion gqlazure.Region
		var azureCrossSubscription string
		if name := spec[keyAzureRegion].(string); name != "" {
			azureRegion = gqlazure.RegionFromName(name)
			if azureRegion == gqlazure.RegionUnknown {
				return nil, fmt.Errorf("unknown Azure region: %s", name)
			}
			azureCrossSubscription = "SAME"
		}
		replicationSpecs = append(replicationSpecs, gqlsla.ReplicationSpec{
			AWSRegion:         awsRegion.ToRegionForReplicationEnum(),
			AWSAccount:        awsCrossAccount,
			AzureRegion:       azureRegion.ToRegionForReplicationEnum(),
			AzureSubscription: azureCrossSubscription,
			RetentionDuration: retention,
		})
	}

	return replicationSpecs, nil
}

// toReplicationSpec returns a slice holding the replication configuration.
func toReplicationSpec(replicationSpecs []gqlsla.ReplicationSpec) []any {
	var replicationSpec []any
	for _, spec := range replicationSpecs {
		replicationSpec = append(replicationSpec, map[string]any{
			keyAWSRegion:       spec.AWSRegion.Name(),
			keyAWSCrossAccount: spec.AWSAccount,
			keyAzureRegion:     spec.AzureRegion.Name(),
			keyRetention:       spec.RetentionDuration.Duration,
			keyRetentionUnit:   spec.RetentionDuration.Unit,
		})
	}

	return replicationSpec
}

// fromSnapshotWindow returns a slice of BackupWindow structs holding the
// snapshot window configuration.
func fromSnapshotWindow(windows []any) ([]gqlsla.BackupWindow, error) {
	var snapshotWindows []gqlsla.BackupWindow
	for _, snapshotWindow := range windows {
		snapshotWindow := snapshotWindow.(map[string]any)

		// Parse start time, e.g. "Mon, 15:30" or "16:45".
		var day string
		var timeParts []string
		parts := strings.Split(snapshotWindow[keyStartAt].(string), ", ")
		switch len(parts) {
		case 1:
			// No day of week specified.
			timeParts = strings.Split(parts[0], ":")
		case 2:
			// Day of week specified.
			switch strings.ToUpper(parts[0]) {
			case "MON":
				day = "MONDAY"
			case "TUE":
				day = "TUESDAY"
			case "WED":
				day = "WEDNESDAY"
			case "THU":
				day = "THURSDAY"
			case "FRI":
				day = "FRIDAY"
			case "SAT":
				day = "SATURDAY"
			case "SUN":
				day = "SUNDAY"
			default:
				return nil, fmt.Errorf("invalid day of week for %s: %s", keyStartAt, snapshotWindow[keyStartAt].(string))
			}
			timeParts = strings.Split(parts[1], ":")
		default:
			return nil, fmt.Errorf("invalid format for %s: %s", keyStartAt, snapshotWindow[keyStartAt].(string))
		}

		if len(timeParts) != 2 {
			return nil, fmt.Errorf("invalid time format for %s: %s", keyStartAt, snapshotWindow[keyStartAt].(string))
		}
		h, err := strconv.Atoi(timeParts[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse hour for %s: %s", keyStartAt, err)
		}
		m, err := strconv.Atoi(timeParts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse minute for %s: %s", keyStartAt, err)
		}

		snapshotWindows = append(snapshotWindows, gqlsla.BackupWindow{
			DurationInHours: snapshotWindow[keyDuration].(int),
			StartTime:       gqlsla.StartTime{DayOfWeek: gqlsla.DayOfWeek{Day: gqlsla.Day(day)}, Hour: h, Minute: m},
		})
	}

	return snapshotWindows, nil
}

// toSnapshotWindow returns a slice holding the snapshot window configuration.
func toSnapshotWindow(backupWindows []gqlsla.BackupWindow) ([]any, error) {
	var snapshotWindow []any
	for _, backupWindow := range backupWindows {
		startAt := fmt.Sprintf("%02d:%02d", backupWindow.StartTime.Hour, backupWindow.StartTime.Minute)
		if day := backupWindow.StartTime.DayOfWeek.Day; day != "" {
			wd, err := day.ToWeekday()
			if err != nil {
				return nil, err
			}
			startAt = wd.String()[:3] + ", " + startAt
		}
		snapshotWindow = append(snapshotWindow, map[string]any{
			keyDuration: backupWindow.DurationInHours,
			keyStartAt:  startAt,
		})
	}

	return snapshotWindow, nil
}

// fromLocalRetention returns a RetentionDuration struct holding the local
// retention configuration, or nil if local retention was not configured.
func fromLocalRetention(d *schema.ResourceData) *gqlsla.RetentionDuration {
	block, ok := d.GetOk(keyLocalRetention)
	if !ok {
		return nil
	}

	localRetention := block.([]any)[0].(map[string]any)
	return &gqlsla.RetentionDuration{
		Duration: localRetention[keyRetention].(int),
		Unit:     gqlsla.RetentionUnit(localRetention[keyRetentionUnit].(string)),
	}
}

// toLocalRetention returns a map holding the source retention configuration or
// nil if the RetentionDuration is nil.
func toLocalRetention(localRetention *gqlsla.RetentionDuration) []any {
	if localRetention == nil {
		return nil
	}
	return []any{map[string]any{
		keyRetention:     localRetention.Duration,
		keyRetentionUnit: string(localRetention.Unit),
	}}
}

// frequenciesFromSchedule returns the frequencies from the given snapshot
// schedule.
func frequenciesFromSchedule(schedule gqlsla.SnapshotSchedule) []gqlsla.RetentionUnit {
	var frequencies []gqlsla.RetentionUnit

	if schedule.Minute != nil {
		frequencies = append(frequencies, gqlsla.Minute)
	}
	if schedule.Hourly != nil {
		frequencies = append(frequencies, gqlsla.Hours)
	}
	if schedule.Daily != nil {
		frequencies = append(frequencies, gqlsla.Days)
	}
	if schedule.Weekly != nil {
		frequencies = append(frequencies, gqlsla.Weeks)
	}
	if schedule.Monthly != nil {
		frequencies = append(frequencies, gqlsla.Months)
	}
	if schedule.Quarterly != nil {
		frequencies = append(frequencies, gqlsla.Quarters)
	}
	if schedule.Yearly != nil {
		frequencies = append(frequencies, gqlsla.Years)
	}

	return frequencies
}

// newSLADomainMutator returns a function that can be used to either create
// or update SLA domain depending on the op parameter.
func newSLADomainMutator(op string) func(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Printf("[TRACE] newSLADomainMutator op: %s", op)
	return func(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
		client, err := m.(*client).polaris()
		if err != nil {
			return diag.FromErr(err)
		}

		// Parse snapshot schedule. Unspecified time frame schedules are nil.
		schedule := gqlsla.SnapshotSchedule{
			Daily:     fromDailySchedule(d),
			Hourly:    fromHourlySchedule(d),
			Minute:    fromMinuteSchedule(d),
			Monthly:   fromMonthlySchedule(d),
			Quarterly: fromQuarterlySchedule(d),
			Weekly:    fromWeeklySchedule(d),
			Yearly:    fromYearlySchedule(d),
		}
		archivalSpecs, err := fromArchival(d, schedule)
		if err != nil {
			return diag.FromErr(err)
		}
		awsRDSConfig, err := fromAWSRDSConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		azureSQLConfig, err := fromAzureSQLConfig(d, keyAzureSQLDatabaseConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		azureSQLMIConfig, err := fromAzureSQLConfig(d, keyAzureSQLManagedInstanceConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		blobConfig, err := fromAzureBlobConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		firstFullSnapshotWindows, err := fromSnapshotWindow(d.Get(keyFirstFullSnapshot).([]any))
		if err != nil {
			return diag.FromErr(err)
		}
		replicationSpecs, err := fromReplicationSpec(d)
		if err != nil {
			return diag.FromErr(err)
		}
		snapshotWindows, err := fromSnapshotWindow(d.Get(keySnapshotWindow).([]any))
		if err != nil {
			return diag.FromErr(err)
		}

		// AWS S3 is supported in two modes. The old mode uses a single backup location
		// with object specific configuration. The new mode uses multiple backup locations.
		mbl, err := core.Wrap(client.GQL).FeatureFlag(ctx, "CNP_AWS_S3_MULTIPLE_BACKUP_LOCATIONS_ENABLED")
		if err != nil {
			return diag.FromErr(err)
		}
		var backupLocations []gqlsla.BackupLocationSpec
		var awsS3Config *gqlsla.AWSS3Config
		if mbl.Enabled {
			backupLocations = fromBackupLocation(d)
		} else {
			if awsS3Config, err = fromAWSS3Config(d); err != nil {
				return diag.FromErr(err)
			}
		}

		var objectTypes []gqlsla.ObjectType
		objectTypeList := d.Get(keyObjectTypes).(*schema.Set).List()
		for _, objectType := range objectTypeList {
			objectType := gqlsla.ObjectType(objectType.(string))

			// Per object type validation.
			switch objectType {
			case gqlsla.ObjectAzureSQLDatabase:
				if len(objectTypeList) > 1 {
					return diag.Errorf("Azure SQL Database object type cannot be combined with other object types")
				}
				if azureSQLConfig == nil {
					return diag.Errorf("Azure SQL Database object type requires Azure SQL Database configuration")
				}
				if len(archivalSpecs) != 1 {
					return diag.Errorf("Azure SQL Database object type requires an archival location with instant archival enabled")
				}
				if archivalSpecs[0].Threshold != 0 {
					return diag.Errorf("Azure SQL Database object type requires an archival location with instant archival enabled")
				}
				if len(replicationSpecs) > 0 {
					return diag.Errorf("Azure SQL Database object type does not support replication")
				}
			case gqlsla.ObjectAzureSQLManagedInstance:
				if azureSQLMIConfig == nil {
					return diag.Errorf("Azure SQL Managed Instance object type requires Azure SQL Managed Instance configuration")
				}
				if azureSQLConfig != nil {
					return diag.Errorf("Azure SQL Managed Instance object type cannot be combined with Azure SQL Database configuration")
				}
				if len(archivalSpecs) > 0 {
					return diag.Errorf("Azure SQL Managed Instance object type does not support archival locations")
				}
				if len(replicationSpecs) > 0 {
					return diag.Errorf("Azure SQL Managed Instance object type does not support replication")
				}
			case gqlsla.ObjectAzureBlob:
				if blobConfig == nil {
					return diag.Errorf("Azure Blob object type requires Azure Blob configuration")
				}
			case gqlsla.ObjectAWSS3:
				if len(objectTypeList) > 1 {
					return diag.Errorf("AWS S3 object type cannot be combined with other object types")
				}
				if mbl.Enabled && len(backupLocations) == 0 {
					return diag.Errorf("AWS S3 object type requires at least one backup location")
				}
				if !mbl.Enabled && awsS3Config == nil {
					return diag.Errorf("AWS S3 object type requires AWS S3 configuration")
				}
			}
			objectTypes = append(objectTypes, objectType)
		}

		createParams := gqlsla.CreateDomainParams{
			ArchivalSpecs:          archivalSpecs,
			BackupLocationSpecs:    backupLocations,
			BackupWindows:          snapshotWindows,
			Description:            d.Get(keyDescription).(string),
			FirstFullBackupWindows: firstFullSnapshotWindows,
			LocalRetentionLimit:    fromLocalRetention(d),
			Name:                   d.Get(keyName).(string),
			ObjectSpecificConfigs: &gqlsla.ObjectSpecificConfigs{
				AWSS3Config:                     awsS3Config,
				AWSRDSConfig:                    awsRDSConfig,
				AzureBlobConfig:                 blobConfig,
				AzureSQLDatabaseDBConfig:        azureSQLConfig,
				AzureSQLManagedInstanceDBConfig: azureSQLMIConfig,
			},
			ObjectTypes:       objectTypes,
			ReplicationSpecs:  replicationSpecs,
			RetentionLock:     false,
			RetentionLockMode: "",
			SnapshotSchedule:  schedule,
		}

		switch op {
		case "create":
			id, err := sla.Wrap(client).CreateDomain(ctx, createParams)
			if err != nil {
				return diag.FromErr(err)
			}

			d.SetId(id.String())
			readSLADomain(ctx, d, m)
			return nil
		case "update":
			id, err := uuid.Parse(d.Id())
			if err != nil {
				return diag.FromErr(err)
			}

			applyToExisting := d.Get(keyApplyChangesToExistingSnapshots).(bool)
			applyToNonPolicy := applyToExisting && d.Get(keyApplyChangesToNonPolicySnapshots).(bool)

			if err := sla.Wrap(client).UpdateDomain(ctx, gqlsla.UpdateDomainParams{
				ID:                              id,
				ShouldApplyToExistingSnapshots:  &gqlsla.BoolValue{Value: applyToExisting},
				ShouldApplyToNonPolicySnapshots: &gqlsla.BoolValue{Value: applyToNonPolicy},
				CreateDomainParams:              createParams,
			}); err != nil {
				return diag.FromErr(err)
			}
			return nil
		default:
			panic("unknown operation")
		}
	}
}

func readSLADomain(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] readSLADomain")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	slaDomain, err := sla.Wrap(client).DomainByID(ctx, id)
	if errors.Is(err, graphql.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	objectTypes := &schema.Set{F: schema.HashString}
	for _, objectType := range slaDomain.ObjectTypes {
		objectTypes.Add(string(objectType))
	}

	if err := d.Set(keyName, slaDomain.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDescription, slaDomain.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyObjectTypes, objectTypes); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDailySchedule, toDailySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyHourlySchedule, toHourlySchedule(slaDomain)); err != nil {
		return diag.FromErr(err)
	}
	//if err := d.Set(keyMinuteSchedule, toMinuteSchedule(slaDomain)); err != nil {
	//	return diag.FromErr(err)
	//}
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
	if err := d.Set(keyAWSRDSConfig, toAWSRDSConfig(slaDomain.ObjectSpecificConfigs.AWSRDSConfig)); err != nil {
		return diag.FromErr(err)
	}

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

	// AWS S3 object type is supported in two ways, either using backup location specs, or
	// using object specific configs if multiple backup locations are not enabled.
	backupLocations, err := toBackupLocations(slaDomain, d.Get(keyBackupLocation).([]any))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyBackupLocation, backupLocations); err != nil {
		return diag.FromErr(err)
	}

	snapshotWindow, err := toSnapshotWindow(slaDomain.BackupWindows)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySnapshotWindow, snapshotWindow); err != nil {
		return diag.FromErr(err)
	}
	firstFullSnapshot, err := toSnapshotWindow(slaDomain.FirstFullBackupWindows)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyFirstFullSnapshot, firstFullSnapshot); err != nil {
		return diag.FromErr(err)
	}

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

	if slaDomain.LocalRetentionLimit != nil {
		if err := d.Set(keyLocalRetention, toLocalRetention(&gqlsla.RetentionDuration{
			Duration: slaDomain.LocalRetentionLimit.Duration,
			Unit:     slaDomain.LocalRetentionLimit.Unit,
		})); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func deleteSLADomain(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	log.Print("[TRACE] deleteSLADomain")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.Parse(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := sla.Wrap(client).DeleteDomain(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func fromAWSRDSConfig(d *schema.ResourceData) (*gqlsla.AWSRDSConfig, error) {
	block, ok := d.GetOk(keyAWSRDSConfig)
	if !ok {
		return nil, nil
	}

	rdsConfig := block.([]any)[0].(map[string]any)
	return &gqlsla.AWSRDSConfig{
		LogRetention: gqlsla.RetentionDuration{
			Duration: rdsConfig[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(rdsConfig[keyLogRetentionUnit].(string)),
		},
	}, nil
}

func toAWSRDSConfig(rdsConfig *gqlsla.AWSRDSConfig) []any {
	if rdsConfig == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention:     rdsConfig.LogRetention.Duration,
		keyLogRetentionUnit: rdsConfig.LogRetention.Unit,
	}}
}

func fromAWSS3Config(d *schema.ResourceData) (*gqlsla.AWSS3Config, error) {
	locations := d.Get(keyBackupLocation).([]any)
	if len(locations) == 0 {
		return nil, nil
	}
	if len(locations) > 1 {
		return nil, fmt.Errorf("multiple backup locations not supported")
	}
	groupID, err := uuid.Parse(locations[0].(map[string]any)[keyArchivalGroupID].(string))
	if err != nil {
		return nil, err
	}
	return &gqlsla.AWSS3Config{
		ArchivalLocationID: groupID,
	}, nil
}

func fromAzureBlobConfig(d *schema.ResourceData) (*gqlsla.AzureBlobConfig, error) {
	block, ok := d.GetOk(keyAzureBlobConfig)
	if !ok {
		return nil, nil
	}

	blobConfig := block.([]any)[0].(map[string]any)
	archivalLocationID, err := uuid.Parse(blobConfig[keyArchivalLocationID].(string))
	if err != nil {
		return nil, err
	}

	return &gqlsla.AzureBlobConfig{
		BackupLocationID:                archivalLocationID,
		ContinuousBackupRetentionInDays: 1,
	}, nil
}

func fromAzureSQLConfig(d *schema.ResourceData, key string) (*gqlsla.AzureDBConfig, error) {
	block, ok := d.GetOk(key)
	if !ok {
		return nil, nil
	}

	sqlConfig := block.([]any)[0].(map[string]any)
	return &gqlsla.AzureDBConfig{
		LogRetentionInDays: sqlConfig[keyLogRetention].(int),
	}, nil
}

func toAzureSQLConfig(sqlConfig *gqlsla.AzureDBConfig) []any {
	if sqlConfig == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention: sqlConfig.LogRetentionInDays,
	}}
}

func fromBackupLocation(d *schema.ResourceData) []gqlsla.BackupLocationSpec {
	var locations []gqlsla.BackupLocationSpec
	for _, l := range d.Get(keyBackupLocation).([]any) {
		l := l.(map[string]any)
		groupID, err := uuid.Parse(l[keyArchivalGroupID].(string))
		if err != nil {
			return nil
		}
		locations = append(locations, gqlsla.BackupLocationSpec{
			ArchivalGroupID: groupID,
		})
	}
	return locations
}

func toBackupLocations(slaDomain gqlsla.Domain, existing []any) ([]any, error) {
	blocks := make(map[string]map[string]any)
	for _, spec := range slaDomain.BackupLocationSpecs {
		id := spec.ArchivalGroup.ID
		if blocks[id] != nil {
			return nil, fmt.Errorf("archival location %q used multiple times", id)
		}
		blocks[id] = map[string]any{
			keyArchivalGroupID: id,
		}
	}

	// Preserve order from existing, then add new ones to the end.
	var sorted []any
	for _, old := range existing {
		id := old.(map[string]any)[keyArchivalGroupID].(string)
		if block, ok := blocks[id]; ok {
			sorted = append(sorted, block)
			delete(blocks, id)
		}
	}

	// Add remaining blocks in the order they appear in backupLocationSpecs.
	for _, spec := range slaDomain.BackupLocationSpecs {
		id := spec.ArchivalGroup.ID
		if _, ok := blocks[id]; !ok {
			continue
		}
		sorted = append(sorted, blocks[id])
	}

	// AWS S3 fallback when multiple backup locations are not enabled.
	if len(sorted) == 0 && slaDomain.ObjectSpecificConfigs.AWSS3Config != nil {
		sorted = append(sorted, map[string]any{
			keyArchivalGroupID: slaDomain.ObjectSpecificConfigs.AWSS3Config.ArchivalLocationID.String(),
		})
	}
	return sorted, nil
}

func fromDailySchedule(d *schema.ResourceData) *gqlsla.DailySnapshotSchedule {
	data, ok := d.GetOk(keyDailySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.DailySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
	}
}

func toDailySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Daily == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     slaDomain.SnapshotSchedule.Daily.BasicSchedule.Frequency,
		keyRetention:     slaDomain.SnapshotSchedule.Daily.BasicSchedule.Retention,
		keyRetentionUnit: slaDomain.SnapshotSchedule.Daily.BasicSchedule.RetentionUnit,
	}}
}

func fromHourlySchedule(d *schema.ResourceData) *gqlsla.HourlySnapshotSchedule {
	data, ok := d.GetOk(keyHourlySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.HourlySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
	}
}

func toHourlySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Hourly == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     slaDomain.SnapshotSchedule.Hourly.BasicSchedule.Frequency,
		keyRetention:     slaDomain.SnapshotSchedule.Hourly.BasicSchedule.Retention,
		keyRetentionUnit: slaDomain.SnapshotSchedule.Hourly.BasicSchedule.RetentionUnit,
	}}
}

func fromMinuteSchedule(d *schema.ResourceData) *gqlsla.MinuteSnapshotSchedule {
	data, ok := d.GetOk(keyMinuteSchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.MinuteSnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
	}
}

func toMinuteSchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Minute == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     slaDomain.SnapshotSchedule.Minute.BasicSchedule.Frequency,
		keyRetention:     slaDomain.SnapshotSchedule.Minute.BasicSchedule.Retention,
		keyRetentionUnit: slaDomain.SnapshotSchedule.Minute.BasicSchedule.RetentionUnit,
	}}
}

func fromMonthlySchedule(d *schema.ResourceData) *gqlsla.MonthlySnapshotSchedule {
	data, ok := d.GetOk(keyMonthlySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.MonthlySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
		DayOfMonth: gqlsla.DayOfMonth(schedule[keyDayOfMonth].(string)),
	}
}

func toMonthlySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Monthly == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     slaDomain.SnapshotSchedule.Monthly.BasicSchedule.Frequency,
		keyRetention:     slaDomain.SnapshotSchedule.Monthly.BasicSchedule.Retention,
		keyRetentionUnit: slaDomain.SnapshotSchedule.Monthly.BasicSchedule.RetentionUnit,
		keyDayOfMonth:    slaDomain.SnapshotSchedule.Monthly.DayOfMonth,
	}}
}

func fromQuarterlySchedule(d *schema.ResourceData) *gqlsla.QuarterlySnapshotSchedule {
	data, ok := d.GetOk(keyQuarterlySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.QuarterlySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
		DayOfQuarter:      gqlsla.DayOfQuarter(schedule[keyDayOfQuarter].(string)),
		QuarterStartMonth: gqlsla.Month(schedule[keyQuarterStartMonth].(string)),
	}
}

func toQuarterlySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Quarterly == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:         slaDomain.SnapshotSchedule.Quarterly.BasicSchedule.Frequency,
		keyRetention:         slaDomain.SnapshotSchedule.Quarterly.BasicSchedule.Retention,
		keyRetentionUnit:     slaDomain.SnapshotSchedule.Quarterly.BasicSchedule.RetentionUnit,
		keyDayOfQuarter:      slaDomain.SnapshotSchedule.Quarterly.DayOfQuarter,
		keyQuarterStartMonth: slaDomain.SnapshotSchedule.Quarterly.QuarterStartMonth,
	}}
}

func fromWeeklySchedule(d *schema.ResourceData) *gqlsla.WeeklySnapshotSchedule {
	data, ok := d.GetOk(keyWeeklySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.WeeklySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
		DayOfWeek: gqlsla.Day(schedule[keyDayOfWeek].(string)),
	}
}

func toWeeklySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Weekly == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     slaDomain.SnapshotSchedule.Weekly.BasicSchedule.Frequency,
		keyRetention:     slaDomain.SnapshotSchedule.Weekly.BasicSchedule.Retention,
		keyRetentionUnit: slaDomain.SnapshotSchedule.Weekly.BasicSchedule.RetentionUnit,
		keyDayOfWeek:     slaDomain.SnapshotSchedule.Weekly.DayOfWeek,
	}}
}

func fromYearlySchedule(d *schema.ResourceData) *gqlsla.YearlySnapshotSchedule {
	data, ok := d.GetOk(keyYearlySchedule)
	if !ok {
		return nil
	}

	schedule := data.([]any)[0].(map[string]any)
	return &gqlsla.YearlySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
		DayOfYear:      gqlsla.DayOfYear(schedule[keyDayOfYear].(string)),
		YearStartMonth: gqlsla.Month(schedule[keyYearStartMonth].(string)),
	}
}

func toYearlySchedule(slaDomain gqlsla.Domain) []any {
	if slaDomain.SnapshotSchedule.Yearly == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:      slaDomain.SnapshotSchedule.Yearly.BasicSchedule.Frequency,
		keyRetention:      slaDomain.SnapshotSchedule.Yearly.BasicSchedule.Retention,
		keyRetentionUnit:  slaDomain.SnapshotSchedule.Yearly.BasicSchedule.RetentionUnit,
		keyDayOfYear:      slaDomain.SnapshotSchedule.Yearly.DayOfYear,
		keyYearStartMonth: slaDomain.SnapshotSchedule.Yearly.YearStartMonth,
	}}
}
