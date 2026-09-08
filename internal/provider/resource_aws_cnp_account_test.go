package provider

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

func TestSplitAccountID(t *testing.T) {
	tt := []struct {
		name       string
		id         string
		accountID  uuid.UUID
		externalID string
		errPrefix  string
	}{{
		name:      "InvalidAccountID",
		id:        "a7b9eafe-e0b8-496d-814f",
		errPrefix: "invalid resource id",
	}, {
		name:      "AccountID",
		id:        "a7b9eafe-e0b8-496d-814f-f81a97af853e",
		accountID: uuid.MustParse("a7b9eafe-e0b8-496d-814f-f81a97af853e"),
	}, {
		name:       "AccountIDWithExternalID",
		id:         "a7b9eafe-e0b8-496d-814f-f81a97af853e-external-id",
		accountID:  uuid.MustParse("a7b9eafe-e0b8-496d-814f-f81a97af853e"),
		externalID: "external-id",
	}}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			accountID, externalID, err := splitAccountID(tc.id)
			if err == nil {
				if accountID != tc.accountID {
					t.Errorf("invalid account id: %s", accountID)
				}
				if externalID != tc.externalID {
					t.Errorf("invalid external id: %s", externalID)
				}
			} else {
				if tc.errPrefix == "" || !strings.HasPrefix(err.Error(), tc.errPrefix) {
					t.Errorf("expected error prefix: %q, got: %s", tc.errPrefix, err)
				}
			}
		})
	}
}

func TestFeaturesToRemove(t *testing.T) {
	cd := core.FeatureCloudDiscovery.WithPermissionGroups(core.PermissionGroupBasic)
	cnp := core.FeatureCloudNativeProtection.WithPermissionGroups(core.PermissionGroupBasic)
	ccr := core.FeatureCloudCostReport

	// accountWith builds a cloud account holding the specified features, as
	// returned by RSC.
	accountWith := func(features ...core.Feature) aws.CloudAccount {
		accountFeatures := make([]aws.Feature, 0, len(features))
		for _, feature := range features {
			accountFeatures = append(accountFeatures, aws.Feature{Feature: feature})
		}

		return aws.CloudAccount{Features: accountFeatures}
	}

	tt := []struct {
		name     string
		declared []core.Feature
		account  aws.CloudAccount
		want     []core.Feature
	}{{
		// Nothing declared and nothing on the account, e.g. the account was
		// already removed out of band.
		name: "NoFeatures",
	}, {
		name:     "NoCostReportOnAccount",
		declared: []core.Feature{cd, cnp},
		account:  accountWith(cd, cnp),
		want:     []core.Feature{cd, cnp},
	}, {
		// Cost reporting is appended, and the declared features keep their
		// permission groups.
		name:     "CostReportOnAccount",
		declared: []core.Feature{cd, cnp},
		account:  accountWith(cd, cnp, ccr),
		want:     []core.Feature{cd, cnp, ccr},
	}, {
		// Only cost reporting is picked up. CLOUDACCOUNTS is a deletion child
		// of CLOUD_NATIVE_PROTECTION and is left to RSC. RSC also keeps it out
		// of the account it returns, so this is defensive.
		name:     "OtherUndeclaredFeatureLeftToRSC",
		declared: []core.Feature{cnp},
		account:  accountWith(cnp, ccr, core.FeatureCloudAccounts),
		want:     []core.Feature{cnp, ccr},
	}, {
		// The declared features can be gone from state while RSC still has
		// cost reporting, in which case it is the only feature to remove.
		name:    "OnlyCostReportOnAccount",
		account: accountWith(ccr),
		want:    []core.Feature{ccr},
	}, {
		// A declared feature which RSC no longer has, e.g. removed out of
		// band, is dropped. RSC rejects the removal of a feature the account
		// does not have.
		name:     "DeclaredFeatureRemovedOutOfBand",
		declared: []core.Feature{cd, cnp},
		account:  accountWith(cd, ccr),
		want:     []core.Feature{cd, ccr},
	}, {
		// Nothing is left to remove when RSC has none of the declared
		// features and no cost reporting. Delete must not pass this on: RSC
		// removes every feature on the account when given an empty list.
		name:     "AllDeclaredFeaturesRemovedOutOfBand",
		declared: []core.Feature{cd, cnp},
		account:  accountWith(core.FeatureDSPMData),
	}}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			declared := slices.Clone(tc.declared)

			features := featuresToRemove(declared, tc.account)

			// Compared in order to pin where cost reporting is appended. The
			// order does not reach RSC as is, the SDK moves CLOUD_DISCOVERY
			// last before sending it.
			if !slices.EqualFunc(features, tc.want, func(a, b core.Feature) bool { return a.DeepEqual(b) }) {
				t.Errorf("features: got %v, want %v", features, tc.want)
			}
			if !slices.EqualFunc(declared, tc.declared, func(a, b core.Feature) bool { return a.DeepEqual(b) }) {
				t.Errorf("declared features were modified: got %v, want %v", declared, tc.declared)
			}
		})
	}
}
