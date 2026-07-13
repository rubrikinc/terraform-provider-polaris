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
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	gqlsla "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
)

// TestValidateAzureSQLSLA covers the V1/V2 separation rules for Azure SQL DB and
// MI SLAs without needing RSC. A V1 (Azure-managed) SLA carries an LTR config and
// must not specify a backup location, Rubrik schedule, or archival location. A V2
// (Rubrik-managed) SLA omits the LTR config and must specify a backup location
// and a schedule, and must not use the legacy archival block.
func TestValidateAzureSQLSLA(t *testing.T) {
	ltr := &gqlsla.AzureSQLLTRConfig{
		WeeklyBackupRetention: &gqlsla.AzureSQLLTRRetention{
			Retention:     4,
			RetentionUnit: gqlsla.Weeks,
		},
	}
	withSchedule := gqlsla.SnapshotSchedule{Daily: &gqlsla.DailySnapshotSchedule{}}
	var noSchedule gqlsla.SnapshotSchedule
	backupLoc := []gqlsla.BackupLocationSpec{{ArchivalGroupID: uuid.New()}}
	archival := []gqlsla.ArchivalSpec{{}}

	tests := []struct {
		name            string
		config          *gqlsla.AzureDBConfig
		schedule        gqlsla.SnapshotSchedule
		backupLocations []gqlsla.BackupLocationSpec
		archivalSpecs   []gqlsla.ArchivalSpec
		wantErr         string // substring; "" means no error expected
	}{
		{
			name:     "V1 valid: ltr config only, no schedule or backup location",
			config:   &gqlsla.AzureDBConfig{LogRetentionInDays: 7, LTRConfig: ltr},
			schedule: noSchedule,
			wantErr:  "",
		},
		{
			name:            "V1 invalid: ltr config with backup location",
			config:          &gqlsla.AzureDBConfig{LTRConfig: ltr},
			schedule:        noSchedule,
			backupLocations: backupLoc,
			wantErr:         "must not specify a backup_location",
		},
		{
			name:     "V1 invalid: ltr config with Rubrik schedule",
			config:   &gqlsla.AzureDBConfig{LTRConfig: ltr},
			schedule: withSchedule,
			wantErr:  "must not specify a Rubrik snapshot schedule",
		},
		{
			name:          "V1 invalid: ltr config with archival block",
			config:        &gqlsla.AzureDBConfig{LTRConfig: ltr},
			schedule:      noSchedule,
			archivalSpecs: archival,
			wantErr:       "must not specify an archival location",
		},
		{
			name:            "V2 valid: backup location and schedule",
			config:          &gqlsla.AzureDBConfig{LogRetentionInDays: 7},
			schedule:        withSchedule,
			backupLocations: backupLoc,
			wantErr:         "",
		},
		{
			name:     "V2 invalid: missing backup location",
			config:   &gqlsla.AzureDBConfig{LogRetentionInDays: 7},
			schedule: withSchedule,
			wantErr:  "requires a backup_location",
		},
		{
			name:            "V2 invalid: backup location but no schedule",
			config:          &gqlsla.AzureDBConfig{LogRetentionInDays: 7},
			schedule:        noSchedule,
			backupLocations: backupLoc,
			wantErr:         "requires a snapshot schedule",
		},
		{
			name:            "V2 invalid: legacy archival block present",
			config:          &gqlsla.AzureDBConfig{LogRetentionInDays: 7},
			schedule:        withSchedule,
			backupLocations: backupLoc,
			archivalSpecs:   archival,
			wantErr:         "remove the archival block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAzureSQLSLA("Azure SQL Database", tt.config, tt.schedule, tt.backupLocations, tt.archivalSpecs)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("expected no error, got: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("expected error containing %q, got: %q", tt.wantErr, err.Error())
			}
		})
	}
}

