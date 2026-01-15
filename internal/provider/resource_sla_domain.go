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
	"regexp"
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
databases, SaaS apps and cloud objects. An SLA Domain can define frequency,
retention, archival and replication.

-> Enabling Instant Archive can increase bandwidth usage and archival storage
   requirements.

-> The hourly retention for snapshots of cloud-native workloads must be a
   multiple of 24.

-> For workloads backed up on a Rubrik cluster, snapshots are scheduled using
   the time zone of that Rubrik cluster. For workloads backed up in the cloud,
   snapshots are scheduled using the UTC time zone.

---

### Frequency

This defines when and how often snapshots are taken. This could be interval-based (days, hours, minutes) or calendar-based (a day of each month).

### Retention

This defines how long the snapshot is kept on the Rubrik cluster.

### Archival
Before You Start: To archive snapshots, make sure you’ve added archival locations.

To avoid early deletion fees, retain snapshots in cool tier archival locations for at least 30 days.

---
# Object types

## Active Directory
Active Directory protection supports a minimum of 4 hours SLA.

## Azure SQL Databases
Archival is mandatory and the backups will be instantly archived. Frequency and Retention apply to archived snapshots of the Azure SQL database.
Continuous backups for point-in-time recovery retentions is configured in ´azure_sql_database_config´.

## Azure SQL Managed Instance
Archival and Replication are not supported by Azure SQL Managed Instance.
Log backup for Azure SQL MI is configured in ´azure_sql_managed_instance_config´.

## Azure Blob Storage
Archival and Replication are not supported by Azure Blob Storage.
Backup location for scheduled snapshots is configured in ´azure_blob_config´.

## AWS RDS
Archival is only supported for PostgrSQL and Aurora PostgreSQL databases.
Continuous backups for point-in-time recovery retention is configured in ´aws_rds_config´. If you don't specify a continuous backup, AWS provides 1 day of continuous backup by default for Aurora databases, which you can change but you can’t disable.

## AWS S3
Archival and Replication are not supported by AWS S3. SLA Domains protecting AWS S3 cannot protect other object types.
Backup location(s) are configured in ´backup_location´.

## AWS DynamoDB
Replication is not supported by AWS DynamoDB.
Primary Backup Encryption KMS Key and Continuous backups for point-in-time recovery are configured in ´aws_dynamodb_config´. Continuous backups will be automatically enabled for your DynamoDB tables.
Disabling continuous backups or changing the retention period in your AWS console may lead to higher storage and consumption costs. To avoid this, keep continuous backups enabled in your AWS console.

## GCE Instance/Disk
Replication is not supported by GCE Instance/Disk.

## Okta
Archival and Replication are not supported by Okta.

## Microsoft 365
Archival and Replication are not supported by Microsoft 365.
M365 protection supports a minimum of 8 hours SLA (12 hours or more recomended).

