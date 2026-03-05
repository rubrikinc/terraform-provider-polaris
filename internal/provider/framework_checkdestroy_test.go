package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)

func customRoleCheckDestroy(s *terraform.State) error {
	client, err := testClient()
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

		_, err = access.Wrap(client).RoleByID(context.Background(), id)
		if err == nil {
			return fmt.Errorf("custom role %s still exists", id)
		}
		if !errors.Is(err, graphql.ErrNotFound) {
			return err
		}
	}

	return nil
}

func roleAssignmentCheckDestroy(s *terraform.State) error {
	client, err := testClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "polaris_role_assignment" {
			continue
		}

		// Try as user.
		user, err := access.Wrap(client).UserByID(context.Background(), rs.Primary.ID)
		if err == nil {
			if len(user.Roles) > 0 {
				return fmt.Errorf("role assignment for user %s still has %d roles",
					rs.Primary.ID, len(user.Roles))
			}
			continue
		}
		if !errors.Is(err, graphql.ErrNotFound) {
			return err
		}

		// Try as SSO group.
		group, err := access.Wrap(client).SSOGroupByID(context.Background(), rs.Primary.ID)
		if err == nil {
			if len(group.Roles) > 0 {
				return fmt.Errorf("role assignment for SSO group %s still has %d roles",
					rs.Primary.ID, len(group.Roles))
			}
			continue
		}
		if !errors.Is(err, graphql.ErrNotFound) {
			return err
		}
	}

	return nil
}
