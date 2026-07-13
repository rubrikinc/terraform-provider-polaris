// Copyright 2026 Rubrik, Inc.
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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// requireAzureSQLSLARevampFeatureFlag skips the test if the
// CNP_AZURE_SQL_SLA_REVAMP feature is not enabled for the account.
func requireAzureSQLSLARevampFeatureFlag(t *testing.T) {
	t.Helper()

	credentials := os.Getenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE")
	if credentials == "" {
		t.Skip("RUBRIK_POLARIS_SERVICEACCOUNT_FILE not set")
	}

	ctx := context.Background()
	c, err := newClient(ctx, credentials, polaris.CacheParams{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	const azureSQLSLARevamp = core.FeatureFlagName("CNP_AZURE_SQL_SLA_REVAMP")
	if !c.flag(ctx, azureSQLSLARevamp) {
		t.Skipf("feature flag %s is not enabled", azureSQLSLARevamp)
	}
}

// azureSQLTestResource holds the per-test values injected into the templates
// below via the testConfig.Resource field.
type azureSQLTestResource struct {
	WeeklyRetention       int
	BackupLocationGroupID string
}

func azureSQLTestConfig(resource azureSQLTestResource) testConfig {
	return testConfig{
		Provider: struct{ Credentials string }{
			Credentials: os.Getenv("RUBRIK_POLARIS_SERVICEACCOUNT_FILE"),
		},
		Resource: resource,
	}
}

// V1 (Azure-managed, long-term retention) Azure SQL Database SLA — carries
// ltr_config, and no Rubrik snapshot schedule or backup location.
const slaDomainAzureSQLV1Tmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_sla_domain" "azure_sql_v1" {
	name         = "Test SLA Azure SQL V1"
	description  = "V1 Azure-managed Azure SQL Database SLA"
	object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE"]

	azure_sql_database_config {
		log_retention = 7
		ltr_config {
			weekly_retention {
				retention      = {{ .Resource.WeeklyRetention }}
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

data "polaris_sla_domain" "azure_sql_v1" {
	name = polaris_sla_domain.azure_sql_v1.name
}
`

// V1 SLA combining the Azure SQL Database and Managed Instance object types.
const slaDomainAzureSQLDbMiTmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_sla_domain" "azure_sql_db_mi" {
	name         = "Test SLA Azure SQL DB and MI"
	description  = "V1 Azure SQL Database and Managed Instance SLA"
	object_types = ["AZURE_SQL_DATABASE_OBJECT_TYPE", "AZURE_SQL_MANAGED_INSTANCE_OBJECT_TYPE"]

	azure_sql_database_config {
		log_retention = 7
		ltr_config {
			weekly_retention {
				retention      = 1
				retention_unit = "WEEKS"
			}
		}
	}

	azure_sql_managed_instance_config {
		log_retention = 7
		ltr_config {
			weekly_retention {
				retention      = 1
				retention_unit = "WEEKS"
			}
		}
	}
}
`

// V2 (Rubrik-managed) Azure SQL Database SLA — a snapshot schedule plus a
// backup location, and no ltr_config.
const slaDomainAzureSQLV2Tmpl = `
provider "polaris" {
	credentials = "{{ .Provider.Credentials }}"
}

resource "polaris_sla_domain" "azure_sql_v2" {
	name         = "Test SLA Azure SQL V2"
	description  = "V2 Rubrik-managed Azure SQL Database SLA"
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
		archival_group_id = "{{ .Resource.BackupLocationGroupID }}"
	}
}

data "polaris_sla_domain" "azure_sql_v2" {
	name = polaris_sla_domain.azure_sql_v2.name
}
`

// TestAccPolarisSLADomain_azureSqlV1 verifies create, read, and update of a V1
// (Azure-managed) Azure SQL Database SLA, including that ltr_config round-trips,
// backup_type reads back as NATIVE, and a retention value can be updated.
func TestAccPolarisSLADomain_azureSqlV1(t *testing.T) {
	requireAzureSQLSLARevampFeatureFlag(t)

	create, err := makeTerraformConfig(azureSQLTestConfig(azureSQLTestResource{WeeklyRetention: 4}), slaDomainAzureSQLV1Tmpl)
	if err != nil {
		t.Fatal(err)
	}
	update, err := makeTerraformConfig(azureSQLTestConfig(azureSQLTestResource{WeeklyRetention: 6}), slaDomainAzureSQLV1Tmpl)
	if err != nil {
		t.Fatal(err)
	}

	const res = "polaris_sla_domain.azure_sql_v1"
	const ds = "data.polaris_sla_domain.azure_sql_v1"

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: create,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(res, "object_types.#", "1"),
				resource.TestCheckResourceAttr(res, "backup_type", "NATIVE"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.log_retention", "7"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.weekly_retention.0.retention", "4"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.weekly_retention.0.retention_unit", "WEEKS"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.monthly_retention.0.retention", "12"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.yearly_retention.0.retention", "7"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.yearly_retention.0.week_of_year", "1"),
				// A V1 SLA has no Rubrik snapshot schedule.
				resource.TestCheckResourceAttr(res, "daily_schedule.#", "0"),
				resource.TestCheckResourceAttr(res, "hourly_schedule.#", "0"),
				// The data source returns the same values.
				resource.TestCheckResourceAttr(ds, "backup_type", "NATIVE"),
				resource.TestCheckResourceAttr(ds, "azure_sql_database_config.0.ltr_config.0.weekly_retention.0.retention", "4"),
			),
		}, {
			Config: update,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.weekly_retention.0.retention", "6"),
				resource.TestCheckResourceAttr(res, "backup_type", "NATIVE"),
			),
		}},
	})
}

