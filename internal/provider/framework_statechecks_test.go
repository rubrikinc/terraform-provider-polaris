package provider

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
)

var _ knownvalue.Check = nonNullUUID{}

type nonNullUUID struct{}

func NonNullUUID() nonNullUUID {
	return nonNullUUID{}
}

func (n nonNullUUID) CheckValue(value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	uuidRegex := regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	)
	if !uuidRegex.MatchString(str) {
		return fmt.Errorf("expected UUID, got: %s", str)
	}

	if str == "00000000-0000-0000-0000-000000000000" {
		return fmt.Errorf("expected non-null UUID, got null UUID")
	}

	return nil
}

func (n nonNullUUID) String() string {
	return "valid non-null UUID"
}