## OLVM
Archival is not supported by OLVM.
`

func resourceSLADomain() *schema.Resource {
	return &schema.Resource{
		CreateContext: newSLADomainMutator("create"),
		ReadContext:   readSLADomain,
		UpdateContext: newSLADomainMutator("update"),
		DeleteContext: deleteSLADomain,

		Description: description(resourceSLADomainDescription),
		Importer: &schema.ResourceImporter{
			StateContext: importSLADomain,
		},
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
							Optional:     true,
							Computed:     true,
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
						keyArchivalLocationToClusterMapping: {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyClusterID: {
										Type:         schema.TypeString,
										Optional:     true,
										Computed:     true,
										Description:  "Cluster ID (UUID).",
										ValidateFunc: validation.IsUUID,
									},
									keyArchivalLocationID: {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "Archival location ID (UUID).",
										ValidateFunc: validation.IsUUID,
									},
									keyClusterName: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Cluster name.",
									},
									keyName: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Archival location name.",
									},
								},
							},
							Optional:    true,
							Description: "Mapping between archival location and Rubrik cluster. Each mapping specifies which cluster should be used for archiving to a specific location.",
						},
						keyArchivalTiering: {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyInstantTiering: {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "Enable instant tiering to cold storage.",
									},
									keyMinAccessibleDurationInSeconds: {
										Type:         schema.TypeInt,
										Optional:     true,
										Description:  "Minimum duration in seconds that data must remain accessible before tiering.",
										ValidateFunc: validation.IntAtLeast(0),
									},
									keyColdStorageClass: {
										Type:     schema.TypeString,
										Optional: true,
										Description: "Cold storage class for tiering. Possible values are " +
											"`AZURE_ARCHIVE`, `AWS_GLACIER`, `AWS_GLACIER_DEEP_ARCHIVE`.",
										ValidateFunc: validation.StringInSlice([]string{
											string(gqlsla.ColdStorageClassAzureArchive),
											string(gqlsla.ColdStorageClassAWSGlacier),
											string(gqlsla.ColdStorageClassAWSGlacierDeepArchive),
										}, false),
									},
									keyTierExistingSnapshots: {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "Whether to tier existing snapshots to cold storage.",
									},
								},
							},
							MaxItems:    1,
							Optional:    true,
							Description: "Archival tiering specification for cold storage.",
						},
					},
				},
				Optional: true,
				Description: "Archive snapshots to the specified archival location. Note, if `instant_archive` is " +
					"enabled, `threshold` and `threshold_unit` are ignored.",
			},
			keyAWSDynamoDBConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyKMSAlias: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "KMS alias for primary backup. Ensure the specified KMS key exists in the respective regions of the DynamoDB tables this SLA will be applied to. Avoid deleting it, as it will be used for data decryption during archival and recovery.",
							ValidateFunc: validation.StringMatch(regexp.MustCompile(`^alias\/[a-zA-Z0-9:/_-]+$`), "KMS alias must be in the format `alias/<name>`"),
						},
					},
				},
				Optional: true,
				MaxItems: 1,
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
			keyVMwareVMConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention specifies for how long, in seconds, the log backups are kept.",
							ValidateFunc: validation.IntAtLeast(1),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "VMware vSphere VM log backups.",
			},
			keySapHanaConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyIncrementalFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Incremental backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyIncrementalFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Incremental frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyLogRetention: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyDifferentialFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Differential backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyDifferentialFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Differential frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyStorageSnapshotConfig: {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyFrequency: {
										Type:         schema.TypeInt,
										Required:     true,
										Description:  "Storage snapshot frequency.",
										ValidateFunc: validation.IntAtLeast(1),
									},
									keyFrequencyUnit: {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      string(gqlsla.Days),
										Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
										ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
									},
									keyRetention: {
										Type:         schema.TypeInt,
										Required:     true,
										Description:  "Storage snapshot retention.",
										ValidateFunc: validation.IntAtLeast(1),
									},
									keyRetentionUnit: {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      string(gqlsla.Days),
										Description:  "Retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
										ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
									},
								},
							},
							Optional:    true,
							MaxItems:    1,
							Description: "SAP HANA storage snapshot configuration.",
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "SAP HANA database configuration.",
			},
			keyDB2Config: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyIncrementalFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Incremental backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyIncrementalFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Incremental frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyLogRetention: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyDifferentialFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Differential backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyDifferentialFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Differential frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyLogArchivalMethod: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Log archival method. Possible values are `LOGARCHMETH1`, `LOGARCHMETH2`.",
							ValidateFunc: validation.StringInSlice([]string{"LOGARCHMETH1", "LOGARCHMETH2"}, false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "Db2 database configuration.",
			},
			keyMSSQLConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "SQL Server database configuration.",
			},
			keyOracleConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyHostLogRetention: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Host log retention duration for archived redo logs.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyHostLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Host log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "Oracle database configuration.",
			},
			keyMongoConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "MongoDB database configuration.",
			},
			keyManagedVolumeConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "Managed Volume configuration.",
			},
			keyPostgresDBClusterConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration for Write-Ahead Logging (WAL) logs.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "Postgres DB Cluster configuration.",
			},
			keyMySQLDBConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "MySQL database configuration.",
			},
			keyInformixConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyIncrementalFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Incremental backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyIncrementalFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Incremental frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyIncrementalRetention: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Incremental backup retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyIncrementalRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Incremental retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyFrequency: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Log backup frequency.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyFrequencyUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Frequency unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
						keyRetention: {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "Informix database configuration.",
			},
			keyGCPCloudSQLConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyLogRetention: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Log retention duration.",
							ValidateFunc: validation.IntAtLeast(1),
						},
						keyLogRetentionUnit: {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      string(gqlsla.Days),
							Description:  "Log retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `YEARS`. Default is `DAYS`.",
							ValidateFunc: validation.StringInSlice(gqlsla.AllRetentionUnitsAsStrings(), false),
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "GCP Cloud SQL configuration.",
			},
			keyNCDConfig: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyMinutelyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for per-minute schedule backups.",
						},
						keyHourlyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for hourly schedule backups.",
						},
						keyDailyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for daily schedule backups.",
						},
						keyWeeklyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for weekly schedule backups.",
						},
						keyMonthlyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for monthly schedule backups.",
						},
						keyQuarterlyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for quarterly schedule backups.",
						},
						keyYearlyBackupLocations: {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsUUID,
							},
							Optional:    true,
							Description: "Target location UUIDs for yearly schedule backups.",
						},
					},
				},
				Optional:    true,
				MaxItems:    1,
				Description: "NAS Cloud Direct configuration.",
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
					keyMinuteSchedule,
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
								"values are `HOURS`, `DAYS`, `WEEKS` and `MONTHS`. Default value is `DAYS`.",
							ValidateFunc: validation.StringInSlice([]string{
								string(gqlsla.Hours),
								string(gqlsla.Days),
								string(gqlsla.Weeks),
								string(gqlsla.Months),
							}, false),
						},
					},
				},
				Optional: true,
				AtLeastOneOf: []string{
					keyDailySchedule,
					keyMinuteSchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in hours.",
			},
			keyMinuteSchedule: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						keyFrequency: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "Frequency in minutes.",
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
					keyHourlySchedule,
					keyMonthlySchedule,
					keyQuarterlySchedule,
					keyWeeklySchedule,
					keyYearlySchedule,
					keyAWSRDSConfig, // For AWS RDS, snapshot frequency is optional.
				},
				MaxItems:    1,
				Description: "Take snapshots with frequency specified in minutes.",
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
					keyMinuteSchedule,
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
					"`ACTIVE_DIRECTORY_OBJECT_TYPE`, `ATLASSIAN_JIRA_OBJECT_TYPE`, `AWS_DYNAMODB_OBJECT_TYPE`, `AWS_EC2_EBS_OBJECT_TYPE`, `AWS_RDS_OBJECT_TYPE`, `AWS_S3_OBJECT_TYPE`, " +
					"`AZURE_AD_OBJECT_TYPE`, `AZURE_BLOB_OBJECT_TYPE`, `AZURE_DEVOPS_OBJECT_TYPE`, `AZURE_OBJECT_TYPE`, `AZURE_SQL_DATABASE_OBJECT_TYPE`, `AZURE_SQL_MANAGED_INSTANCE_OBJECT_TYPE`, " +
					"`CASSANDRA_OBJECT_TYPE`, `D365_OBJECT_TYPE`, `DB2_OBJECT_TYPE`, `EXCHANGE_OBJECT_TYPE`, `FILESET_OBJECT_TYPE`, `GCP_CLOUD_SQL_OBJECT_TYPE`, `GCP_OBJECT_TYPE`, " +
					"`GOOGLE_WORKSPACE_OBJECT_TYPE`, `HYPERV_OBJECT_TYPE`, `INFORMIX_INSTANCE_OBJECT_TYPE`, `K8S_OBJECT_TYPE`, `KUPR_OBJECT_TYPE`, " +
					"`M365_BACKUP_STORAGE_OBJECT_TYPE`, `MANAGED_VOLUME_OBJECT_TYPE`, `MONGO_OBJECT_TYPE`, `MONGODB_OBJECT_TYPE`, `MSSQL_OBJECT_TYPE`, `MYSQLDB_OBJECT_TYPE`, " +
					"`NAS_OBJECT_TYPE`, `NCD_OBJECT_TYPE`, `NUTANIX_OBJECT_TYPE`, `O365_OBJECT_TYPE`, `OKTA_OBJECT_TYPE`, `OLVM_OBJECT_TYPE`, `OPENSTACK_OBJECT_TYPE`, " +
					"`ORACLE_OBJECT_TYPE`, `POSTGRES_DB_CLUSTER_OBJECT_TYPE`, `PROXMOX_OBJECT_TYPE`, `SALESFORCE_OBJECT_TYPE`, `SAP_HANA_OBJECT_TYPE`, " +
					"`SNAPMIRROR_CLOUD_OBJECT_TYPE`, `VCD_OBJECT_TYPE`, `VOLUME_GROUP_OBJECT_TYPE`, and `VSPHERE_OBJECT_TYPE`. " +
					"Note, `AZURE_SQL_DATABASE_OBJECT_TYPE` cannot be provided at the same time as other object types.",
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
					keyMinuteSchedule,
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
						keyReplicationPair: {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keySourceCluster: {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "Source cluster ID (UUID).",
										ValidateFunc: validation.IsUUID,
									},
									keyTargetCluster: {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "Target cluster ID (UUID).",
										ValidateFunc: validation.IsUUID,
									},
								},
							},
							Optional:    true,
							Description: "Replication pairs specifying source and target clusters.",
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
						keyLocalRetention: {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyRetention: {
										Type:         schema.TypeInt,
										Required:     true,
										Description:  "Local retention on replication target specifies for how long the snapshots are kept on the replication target before being archived.",
										ValidateFunc: validation.IntAtLeast(1),
									},
									keyRetentionUnit: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Local retention unit. Possible values are `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
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
							Description: "Local retention on replication target.",
						},
						keyCascadingArchival: {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									keyArchivalLocationID: {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "Archival location ID (UUID) for cascading archival.",
										ValidateFunc: validation.IsUUID,
									},
									keyArchivalThreshold: {
										Type:         schema.TypeInt,
										Optional:     true,
										Description:  "Archival threshold specifies when to archive replicated snapshots.",
										ValidateFunc: validation.IntAtLeast(1),
									},
									keyArchivalThresholdUnit: {
										Type:     schema.TypeString,
										Optional: true,
										Description: "Archival threshold unit. Possible values are " +
											"`DAYS`, `WEEKS`, `MONTHS`, `QUARTERS` and `YEARS`.",
										ValidateFunc: validation.StringInSlice([]string{
											string(gqlsla.Days),
											string(gqlsla.Weeks),
											string(gqlsla.Months),
											string(gqlsla.Quarters),
											string(gqlsla.Years),
										}, false),
									},
									keyArchivalTiering: {
										Type: schema.TypeList,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												keyInstantTiering: {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "Enable instant tiering to cold storage.",
												},
												keyMinAccessibleDurationInSeconds: {
													Type:         schema.TypeInt,
													Optional:     true,
													Description:  "Minimum duration in seconds that data must remain accessible before tiering.",
													ValidateFunc: validation.IntAtLeast(0),
												},
												keyColdStorageClass: {
													Type:     schema.TypeString,
													Optional: true,
													Description: "Cold storage class for tiering. Possible values are " +
														"`AZURE_ARCHIVE`, `AWS_GLACIER`, `AWS_GLACIER_DEEP_ARCHIVE`.",
													ValidateFunc: validation.StringInSlice([]string{
														string(gqlsla.ColdStorageClassAzureArchive),
														string(gqlsla.ColdStorageClassAWSGlacier),
														string(gqlsla.ColdStorageClassAWSGlacierDeepArchive),
													}, false),
												},
												keyTierExistingSnapshots: {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "Whether to tier existing snapshots to cold storage.",
												},
											},
										},
										MaxItems:    1,
										Optional:    true,
										Description: "Archival tiering specification for cold storage.",
									},
									keyFrequency: {
										Type: schema.TypeSet,
										Elem: &schema.Schema{
											Type: schema.TypeString,
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
										Optional:    true,
										Description: "Frequencies for cascading archival. Possible values are `MINUTE`, `HOURS`, `DAYS`, `WEEKS`, `MONTHS`, `QUARTERS`, `YEARS`.",
									},
								},
							},
							Optional:    true,
							Description: "Cascading archival specifications for replication.",
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
								string(gqlsla.Compliance),
								string(gqlsla.Protection),
							}, false),
						},
						keyRetentionLockComplianceAcknowledgment: {
							Type:     schema.TypeBool,
							Optional: true,
							Description: "Acknowledgment that snapshots protected under compliance mode cannot be deleted " +
								"before the scheduled expiry date. This field must be set to `true` when using `COMPLIANCE` mode. " +
								"Compliance mode is recommended to meet regulations and governance mode is recommended to only " +
								"protect data. Default value is `false`.\n\n" +
								"!> **Warning:** Snapshots protected under compliance mode cannot be deleted before the scheduled expiry date.",
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
							Optional: true,
							Description: "Day of week. Possible values are `MONDAY`, `TUESDAY`, `WEDNESDAY`, " +
								"`THURSDAY`, `FRIDAY`, `SATURDAY` and `SUNDAY`. " +
								"Note: For M365 Backup Storage SLAs, this field should be omitted.",
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
					keyMinuteSchedule,
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
					keyMinuteSchedule,
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

		var groupID uuid.UUID
		var err error
		if alID := archival[keyArchivalLocationID].(string); len(alID) > 0 {
			groupID, err = uuid.Parse(archival[keyArchivalLocationID].(string))
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %s", keyArchivalLocationID, err)
			}
		}

		// Parse archival location to cluster mapping
		var mappings []gqlsla.ArchivalLocationToClusterMapping
		if mappingList, ok := archival[keyArchivalLocationToClusterMapping].([]any); ok {
			for _, m := range mappingList {
				mapping := m.(map[string]any)

				locationID, err := uuid.Parse(mapping[keyArchivalLocationID].(string))
				if err != nil {
					return nil, fmt.Errorf("failed to parse archival location ID in mapping: %s", err)
				}

				var clusterID uuid.UUID
				if clusterIDStr, ok := mapping[keyClusterID].(string); ok && clusterIDStr != "" {
					clusterID, err = uuid.Parse(clusterIDStr)
					if err != nil {
						return nil, fmt.Errorf("failed to parse cluster ID in mapping: %s", err)
					}
				}

				mappings = append(mappings, gqlsla.ArchivalLocationToClusterMapping{
					ClusterID:  clusterID,
					LocationID: locationID,
				})
			}
		}

		// Parse archival tiering
		var tieringSpec *gqlsla.ArchivalTieringSpec
		if tieringList, ok := archival[keyArchivalTiering].([]any); ok && len(tieringList) > 0 {
			tieringMap := tieringList[0].(map[string]any)
			tieringSpec = &gqlsla.ArchivalTieringSpec{
				InstantTiering:                 tieringMap[keyInstantTiering].(bool),
				MinAccessibleDurationInSeconds: int64(tieringMap[keyMinAccessibleDurationInSeconds].(int)),
				ColdStorageClass:               gqlsla.ColdStorageClass(tieringMap[keyColdStorageClass].(string)),
				TierExistingSnapshots:          tieringMap[keyTierExistingSnapshots].(bool),
			}
		}

		archivalSpecs = append(archivalSpecs, gqlsla.ArchivalSpec{
			GroupID:                          groupID,
			Frequencies:                      frequenciesFromSchedule(schedule),
			Threshold:                        archival[keyThreshold].(int),
			ThresholdUnit:                    gqlsla.RetentionUnit(archival[keyThresholdUnit].(string)),
			ArchivalLocationToClusterMapping: mappings,
			ArchivalTieringSpec:              tieringSpec,
		})
	}

	return archivalSpecs, nil
}