// TestAccPolarisSLADomain_azureSqlDbMi verifies that the Azure SQL Database and
// Managed Instance object types can be combined in a single SLA.
func TestAccPolarisSLADomain_azureSqlDbMi(t *testing.T) {
	requireAzureSQLSLARevampFeatureFlag(t)

	config, err := makeTerraformConfig(azureSQLTestConfig(azureSQLTestResource{}), slaDomainAzureSQLDbMiTmpl)
	if err != nil {
		t.Fatal(err)
	}

	const res = "polaris_sla_domain.azure_sql_db_mi"

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(res, "object_types.#", "2"),
				resource.TestCheckResourceAttr(res, "backup_type", "NATIVE"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.0.weekly_retention.0.retention", "1"),
				resource.TestCheckResourceAttr(res, "azure_sql_managed_instance_config.0.ltr_config.0.weekly_retention.0.retention", "1"),
			),
		}},
	})
}

// TestAccPolarisSLADomain_azureSqlV2 verifies create and read of a V2
// (Rubrik-managed) Azure SQL Database SLA. It requires an archival group to use
// as the backup location, provided via the TEST_AZURE_SQL_BACKUP_LOCATION_GROUP_ID
// environment variable; the test is skipped when it is not set.
func TestAccPolarisSLADomain_azureSqlV2(t *testing.T) {
	requireAzureSQLSLARevampFeatureFlag(t)

	groupID := os.Getenv("TEST_AZURE_SQL_BACKUP_LOCATION_GROUP_ID")
	if groupID == "" {
		t.Skip("TEST_AZURE_SQL_BACKUP_LOCATION_GROUP_ID not set")
	}

	config, err := makeTerraformConfig(azureSQLTestConfig(azureSQLTestResource{BackupLocationGroupID: groupID}), slaDomainAzureSQLV2Tmpl)
	if err != nil {
		t.Fatal(err)
	}

	const res = "polaris_sla_domain.azure_sql_v2"
	const ds = "data.polaris_sla_domain.azure_sql_v2"

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(res, "backup_type", "RUBRIK"),
				resource.TestCheckResourceAttr(res, "azure_sql_database_config.0.ltr_config.#", "0"),
				resource.TestCheckResourceAttr(res, "backup_location.0.archival_group_id", groupID),
				resource.TestCheckResourceAttr(res, "hourly_schedule.0.frequency", "1"),
				resource.TestCheckResourceAttr(ds, "backup_type", "RUBRIK"),
			),
		}},
	})
}
