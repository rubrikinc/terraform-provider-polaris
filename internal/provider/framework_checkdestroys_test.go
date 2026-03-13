package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

// customRoleCheckDestroy verifies that all polaris_custom_role resources have
// been deleted.
func customRoleCheckDestroy(ctx context.Context) func(*terraform.State) error {
	return func(s *terraform.State) error {
		client, err := testClient(ctx)
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "polaris_custom_role" {
				continue
			}

			id, err := uuid.Parse(rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = access.Wrap(client).RoleByID(ctx, id)
			if err == nil {
				return fmt.Errorf("custom role %s still exists", id)
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				return err
			}
		}

		return nil
	}
}

// roleAssignmentCheckDestroy verifies that the specific roles managed by each
// polaris_role_assignment resource have been unassigned. Roles outside the
// resource's management are ignored. Users or SSO groups not found are ignored.
func roleAssignmentCheckDestroy(ctx context.Context) func(*terraform.State) error {
	return func(s *terraform.State) error {
		client, err := testClient(ctx)
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "polaris_role_assignment" {
				continue
			}

			// Collect the managed role IDs from the state.
			managedRoleIDs := make(map[uuid.UUID]struct{})
			if v, ok := rs.Primary.Attributes[keyRoleID]; ok && v != "" {
				id, err := uuid.Parse(v)
				if err != nil {
					return err
				}
				managedRoleIDs[id] = struct{}{}
			}
			if countStr, ok := rs.Primary.Attributes[keyRoleIDs+".#"]; ok {
				count, err := strconv.Atoi(countStr)
				if err != nil {
					return err
				}
				for i := 0; i < count; i++ {
					v := rs.Primary.Attributes[fmt.Sprintf("%s.%d", keyRoleIDs, i)]
					id, err := uuid.Parse(v)
					if err != nil {
						return err
					}
					managedRoleIDs[id] = struct{}{}
				}
			}

			// Try as user.
			user, err := access.Wrap(client).UserByID(ctx, rs.Primary.ID)
			if err == nil {
				for _, role := range user.Roles {
					if _, ok := managedRoleIDs[role.ID]; ok {
						return fmt.Errorf("role %q still assigned to user %q", role.ID, rs.Primary.ID)
					}
				}
				continue
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				return err
			}

			// Try as SSO group.
			group, err := access.Wrap(client).SSOGroupByID(ctx, rs.Primary.ID)
			if err == nil {
				for _, role := range group.Roles {
					if _, ok := managedRoleIDs[role.ID]; ok {
						return fmt.Errorf("role %q still assigned to SSO group %q", role.ID, rs.Primary.ID)
					}
				}
				continue
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				return err
			}
		}

		return nil
	}
}

// userCheckDestroy verifies that all polaris_user resources have been deleted.
func userCheckDestroy(ctx context.Context) func(*terraform.State) error {
	return func(s *terraform.State) error {
		client, err := testClient(ctx)
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "polaris_user" {
				continue
			}

			_, err := access.Wrap(client).UserByID(ctx, rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("user %s still exists", rs.Primary.ID)
			}
			if !errors.Is(err, graphql.ErrNotFound) {
				return err
			}
		}

		return nil
	}
}