// toArchival returns a slice holding the archival configuration.
func toArchival(domain gqlsla.Domain, existing []any) ([]any, error) {
	blocks := make(map[string]map[string]any)
	for _, spec := range domain.ArchivalSpecs {
		id := spec.StorageSetting.ID
		if blocks[id] != nil {
			return nil, fmt.Errorf("archival location %q used multiple times", id)
		}

		// Convert archival location to cluster mapping
		var mappings []any
		for _, mapping := range spec.ArchivalLocationToClusterMapping {
			mappings = append(mappings, map[string]any{
				keyClusterID:          mapping.Cluster.ID,
				keyClusterName:        mapping.Cluster.Name,
				keyArchivalLocationID: mapping.Location.ID,
				keyName:               mapping.Location.Name,
			})
		}

		block := map[string]any{
			keyArchivalLocationID:               id,
			keyThreshold:                        spec.Threshold,
			keyThresholdUnit:                    string(spec.ThresholdUnit),
			keyArchivalLocationToClusterMapping: mappings,
		}

		// Convert archival tiering spec
		if spec.ArchivalTieringSpec != nil {
			tieringMap := map[string]any{
				keyInstantTiering:                 spec.ArchivalTieringSpec.InstantTiering,
				keyMinAccessibleDurationInSeconds: spec.ArchivalTieringSpec.MinAccessibleDurationInSeconds,
				keyColdStorageClass:               spec.ArchivalTieringSpec.ColdStorageClass,
				keyTierExistingSnapshots:          spec.ArchivalTieringSpec.TierExistingSnapshots,
			}
			block[keyArchivalTiering] = []any{tieringMap}
		}

		blocks[id] = block
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
	for _, spec := range domain.ArchivalSpecs {
		id := spec.StorageSetting.ID
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

		// Parse replication pairs
		var replicationPairs []gqlsla.ReplicationPair
		if pairs, ok := spec[keyReplicationPair].([]any); ok {
			for _, pair := range pairs {
				pairMap := pair.(map[string]any)
				replicationPairs = append(replicationPairs, gqlsla.ReplicationPair{
					SourceClusterID: pairMap[keySourceCluster].(string),
					TargetClusterID: pairMap[keyTargetCluster].(string),
				})
			}
		}

		// Parse replication local retention
		var replicationLocalRetention *gqlsla.RetentionDuration
		if localRetentionList, ok := spec[keyLocalRetention].([]any); ok && len(localRetentionList) > 0 {
			localRetentionMap := localRetentionList[0].(map[string]any)
			replicationLocalRetention = &gqlsla.RetentionDuration{
				Duration: localRetentionMap[keyRetention].(int),
				Unit:     gqlsla.RetentionUnit(localRetentionMap[keyRetentionUnit].(string)),
			}
		}

		// Parse cascading archival specs
		var cascadingArchivalSpecs []gqlsla.CascadingArchivalSpec
		if cascadingArchival, ok := spec[keyCascadingArchival].([]any); ok {
			for _, ca := range cascadingArchival {
				caMap := ca.(map[string]any)

				archivalLocationID, err := uuid.Parse(caMap[keyArchivalLocationID].(string))
				if err != nil {
					return nil, fmt.Errorf("invalid archival location ID: %w", err)
				}

				cascadingSpec := gqlsla.CascadingArchivalSpec{}

				// Build archival location to cluster mapping for each target cluster
				// This is the recommended approach instead of using the deprecated ArchivalLocationID
				for _, pair := range replicationPairs {
					targetClusterID, err := uuid.Parse(pair.TargetClusterID)
					if err != nil {
						return nil, fmt.Errorf("invalid target cluster ID: %w", err)
					}
					cascadingSpec.ArchivalLocationToClusterMappings = append(
						cascadingSpec.ArchivalLocationToClusterMappings,
						gqlsla.ArchivalLocationToClusterMapping{
							ClusterID:  targetClusterID,
							LocationID: archivalLocationID,
						},
					)
				}

				// Parse archival threshold
				if threshold, ok := caMap[keyArchivalThreshold].(int); ok && threshold > 0 {
					cascadingSpec.ArchivalThreshold = &gqlsla.RetentionDuration{
						Duration: threshold,
						Unit:     gqlsla.RetentionUnit(caMap[keyArchivalThresholdUnit].(string)),
					}
				}

				// Parse archival tiering
				if tieringList, ok := caMap[keyArchivalTiering].([]any); ok && len(tieringList) > 0 {
					tieringMap := tieringList[0].(map[string]any)
					cascadingSpec.ArchivalTieringSpec = &gqlsla.ArchivalTieringSpec{
						InstantTiering:                 tieringMap[keyInstantTiering].(bool),
						MinAccessibleDurationInSeconds: int64(tieringMap[keyMinAccessibleDurationInSeconds].(int)),
						ColdStorageClass:               gqlsla.ColdStorageClass(tieringMap[keyColdStorageClass].(string)),
						TierExistingSnapshots:          tieringMap[keyTierExistingSnapshots].(bool),
					}
				}

				// Parse frequencies
				if freqSet, ok := caMap[keyFrequency].(*schema.Set); ok {
					for _, freq := range freqSet.List() {
						cascadingSpec.Frequencies = append(cascadingSpec.Frequencies, gqlsla.RetentionUnit(freq.(string)))
					}
				}

				cascadingArchivalSpecs = append(cascadingArchivalSpecs, cascadingSpec)
			}
		}

		replicationSpecs = append(replicationSpecs, gqlsla.ReplicationSpec{
			AWSRegion:                         awsRegion.ToRegionForReplicationEnum(),
			AWSAccount:                        awsCrossAccount,
			AzureRegion:                       azureRegion.ToRegionForReplicationEnum(),
			AzureSubscription:                 azureCrossSubscription,
			RetentionDuration:                 retention,
			ReplicationPairs:                  replicationPairs,
			ReplicationLocalRetentionDuration: replicationLocalRetention,
			CascadingArchivalSpecs:            cascadingArchivalSpecs,
		})
	}

	return replicationSpecs, nil
}

// toReplicationSpec returns a slice holding the replication configuration.
func toReplicationSpec(replicationSpecs []gqlsla.ReplicationSpec) []any {
	var replicationSpec []any
	for _, spec := range replicationSpecs {
		specMap := map[string]any{
			keyAWSRegion:       spec.AWSRegion.Name(),
			keyAWSCrossAccount: spec.AWSAccount,
			keyAzureRegion:     spec.AzureRegion.Name(),
			keyRetention:       spec.RetentionDuration.Duration,
			keyRetentionUnit:   spec.RetentionDuration.Unit,
		}

		var replicationPairs []any
		for _, pair := range spec.ReplicationPairs {
			replicationPairs = append(replicationPairs, map[string]any{
				keySourceCluster: pair.SourceClusterID,
				keyTargetCluster: pair.TargetClusterID,
			})
		}
		specMap[keyReplicationPair] = replicationPairs

		// Add replication local retention (only if present)
		if spec.ReplicationLocalRetentionDuration != nil {
			localRetentionMap := map[string]any{
				keyRetention:     spec.ReplicationLocalRetentionDuration.Duration,
				keyRetentionUnit: spec.ReplicationLocalRetentionDuration.Unit,
			}
			specMap[keyLocalRetention] = []any{localRetentionMap}
		}

		// Add cascading archival specs (only if present)
		if len(spec.CascadingArchivalSpecs) > 0 {
			var cascadingArchival []any
			for _, ca := range spec.CascadingArchivalSpecs {
				caMap := map[string]any{}

				// Extract archival location ID from the mapping
				// The mapping should have the same location ID for all clusters
				if len(ca.ArchivalLocationToClusterMappings) > 0 {
					caMap[keyArchivalLocationID] = ca.ArchivalLocationToClusterMappings[0].LocationID.String()
					//lint:ignore SA1019 Fallback to deprecated field when reading in the else branch below.
				} else if alID := ca.ArchivalLocationID; alID != nil {
					// Fallback to deprecated field if mapping is not present
					caMap[keyArchivalLocationID] = alID.String()
				}

				if ca.ArchivalThreshold != nil {
					caMap[keyArchivalThreshold] = ca.ArchivalThreshold.Duration
					caMap[keyArchivalThresholdUnit] = ca.ArchivalThreshold.Unit
				}

				if ca.ArchivalTieringSpec != nil {
					tieringMap := map[string]any{
						keyInstantTiering:                 ca.ArchivalTieringSpec.InstantTiering,
						keyMinAccessibleDurationInSeconds: ca.ArchivalTieringSpec.MinAccessibleDurationInSeconds,
						keyColdStorageClass:               ca.ArchivalTieringSpec.ColdStorageClass,
						keyTierExistingSnapshots:          ca.ArchivalTieringSpec.TierExistingSnapshots,
					}
					caMap[keyArchivalTiering] = []any{tieringMap}
				}

				if len(ca.Frequencies) > 0 {
					frequencies := &schema.Set{F: schema.HashString}
					for _, freq := range ca.Frequencies {
						frequencies.Add(string(freq))
					}
					caMap[keyFrequency] = frequencies
				}

				cascadingArchival = append(cascadingArchival, caMap)
			}
			specMap[keyCascadingArchival] = cascadingArchival
		}

		replicationSpec = append(replicationSpec, specMap)
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
		awsDynamoDBConfig, err := fromAWSDynamoDBConfig(d)
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
		vmwareVMConfig, err := fromVMwareVMConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		sapHanaConfig, err := fromSapHanaConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		db2Config, err := fromDB2Config(d)
		if err != nil {
			return diag.FromErr(err)
		}
		mssqlConfig, err := fromMssqlConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		oracleConfig, err := fromOracleConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		mongoConfig, err := fromMongoConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		managedVolumeConfig, err := fromManagedVolumeConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		postgresDbClusterConfig, err := fromPostgresDbClusterConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		mysqldbConfig, err := fromMysqldbConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		informixConfig, err := fromInformixConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		gcpCloudSqlConfig, err := fromGcpCloudSqlConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		ncdConfig, err := fromNcdConfig(d)
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
		var retentionLockMode gqlsla.RetentionLockMode
		if rl := d.Get(keyRetentionLock).([]any); len(rl) > 0 {
			rlMap := rl[0].(map[string]any)
			retentionLockMode = gqlsla.RetentionLockMode(rlMap[keyMode].(string))

			// Validate that COMPLIANCE mode requires acknowledgment
			if retentionLockMode == gqlsla.Compliance {
				// Check if the acknowledgment field is set and is true
				acknowledged, ok := rlMap[keyRetentionLockComplianceAcknowledgment].(bool)
				if !ok || !acknowledged {
					return diag.Errorf("compliance_mode_acknowledgment must be set to true when using COMPLIANCE mode. " +
						"This acknowledges that snapshots protected under compliance mode cannot be deleted before the " +
						"scheduled expiry date. Compliance mode is recommended to meet regulations and governance mode " +
						"is recommended to only protect data.")
				}
			}
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
			case gqlsla.ObjectActiveDirectory:
				if schedule.Hourly != nil && schedule.Hourly.BasicSchedule.Frequency < 4 {
					return diag.Errorf("Active Directory object type requires minimum of 4 hours SLA")
				}
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
			case gqlsla.ObjectMicrosoft365:
				if len(snapshotWindows) > 0 {
					return diag.Errorf("Microsoft 365 object type does not support snapshot windows")
				}
				if len(firstFullSnapshotWindows) > 0 {
					return diag.Errorf("Microsoft 365 object type does not support first full snapshot windows")
				}
				if schedule.Hourly != nil && schedule.Hourly.BasicSchedule.Frequency < 8 {
					return diag.Errorf("Microsoft 365 object type requires minimum of 8 hours SLA")
				}
			case gqlsla.ObjectOLVM:
				if len(archivalSpecs) > 0 {
					return diag.Errorf("OLVM object type does not support archival locations")
				}
			case gqlsla.ObjectCassandra:
				if len(archivalSpecs) > 0 {
					return diag.Errorf("Cassandra object type does not support archival locations")
				}
				if len(replicationSpecs) > 0 {
					return diag.Errorf("Cassandra object type does not support replication")
				}
			case gqlsla.ObjectMongoDB:
				if len(replicationSpecs) > 0 {
					return diag.Errorf("MongoDB object type does not support replication")
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
				AWSDynamoDBConfig:               awsDynamoDBConfig,
				AWSS3Config:                     awsS3Config,
				AWSRDSConfig:                    awsRDSConfig,
				AzureBlobConfig:                 blobConfig,
				AzureSQLDatabaseDBConfig:        azureSQLConfig,
				AzureSQLManagedInstanceDBConfig: azureSQLMIConfig,
				VMwareVMConfig:                  vmwareVMConfig,
				SapHanaConfig:                   sapHanaConfig,
				DB2Config:                       db2Config,
				MssqlConfig:                     mssqlConfig,
				OracleConfig:                    oracleConfig,
				MongoConfig:                     mongoConfig,
				ManagedVolumeSlaConfig:          managedVolumeConfig,
				PostgresDbClusterSlaConfig:      postgresDbClusterConfig,
				MysqldbSlaConfig:                mysqldbConfig,
				NcdSlaConfig:                    ncdConfig,
				InformixSlaConfig:               informixConfig,
				GcpCloudSqlConfig:               gcpCloudSqlConfig,
			},
			ObjectTypes:       objectTypes,
			ReplicationSpecs:  replicationSpecs,
			RetentionLock:     retentionLockMode != "" && retentionLockMode != gqlsla.NoLock,
			RetentionLockMode: retentionLockMode,
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

			// When updating a data center archival, RSC requires the group ID to be set to nil.
			for i, spec := range createParams.ArchivalSpecs {
				if len(spec.ArchivalLocationToClusterMapping) > 0 {
					createParams.ArchivalSpecs[i].GroupID = uuid.Nil
				}
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

	archival, err := toArchival(slaDomain, d.Get(keyArchival).([]any))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyArchival, archival); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyAWSDynamoDBConfig, toAWSDynamoDBConfig(slaDomain.ObjectSpecificConfigs.AWSDynamoDBConfig)); err != nil {
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
	if err := d.Set(keyVMwareVMConfig, toVMwareVMConfig(slaDomain.ObjectSpecificConfigs.VMwareVMConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keySapHanaConfig, toSapHanaConfig(slaDomain.ObjectSpecificConfigs.SapHanaConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyDB2Config, toDB2Config(slaDomain.ObjectSpecificConfigs.DB2Config)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyMSSQLConfig, toMssqlConfig(slaDomain.ObjectSpecificConfigs.MssqlConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyOracleConfig, toOracleConfig(slaDomain.ObjectSpecificConfigs.OracleConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyMongoConfig, toMongoConfig(slaDomain.ObjectSpecificConfigs.MongoConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyManagedVolumeConfig, toManagedVolumeConfig(slaDomain.ObjectSpecificConfigs.ManagedVolumeSlaConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyPostgresDBClusterConfig, toPostgresDbClusterConfig(slaDomain.ObjectSpecificConfigs.PostgresDbClusterSlaConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyMySQLDBConfig, toMysqldbConfig(slaDomain.ObjectSpecificConfigs.MysqldbSlaConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyInformixConfig, toInformixConfig(slaDomain.ObjectSpecificConfigs.InformixSlaConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyGCPCloudSQLConfig, toGcpCloudSqlConfig(slaDomain.ObjectSpecificConfigs.GcpCloudSqlConfig)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyNCDConfig, toNcdConfig(slaDomain.ObjectSpecificConfigs.NcdSlaConfig)); err != nil {
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
		var replicationPairs []gqlsla.ReplicationPair
		for _, pair := range spec.ReplicationPairs {
			replicationPairs = append(replicationPairs, gqlsla.ReplicationPair{
				SourceClusterID: pair.SourceCluster.ID,
				TargetClusterID: pair.TargetCluster.ID,
			})
		}

		// Convert cascading archival specs from GraphQL response to SDK structure
		var cascadingArchivalSpecs []gqlsla.CascadingArchivalSpec
		for _, cas := range spec.CascadingArchivalSpecs {
			cascadingSpec := gqlsla.CascadingArchivalSpec{
				ArchivalThreshold: cas.ArchivalThreshold,
				Frequencies:       cas.Frequencies,
			}

			// Convert archival location
			if cas.ArchivalLocation != nil {
				locationID, err := uuid.Parse(cas.ArchivalLocation.ID)
				if err == nil {
					//lint:ignore SA1019 Allow deprecated field when reading
					cascadingSpec.ArchivalLocationID = &locationID
				}
			}

			// Convert archival tiering spec
			if cas.ArchivalTieringSpec != nil {
				cascadingSpec.ArchivalTieringSpec = &gqlsla.ArchivalTieringSpec{
					InstantTiering:                 cas.ArchivalTieringSpec.InstantTiering,
					MinAccessibleDurationInSeconds: cas.ArchivalTieringSpec.MinAccessibleDurationInSeconds,
					ColdStorageClass:               cas.ArchivalTieringSpec.ColdStorageClass,
					TierExistingSnapshots:          cas.ArchivalTieringSpec.TierExistingSnapshots,
				}
			}

			// Convert archival location to cluster mapping
			for _, mapping := range cas.ArchivalLocationToClusterMapping {
				clusterID, err := uuid.Parse(mapping.Cluster.ID)
				if err != nil {
					continue
				}
				locationID, err := uuid.Parse(mapping.Location.ID)
				if err != nil {
					continue
				}
				cascadingSpec.ArchivalLocationToClusterMappings = append(
					cascadingSpec.ArchivalLocationToClusterMappings,
					gqlsla.ArchivalLocationToClusterMapping{
						ClusterID:  clusterID,
						LocationID: locationID,
					},
				)
			}

			cascadingArchivalSpecs = append(cascadingArchivalSpecs, cascadingSpec)
		}

		replicationSpecs = append(replicationSpecs, gqlsla.ReplicationSpec{
			AWSRegion:   spec.AWSRegion,
			AWSAccount:  spec.AWS.AccountID,
			AzureRegion: spec.AzureRegion,
			RetentionDuration: &gqlsla.RetentionDuration{
				Duration: spec.RetentionDuration.Duration,
				Unit:     spec.RetentionDuration.Unit,
			},
			ReplicationLocalRetentionDuration: spec.ReplicationLocalRetentionDuration,
			CascadingArchivalSpecs:            cascadingArchivalSpecs,
			ReplicationPairs:                  replicationPairs,
		})
	}
	if err := d.Set(keyReplicationSpec, toReplicationSpec(replicationSpecs)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyRetentionLock, toRetentionLock(slaDomain)); err != nil {
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

// importSLADomain imports an SLA domain by ID (UUID) or name. If the import ID
// is a valid UUID, the SLA domain is looked up by ID. Otherwise, the SLA domain
// is looked up by name.
func importSLADomain(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	log.Print("[TRACE] importSLADomain")

	client, err := m.(*client).polaris()
	if err != nil {
		return nil, err
	}

	importID := d.Id()

	// Try to parse the import ID as a UUID.
	id, err := uuid.Parse(importID)
	if err != nil {
		// If it's not a UUID, treat it as a name and look up the SLA domain.
		slaDomain, err := sla.Wrap(client).DomainByName(ctx, importID)
		if err != nil {
			return nil, fmt.Errorf("failed to find SLA domain by name %q: %w", importID, err)
		}
		d.SetId(slaDomain.ID.String())
		return []*schema.ResourceData{d}, nil
	}
	// Verify the SLA domain exists.
	if _, err := sla.Wrap(client).DomainByID(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to find SLA domain by ID %q: %w", importID, err)
	}

	d.SetId(id.String())
	return []*schema.ResourceData{d}, nil
}

func fromAWSDynamoDBConfig(d *schema.ResourceData) (*gqlsla.AWSDynamoDBConfig, error) {
	block, ok := d.GetOk(keyAWSDynamoDBConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	dynamoDBConfig := block.([]any)[0].(map[string]any)
	kmsAlias, ok := dynamoDBConfig[keyKMSAlias].(string)
	if !ok {
		return nil, nil
	}
	return &gqlsla.AWSDynamoDBConfig{
		KMSAliasForPrimaryBackup: kmsAlias,
	}, nil
}

func toAWSDynamoDBConfig(dynamoDBConfig *gqlsla.AWSDynamoDBConfig) []any {
	if dynamoDBConfig == nil {
		return nil
	}

	return []any{map[string]any{
		keyKMSAlias: dynamoDBConfig.KMSAliasForPrimaryBackup,
	}}
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

func fromVMwareVMConfig(d *schema.ResourceData) (*gqlsla.VMwareVMConfig, error) {
	block, ok := d.GetOk(keyVMwareVMConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	vmwareConfig := block.([]any)[0].(map[string]any)
	return &gqlsla.VMwareVMConfig{
		LogRetentionSeconds: int64(vmwareConfig[keyLogRetention].(int)),
	}, nil
}

func toVMwareVMConfig(vmwareConfig *gqlsla.VMwareVMConfig) []any {
	if vmwareConfig == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention: int(vmwareConfig.LogRetentionSeconds),
	}}
}

func fromSapHanaConfig(d *schema.ResourceData) (*gqlsla.SapHanaConfig, error) {
	block, ok := d.GetOk(keySapHanaConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	sapHanaConfig := &gqlsla.SapHanaConfig{}

	if freq, ok := config[keyIncrementalFrequency].(int); ok && freq > 0 {
		sapHanaConfig.IncrementalFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyIncrementalFrequencyUnit].(string)),
		}
	}
	if ret, ok := config[keyLogRetention].(int); ok && ret > 0 {
		sapHanaConfig.LogRetention = gqlsla.RetentionDuration{
			Duration: ret,
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		}
	}
	if freq, ok := config[keyDifferentialFrequency].(int); ok && freq > 0 {
		sapHanaConfig.DifferentialFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyDifferentialFrequencyUnit].(string)),
		}
	}

	if snapConfig, ok := config[keyStorageSnapshotConfig].([]any); ok && len(snapConfig) > 0 && snapConfig[0] != nil {
		snap := snapConfig[0].(map[string]any)
		sapHanaConfig.StorageSnapshotConfig = &gqlsla.SapHanaStorageSnapshotConfig{
			Frequency: gqlsla.RetentionDuration{
				Duration: snap[keyFrequency].(int),
				Unit:     gqlsla.RetentionUnit(snap[keyFrequencyUnit].(string)),
			},
			Retention: gqlsla.RetentionDuration{
				Duration: snap[keyRetention].(int),
				Unit:     gqlsla.RetentionUnit(snap[keyRetentionUnit].(string)),
			},
		}
	}

	return sapHanaConfig, nil
}

func toSapHanaConfig(config *gqlsla.SapHanaConfig) []any {
	if config == nil {
		return nil
	}

	result := map[string]any{}
	if config.IncrementalFrequency.Duration > 0 {
		result[keyIncrementalFrequency] = config.IncrementalFrequency.Duration
		result[keyIncrementalFrequencyUnit] = config.IncrementalFrequency.Unit
	}
	if config.LogRetention.Duration > 0 {
		result[keyLogRetention] = config.LogRetention.Duration
		result[keyLogRetentionUnit] = config.LogRetention.Unit
	}
	if config.DifferentialFrequency.Duration > 0 {
		result[keyDifferentialFrequency] = config.DifferentialFrequency.Duration
		result[keyDifferentialFrequencyUnit] = config.DifferentialFrequency.Unit
	}
	if config.StorageSnapshotConfig != nil {
		result[keyStorageSnapshotConfig] = []any{map[string]any{
			keyFrequency:     config.StorageSnapshotConfig.Frequency.Duration,
			keyFrequencyUnit: config.StorageSnapshotConfig.Frequency.Unit,
			keyRetention:     config.StorageSnapshotConfig.Retention.Duration,
			keyRetentionUnit: config.StorageSnapshotConfig.Retention.Unit,
		}}
	}

	return []any{result}
}

func fromDB2Config(d *schema.ResourceData) (*gqlsla.DB2Config, error) {
	block, ok := d.GetOk(keyDB2Config)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	db2Config := &gqlsla.DB2Config{}

	if freq, ok := config[keyIncrementalFrequency].(int); ok && freq > 0 {
		db2Config.IncrementalFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyIncrementalFrequencyUnit].(string)),
		}
	}
	if ret, ok := config[keyLogRetention].(int); ok && ret > 0 {
		db2Config.LogRetention = gqlsla.RetentionDuration{
			Duration: ret,
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		}
	}
	if freq, ok := config[keyDifferentialFrequency].(int); ok && freq > 0 {
		db2Config.DifferentialFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyDifferentialFrequencyUnit].(string)),
		}
	}
	if method, ok := config[keyLogArchivalMethod].(string); ok && method != "" {
		db2Config.LogArchivalMethod = gqlsla.Db2LogArchivalMethod(method)
	}

	return db2Config, nil
}

func toDB2Config(config *gqlsla.DB2Config) []any {
	if config == nil {
		return nil
	}

	result := map[string]any{}
	if config.IncrementalFrequency.Duration > 0 {
		result[keyIncrementalFrequency] = config.IncrementalFrequency.Duration
		result[keyIncrementalFrequencyUnit] = config.IncrementalFrequency.Unit
	}
	if config.LogRetention.Duration > 0 {
		result[keyLogRetention] = config.LogRetention.Duration
		result[keyLogRetentionUnit] = config.LogRetention.Unit
	}
	if config.DifferentialFrequency.Duration > 0 {
		result[keyDifferentialFrequency] = config.DifferentialFrequency.Duration
		result[keyDifferentialFrequencyUnit] = config.DifferentialFrequency.Unit
	}
	if config.LogArchivalMethod != "" {
		result[keyLogArchivalMethod] = string(config.LogArchivalMethod)
	}

	return []any{result}
}

func fromMssqlConfig(d *schema.ResourceData) (*gqlsla.MssqlConfig, error) {
	block, ok := d.GetOk(keyMSSQLConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.MssqlConfig{
		Frequency: gqlsla.RetentionDuration{
			Duration: config[keyFrequency].(int),
			Unit:     gqlsla.RetentionUnit(config[keyFrequencyUnit].(string)),
		},
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		},
	}, nil
}

func toMssqlConfig(config *gqlsla.MssqlConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:        config.Frequency.Duration,
		keyFrequencyUnit:    config.Frequency.Unit,
		keyLogRetention:     config.LogRetention.Duration,
		keyLogRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromOracleConfig(d *schema.ResourceData) (*gqlsla.OracleConfig, error) {
	block, ok := d.GetOk(keyOracleConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	oracleConfig := &gqlsla.OracleConfig{
		Frequency: gqlsla.RetentionDuration{
			Duration: config[keyFrequency].(int),
			Unit:     gqlsla.RetentionUnit(config[keyFrequencyUnit].(string)),
		},
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		},
	}

	if hostRet, ok := config[keyHostLogRetention].(int); ok && hostRet > 0 {
		oracleConfig.HostLogRetention = gqlsla.RetentionDuration{
			Duration: hostRet,
			Unit:     gqlsla.RetentionUnit(config[keyHostLogRetentionUnit].(string)),
		}
	}

	return oracleConfig, nil
}

func toOracleConfig(config *gqlsla.OracleConfig) []any {
	if config == nil {
		return nil
	}

	result := map[string]any{
		keyFrequency:        config.Frequency.Duration,
		keyFrequencyUnit:    config.Frequency.Unit,
		keyLogRetention:     config.LogRetention.Duration,
		keyLogRetentionUnit: config.LogRetention.Unit,
	}

	if config.HostLogRetention.Duration > 0 {
		result[keyHostLogRetention] = config.HostLogRetention.Duration
		result[keyHostLogRetentionUnit] = config.HostLogRetention.Unit
	}

	return []any{result}
}

func fromMongoConfig(d *schema.ResourceData) (*gqlsla.MongoConfig, error) {
	block, ok := d.GetOk(keyMongoConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.MongoConfig{
		LogFrequency: gqlsla.RetentionDuration{
			Duration: config[keyFrequency].(int),
			Unit:     gqlsla.RetentionUnit(config[keyFrequencyUnit].(string)),
		},
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyRetentionUnit].(string)),
		},
	}, nil
}

func toMongoConfig(config *gqlsla.MongoConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     config.LogFrequency.Duration,
		keyFrequencyUnit: config.LogFrequency.Unit,
		keyRetention:     config.LogRetention.Duration,
		keyRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromManagedVolumeConfig(d *schema.ResourceData) (*gqlsla.ManagedVolumeSlaConfig, error) {
	block, ok := d.GetOk(keyManagedVolumeConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.ManagedVolumeSlaConfig{
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		},
	}, nil
}

func toManagedVolumeConfig(config *gqlsla.ManagedVolumeSlaConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention:     config.LogRetention.Duration,
		keyLogRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromPostgresDbClusterConfig(d *schema.ResourceData) (*gqlsla.PostgresDbClusterSlaConfig, error) {
	block, ok := d.GetOk(keyPostgresDBClusterConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.PostgresDbClusterSlaConfig{
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		},
	}, nil
}

func toPostgresDbClusterConfig(config *gqlsla.PostgresDbClusterSlaConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention:     config.LogRetention.Duration,
		keyLogRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromMysqldbConfig(d *schema.ResourceData) (*gqlsla.MysqldbSlaConfig, error) {
	block, ok := d.GetOk(keyMySQLDBConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.MysqldbSlaConfig{
		LogFrequency: gqlsla.RetentionDuration{
			Duration: config[keyFrequency].(int),
			Unit:     gqlsla.RetentionUnit(config[keyFrequencyUnit].(string)),
		},
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyRetentionUnit].(string)),
		},
	}, nil
}

func toMysqldbConfig(config *gqlsla.MysqldbSlaConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyFrequency:     config.LogFrequency.Duration,
		keyFrequencyUnit: config.LogFrequency.Unit,
		keyRetention:     config.LogRetention.Duration,
		keyRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromInformixConfig(d *schema.ResourceData) (*gqlsla.InformixSlaConfig, error) {
	block, ok := d.GetOk(keyInformixConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	informixConfig := &gqlsla.InformixSlaConfig{}

	if freq, ok := config[keyIncrementalFrequency].(int); ok && freq > 0 {
		informixConfig.IncrementalFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyIncrementalFrequencyUnit].(string)),
		}
	}
	if ret, ok := config[keyIncrementalRetention].(int); ok && ret > 0 {
		informixConfig.IncrementalRetention = gqlsla.RetentionDuration{
			Duration: ret,
			Unit:     gqlsla.RetentionUnit(config[keyIncrementalRetentionUnit].(string)),
		}
	}
	if freq, ok := config[keyFrequency].(int); ok && freq > 0 {
		informixConfig.LogFrequency = gqlsla.RetentionDuration{
			Duration: freq,
			Unit:     gqlsla.RetentionUnit(config[keyFrequencyUnit].(string)),
		}
	}
	if ret, ok := config[keyRetention].(int); ok && ret > 0 {
		informixConfig.LogRetention = gqlsla.RetentionDuration{
			Duration: ret,
			Unit:     gqlsla.RetentionUnit(config[keyRetentionUnit].(string)),
		}
	}

	return informixConfig, nil
}

func toInformixConfig(config *gqlsla.InformixSlaConfig) []any {
	if config == nil {
		return nil
	}

	result := map[string]any{}
	if config.IncrementalFrequency.Duration > 0 {
		result[keyIncrementalFrequency] = config.IncrementalFrequency.Duration
		result[keyIncrementalFrequencyUnit] = config.IncrementalFrequency.Unit
	}
	if config.IncrementalRetention.Duration > 0 {
		result[keyIncrementalRetention] = config.IncrementalRetention.Duration
		result[keyIncrementalRetentionUnit] = config.IncrementalRetention.Unit
	}
	if config.LogFrequency.Duration > 0 {
		result[keyFrequency] = config.LogFrequency.Duration
		result[keyFrequencyUnit] = config.LogFrequency.Unit
	}
	if config.LogRetention.Duration > 0 {
		result[keyRetention] = config.LogRetention.Duration
		result[keyRetentionUnit] = config.LogRetention.Unit
	}

	return []any{result}
}

func fromGcpCloudSqlConfig(d *schema.ResourceData) (*gqlsla.GcpCloudSqlConfig, error) {
	block, ok := d.GetOk(keyGCPCloudSQLConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	return &gqlsla.GcpCloudSqlConfig{
		LogRetention: gqlsla.RetentionDuration{
			Duration: config[keyLogRetention].(int),
			Unit:     gqlsla.RetentionUnit(config[keyLogRetentionUnit].(string)),
		},
	}, nil
}

func toGcpCloudSqlConfig(config *gqlsla.GcpCloudSqlConfig) []any {
	if config == nil {
		return nil
	}

	return []any{map[string]any{
		keyLogRetention:     config.LogRetention.Duration,
		keyLogRetentionUnit: config.LogRetention.Unit,
	}}
}

func fromNcdConfig(d *schema.ResourceData) (*gqlsla.NcdSlaConfig, error) {
	block, ok := d.GetOk(keyNCDConfig)
	if !ok {
		return nil, nil
	}

	if len(block.([]any)) == 0 || block.([]any)[0] == nil {
		return nil, nil
	}

	config := block.([]any)[0].(map[string]any)
	ncdConfig := &gqlsla.NcdSlaConfig{}

	parseUUIDs := func(key string) []uuid.UUID {
		if locs, ok := config[key].([]any); ok && len(locs) > 0 {
			var uuids []uuid.UUID
			for _, loc := range locs {
				if id, err := uuid.Parse(loc.(string)); err == nil {
					uuids = append(uuids, id)
				}
			}
			return uuids
		}
		return nil
	}

	ncdConfig.MinutelyBackupLocations = parseUUIDs(keyMinutelyBackupLocations)
	ncdConfig.HourlyBackupLocations = parseUUIDs(keyHourlyBackupLocations)
	ncdConfig.DailyBackupLocations = parseUUIDs(keyDailyBackupLocations)
	ncdConfig.WeeklyBackupLocations = parseUUIDs(keyWeeklyBackupLocations)
	ncdConfig.MonthlyBackupLocations = parseUUIDs(keyMonthlyBackupLocations)
	ncdConfig.QuarterlyBackupLocations = parseUUIDs(keyQuarterlyBackupLocations)
	ncdConfig.YearlyBackupLocations = parseUUIDs(keyYearlyBackupLocations)

	return ncdConfig, nil
}

func toNcdConfig(config *gqlsla.NcdSlaConfig) []any {
	if config == nil {
		return nil
	}

	uuidsToStrings := func(uuids []uuid.UUID) []any {
		if len(uuids) == 0 {
			return nil
		}
		var strs []any
		for _, id := range uuids {
			strs = append(strs, id.String())
		}
		return strs
	}

	result := map[string]any{}
	if locs := uuidsToStrings(config.MinutelyBackupLocations); locs != nil {
		result[keyMinutelyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.HourlyBackupLocations); locs != nil {
		result[keyHourlyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.DailyBackupLocations); locs != nil {
		result[keyDailyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.WeeklyBackupLocations); locs != nil {
		result[keyWeeklyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.MonthlyBackupLocations); locs != nil {
		result[keyMonthlyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.QuarterlyBackupLocations); locs != nil {
		result[keyQuarterlyBackupLocations] = locs
	}
	if locs := uuidsToStrings(config.YearlyBackupLocations); locs != nil {
		result[keyYearlyBackupLocations] = locs
	}

	return []any{result}
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
	weeklySchedule := &gqlsla.WeeklySnapshotSchedule{
		BasicSchedule: gqlsla.BasicSnapshotSchedule{
			Frequency:     schedule[keyFrequency].(int),
			Retention:     schedule[keyRetention].(int),
			RetentionUnit: gqlsla.RetentionUnit(schedule[keyRetentionUnit].(string)),
		},
	}

	// Only set DayOfWeek if it's provided. For M365 Backup Storage SLAs,
	// the day_of_week field should be omitted.
	if dayOfWeek, ok := schedule[keyDayOfWeek].(string); ok && dayOfWeek != "" {
		weeklySchedule.DayOfWeek = gqlsla.Day(dayOfWeek)
	}

	return weeklySchedule
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

func toRetentionLock(slaDomain gqlsla.Domain) []any {
	if !slaDomain.RetentionLock {
		return nil
	}

	mode := slaDomain.RetentionLockMode
	if mode != gqlsla.Compliance && mode != gqlsla.Protection {
		mode = gqlsla.NoLock
	}

	// Set acknowledgment to true if mode is COMPLIANCE, false otherwise
	// This ensures the state reflects the requirement
	acknowledged := mode == gqlsla.Compliance

	return []any{map[string]any{
		keyMode:                                  mode,
		keyRetentionLockComplianceAcknowledgment: acknowledged,
	}}
}
