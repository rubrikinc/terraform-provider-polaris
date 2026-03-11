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
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

// testUserEmail returns the new user email from the RSC test configuration.
func testUserEmail(t *testing.T) string {
	t.Helper()

	rsc, err := loadRSCTestConf()
	if err != nil {
		t.Fatal(err)
	}

	return rsc.NewUserEmail
}

// createTestRole creates a custom role via the SDK and registers a cleanup
// function to delete it. The role will have the VIEW_CLUSTER permission on the
// CLUSTER_ROOT. Returns the role ID.
func createTestRole(t *testing.T, name string) uuid.UUID {
	t.Helper()

	client, err := testClient(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	desc := "Test Role: Delete Me!"
	roleID, err := access.Wrap(client).CreateRole(t.Context(), name, desc, []gqlaccess.Permission{{
		Operation: "VIEW_CLUSTER",
		ObjectsForHierarchyTypes: []gqlaccess.ObjectsForHierarchyType{{
			SnappableType: "AllSubHierarchyType",
			ObjectIDs:     []string{"CLUSTER_ROOT"},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := access.Wrap(client).DeleteRole(context.Background(), roleID); err != nil {
			t.Logf("failed to delete test role %q: %s", roleID, err)
		}
	})

	return roleID
}

// createTestRoleWithUniqueName creates a custom role with a unqiue name via
// the SDK and registers a cleanup function to delete it. The role will have
// the VIEW_CLUSTER permission on the CLUSTER_ROOT. Returns the role ID.
func createTestRoleWithUniqueName(t *testing.T) uuid.UUID {
	t.Helper()

	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	return createTestRole(t, fmt.Sprintf("Test Role %s", id.String()))
}

// createTestUser creates a user with the specified email and role via the SDK.
// Registers a cleanup function to delete the user. Returns the user ID.
func createTestUser(t *testing.T, email string, roleID uuid.UUID) string {
	t.Helper()

	client, err := testClient(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	userID, err := access.Wrap(client).CreateUser(t.Context(), email, []uuid.UUID{roleID})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := access.Wrap(client).DeleteUser(context.Background(), userID); err != nil {
			t.Logf("failed to delete test user %q: %s", userID, err)
		}
	})

	return userID
}
