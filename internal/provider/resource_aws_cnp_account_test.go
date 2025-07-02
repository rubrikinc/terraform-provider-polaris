package provider

import (
	"strings"
	"testing"

	"github.com/google/uuid"
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
