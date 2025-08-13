package provider

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestJoinTrustPolicyID(t *testing.T) {
	tt := []struct {
		name      string
		id        string
		roleKey   string
		accountID uuid.UUID
		errPrefix string
	}{{
		name:      "InvalidRoleKey",
		roleKey:   "INVALID",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		errPrefix: "invalid role key",
	}, {
		name:      "CrossAccount",
		id:        "CROSSACCOUNT-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		roleKey:   "CROSSACCOUNT",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
	}, {
		name:      "ExocomputeEKSMasterNode",
		id:        "EXOCOMPUTE_EKS_MASTERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		roleKey:   "EXOCOMPUTE_EKS_MASTERNODE",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
	}, {
		name:      "ExocomputeEKSWorkerNode",
		id:        "EXOCOMPUTE_EKS_WORKERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		roleKey:   "EXOCOMPUTE_EKS_WORKERNODE",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
	}}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			id, err := joinTrustPolicyID(tc.roleKey, tc.accountID)
			if err == nil {
				if id != tc.id {
					t.Errorf("invalid id: %s", id)
				}
			} else {
				if tc.errPrefix == "" || !strings.HasPrefix(err.Error(), tc.errPrefix) {
					t.Errorf("expected error prefix: %q, got: %s", tc.errPrefix, err)
				}
			}
		})
	}
}

func TestSplitTrustPolicyID(t *testing.T) {
	tt := []struct {
		name       string
		id         string
		roleKey    string
		accountID  uuid.UUID
		externalID string
		errPrefix  string
	}{{
		name:      "InvalidRoleKey",
		id:        "invalid-role-key-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		errPrefix: "invalid resource id",
	}, {
		name:      "InvalidAccountID",
		id:        "CROSSACCOUNT-c1bf026b-bb95-4d00-baba",
		errPrefix: "invalid resource id",
	}, {
		name:      "InvalidAccountIDWithExternalID",
		id:        "CROSSACCOUNT-c1bf026b-bb95-4d00-baba-external-id",
		errPrefix: "invalid resource id",
	}, {
		name:      "CrossAccount",
		id:        "CROSSACCOUNT-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		roleKey:   "CROSSACCOUNT",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
	}, {
		name:       "CrossAccountWithExternalID",
		id:         "CROSSACCOUNT-c1bf026b-bb95-4d00-baba-03a188abe9b8-external-id",
		roleKey:    "CROSSACCOUNT",
		accountID:  uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		externalID: "external-id",
	}, {
		name:      "ExocomputeEKSMasterNode",
		id:        "EXOCOMPUTE_EKS_MASTERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		roleKey:   "EXOCOMPUTE_EKS_MASTERNODE",
	}, {
		name:       "ExocomputeEKSMasterNodeWithExternalID",
		id:         "EXOCOMPUTE_EKS_MASTERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8-external-id",
		accountID:  uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		roleKey:    "EXOCOMPUTE_EKS_MASTERNODE",
		externalID: "external-id",
	}, {
		name:      "ExocomputeEKSWorkerNode",
		id:        "EXOCOMPUTE_EKS_MASTERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8",
		accountID: uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		roleKey:   "EXOCOMPUTE_EKS_MASTERNODE",
	}, {
		name:       "ExocomputeEKSWorkerNodeWithExternalID",
		id:         "EXOCOMPUTE_EKS_MASTERNODE-c1bf026b-bb95-4d00-baba-03a188abe9b8-external-id",
		accountID:  uuid.MustParse("c1bf026b-bb95-4d00-baba-03a188abe9b8"),
		roleKey:    "EXOCOMPUTE_EKS_MASTERNODE",
		externalID: "external-id",
	}}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			roleKey, accountID, externalID, err := splitTrustPolicyID(tc.id)
			if err == nil {
				if roleKey != tc.roleKey {
					t.Errorf("invalid role key: %s", roleKey)
				}
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