// TestValidateAzureSQLObjectTypeFeatureGate verifies the CNP_AZURE_SQL_SLA_REVAMP
// gating: when disabled, the legacy Azure SQL model applies (instant archival
// required, ltr_config rejected); when enabled, the V1/V2 model applies.
func TestValidateAzureSQLObjectTypeFeatureGate(t *testing.T) {
	ltr := &gqlsla.AzureDBConfig{LTRConfig: &gqlsla.AzureSQLLTRConfig{
		WeeklyBackupRetention: &gqlsla.AzureSQLLTRRetention{Retention: 1, RetentionUnit: gqlsla.Weeks},
	}}
	plain := &gqlsla.AzureDBConfig{LogRetentionInDays: 7}
	instantArchival := []gqlsla.ArchivalSpec{{Threshold: 0}}
	delayedArchival := []gqlsla.ArchivalSpec{{Threshold: 1}}
	backupLoc := []gqlsla.BackupLocationSpec{{ArchivalGroupID: uuid.New()}}
	withSchedule := gqlsla.SnapshotSchedule{Daily: &gqlsla.DailySnapshotSchedule{}}
	dbTypes := []any{string(gqlsla.ObjectAzureSQLDatabase)}
	miTypes := []any{string(gqlsla.ObjectAzureSQLManagedInstance)}

	t.Run("DB legacy: instant archival accepted", func(t *testing.T) {
		if err := validateAzureSQLDatabaseObjectType(false, dbTypes, plain, gqlsla.SnapshotSchedule{}, nil, instantArchival, nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("DB legacy: ltr_config rejected with feature error", func(t *testing.T) {
		err := validateAzureSQLDatabaseObjectType(false, dbTypes, ltr, gqlsla.SnapshotSchedule{}, nil, instantArchival, nil)
		if !errors.Is(err, errLTRConfigRequiresFeature) {
			t.Fatalf("expected errLTRConfigRequiresFeature, got %v", err)
		}
	})
	t.Run("DB legacy: non-instant archival rejected", func(t *testing.T) {
		err := validateAzureSQLDatabaseObjectType(false, dbTypes, plain, gqlsla.SnapshotSchedule{}, nil, delayedArchival, nil)
		if err == nil || !strings.Contains(err.Error(), "instant archival") {
			t.Fatalf("expected instant archival error, got %v", err)
		}
	})
	t.Run("DB revamp: V2 backup location accepted", func(t *testing.T) {
		if err := validateAzureSQLDatabaseObjectType(true, dbTypes, plain, withSchedule, backupLoc, nil, nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("MI legacy: plain accepted", func(t *testing.T) {
		if err := validateAzureSQLManagedInstanceObjectType(false, miTypes, plain, nil, gqlsla.SnapshotSchedule{}, nil, nil, nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("MI legacy: ltr_config rejected with feature error", func(t *testing.T) {
		err := validateAzureSQLManagedInstanceObjectType(false, miTypes, ltr, nil, gqlsla.SnapshotSchedule{}, nil, nil, nil)
		if !errors.Is(err, errLTRConfigRequiresFeature) {
			t.Fatalf("expected errLTRConfigRequiresFeature, got %v", err)
		}
	})
	t.Run("MI legacy: archival rejected", func(t *testing.T) {
		err := validateAzureSQLManagedInstanceObjectType(false, miTypes, plain, nil, gqlsla.SnapshotSchedule{}, nil, instantArchival, nil)
		if err == nil || !strings.Contains(err.Error(), "does not support archival") {
			t.Fatalf("expected archival error, got %v", err)
		}
	})
}

// TestConfigHasLTRConfig verifies detection of a V1 (Azure-managed) config —
// an azure_sql_*_config block carrying a non-empty ltr_config.
func TestConfigHasLTRConfig(t *testing.T) {
	v1 := []any{map[string]any{
		keyLogRetention: 7,
		keyLTRConfig:    []any{map[string]any{keyWeeklyRetention: []any{}}},
	}}
	v2 := []any{map[string]any{
		keyLogRetention: 7,
		keyLTRConfig:    []any{},
	}}
	tests := []struct {
		name string
		in   any
		want bool
	}{
		{"V1 with ltr_config", v1, true},
		{"V2 without ltr_config", v2, false},
		{"empty block", []any{}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configHasLTRConfig(tt.in); got != tt.want {
				t.Fatalf("configHasLTRConfig(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestOnlyAzureSQLObjectTypes verifies that Azure SQL DB and MI may be combined
// with each other but not with any other object type.
func TestOnlyAzureSQLObjectTypes(t *testing.T) {
	tests := []struct {
		name        string
		objectTypes []any
		want        bool
	}{
		{"DB only", []any{string(gqlsla.ObjectAzureSQLDatabase)}, true},
		{"MI only", []any{string(gqlsla.ObjectAzureSQLManagedInstance)}, true},
		{"DB + MI", []any{string(gqlsla.ObjectAzureSQLDatabase), string(gqlsla.ObjectAzureSQLManagedInstance)}, true},
		{"DB + Blob", []any{string(gqlsla.ObjectAzureSQLDatabase), string(gqlsla.ObjectAzureBlob)}, false},
		{"MI + other", []any{string(gqlsla.ObjectAzureSQLManagedInstance), "AWS_EC2_EBS_OBJECT_TYPE"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyAzureSQLObjectTypes(tt.objectTypes); got != tt.want {
				t.Fatalf("onlyAzureSQLObjectTypes(%v) = %v, want %v", tt.objectTypes, got, tt.want)
			}
		})
	}
}

// TestScheduleEmpty verifies scheduleEmpty reports true only when no Rubrik
// snapshot schedule of any frequency is set.
func TestScheduleEmpty(t *testing.T) {
	if !scheduleEmpty(gqlsla.SnapshotSchedule{}) {
		t.Fatal("expected empty schedule to report empty")
	}
	schedules := []gqlsla.SnapshotSchedule{
		{Daily: &gqlsla.DailySnapshotSchedule{}},
		{Hourly: &gqlsla.HourlySnapshotSchedule{}},
		{Minute: &gqlsla.MinuteSnapshotSchedule{}},
		{Monthly: &gqlsla.MonthlySnapshotSchedule{}},
		{Quarterly: &gqlsla.QuarterlySnapshotSchedule{}},
		{Weekly: &gqlsla.WeeklySnapshotSchedule{}},
		{Yearly: &gqlsla.YearlySnapshotSchedule{}},
	}
	for _, s := range schedules {
		if scheduleEmpty(s) {
			t.Fatalf("expected non-empty schedule to report non-empty: %+v", s)
		}
	}
}
