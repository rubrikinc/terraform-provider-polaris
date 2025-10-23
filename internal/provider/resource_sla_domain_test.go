package provider

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/sla"
)

func TestToArchival(t *testing.T) {
	locationID1 := uuid.New()
	locationID2 := uuid.New()
	locationID3 := uuid.New()

	tests := []struct {
		name          string
		archivalSpecs []sla.ArchivalSpec
		existing      []any
		expected      []any
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty specs and existing",
			archivalSpecs: []sla.ArchivalSpec{},
			existing:      []any{},
			expected:      []any{},
		}, {
			name: "new specs only",
			archivalSpecs: []sla.ArchivalSpec{
				{GroupID: locationID1, Threshold: 30, ThresholdUnit: sla.Days},
				{GroupID: locationID2, Threshold: 90, ThresholdUnit: sla.Weeks},
			},
			existing: []any{},
			expected: []any{
				map[string]any{keyArchivalLocationID: locationID1.String(), keyThreshold: 30, keyThresholdUnit: "DAYS"},
				map[string]any{keyArchivalLocationID: locationID2.String(), keyThreshold: 90, keyThresholdUnit: "WEEKS"},
			},
		}, {
			name: "preserve existing order",
			archivalSpecs: []sla.ArchivalSpec{
				{GroupID: locationID1, Threshold: 30, ThresholdUnit: sla.Days},
				{GroupID: locationID2, Threshold: 90, ThresholdUnit: sla.Weeks},
			},
			existing: []any{
				map[string]any{keyArchivalLocationID: locationID2.String()},
				map[string]any{keyArchivalLocationID: locationID1.String()},
			},
			expected: []any{
				map[string]any{keyArchivalLocationID: locationID2.String(), keyThreshold: 90, keyThresholdUnit: "WEEKS"},
				map[string]any{keyArchivalLocationID: locationID1.String(), keyThreshold: 30, keyThresholdUnit: "DAYS"},
			},
		}, {
			name: "add new specs to end",
			archivalSpecs: []sla.ArchivalSpec{
				{GroupID: locationID1, Threshold: 30, ThresholdUnit: sla.Days},
				{GroupID: locationID2, Threshold: 90, ThresholdUnit: sla.Weeks},
				{GroupID: locationID3, Threshold: 12, ThresholdUnit: sla.Months},
			},
			existing: []any{
				map[string]any{keyArchivalLocationID: locationID2.String()},
			},
			expected: []any{
				map[string]any{keyArchivalLocationID: locationID2.String(), keyThreshold: 90, keyThresholdUnit: "WEEKS"},
				map[string]any{keyArchivalLocationID: locationID1.String(), keyThreshold: 30, keyThresholdUnit: "DAYS"},
				map[string]any{keyArchivalLocationID: locationID3.String(), keyThreshold: 12, keyThresholdUnit: "MONTHS"},
			},
		}, {
			name: "remove existing specs",
			archivalSpecs: []sla.ArchivalSpec{
				{GroupID: locationID1, Threshold: 30, ThresholdUnit: sla.Days},
				{GroupID: locationID3, Threshold: 12, ThresholdUnit: sla.Months},
			},
			existing: []any{
				map[string]any{keyArchivalLocationID: locationID3.String()},
				map[string]any{keyArchivalLocationID: locationID2.String()},
				map[string]any{keyArchivalLocationID: locationID1.String()},
			},
			expected: []any{
				map[string]any{keyArchivalLocationID: locationID3.String(), keyThreshold: 12, keyThresholdUnit: "MONTHS"},
				map[string]any{keyArchivalLocationID: locationID1.String(), keyThreshold: 30, keyThresholdUnit: "DAYS"},
			},
		}, {
			name: "duplicate location IDs",
			archivalSpecs: []sla.ArchivalSpec{
				{GroupID: locationID1, Threshold: 30, ThresholdUnit: sla.Days},
				{GroupID: locationID1, Threshold: 90, ThresholdUnit: sla.Weeks},
			},
			existing:      []any{},
			expectError:   true,
			errorContains: "used multiple times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toArchival(tt.archivalSpecs, tt.existing)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				actual := result[i].(map[string]any)
				expectedMap := expected.(map[string]any)

				for key, expectedValue := range expectedMap {
					if actual[key] != expectedValue {
						t.Fatalf("item %d: expected %s=%v, got %v", i, key, expectedValue, actual[key])
					}
				}
			}
		})
	}
}
