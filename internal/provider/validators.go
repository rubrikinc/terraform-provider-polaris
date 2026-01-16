package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"net/mail"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// validateAwsAccountID verifies the account number is numeric, 12 digits.
func validateAwsAccountID(i any, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if len(v) != 12 {
		return nil, []error{fmt.Errorf("%q is not a valid account number", v)}
	}
	if _, err := strconv.ParseUint(v, 10, 64); err != nil {
		return nil, []error{fmt.Errorf("%q is not a valid account number", v)}
	}

	return nil, nil
}

// validateEmailAddress verifies that i contains a valid email address.
func validateEmailAddress(i any, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if _, err := mail.ParseAddress(v); err != nil {
		return nil, []error{fmt.Errorf("%q is not a valid email address", v)}
	}

	return nil, nil
}

// validateFileExist assumes m is a file path and returns nil if the file exist,
// otherwise a diagnostic message is returned.
func validateFileExist(i any, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	if _, err := os.Stat(v); err != nil {
		details := "unknown error"
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			details = pathErr.Err.Error()
		}

		return nil, []error{fmt.Errorf("failed to access file: %s", details)}
	}

	return nil, nil
}

// validate numNodes verifies that the num_nodes value is valid. 2 is not allowed
func validateNumNodes(i any, k string) ([]string, []error) {
	v, ok := i.(int)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if v == 2 {
		return nil, []error{fmt.Errorf("num_nodes cannot be 2")}
	}
	if v < 1 {
		return nil, []error{fmt.Errorf("num_nodes must be greater than 0")}
	}
	return nil, nil
}

// validatePermissions verifies that the permissions value is valid.
func validatePermissions(m any, p cty.Path) diag.Diagnostics {
	if m.(string) != "update" {
		return diag.Errorf("invalid permissions value")
	}

	return nil
}

// validateRoleARN verifies that the role ARN is a valid AWS ARN.
func validateRoleARN(m any, p cty.Path) diag.Diagnostics {
	if _, err := arn.Parse(m.(string)); err != nil {
		return diag.Errorf("failed to parse role ARN: %v", err)
	}

	return nil
}

// validateStringIsNumber assumes m is a string holding an integer and returns
// nil if the string can be converted to an integer, otherwise a diagnostic
// message is returned.
func validateStringIsNumber(i any, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if _, err := strconv.ParseInt(v, 10, 64); err != nil {
		return nil, []error{fmt.Errorf("%q is not an integer: %s", v, err)}
	}

	return nil, nil
}

// validateStartAt returns a function that validates the start_at value.
// The GQL type allows specifying a day of week optionally, but different
// endpoints use the value in different ways. E.g. "First full snapshot"
// requires the day of week, but "Snapshot window" requires it to be omitted.
func validateStartAt(withDay bool) func(i interface{}, k string) ([]string, []error) {
	return func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
		}
		parts := strings.Split(v, ", ")
		var timeParts []string
		switch len(parts) {
		case 1:
			// Day of week not specified.
			if withDay {
				return nil, []error{fmt.Errorf("day of week required for %s: %s", k, v)}
			}
			timeParts = strings.Split(parts[0], ":")
		case 2:
			// Day of week specified.
			if !withDay {
				return nil, []error{fmt.Errorf("day of week not allowed for %s: %s", k, v)}
			}
			if !slices.Contains([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}, parts[0]) {
				return nil, []error{fmt.Errorf("invalid day of week for %s: %s", k, v)}
			}
			timeParts = strings.Split(parts[1], ":")
		default:
			return nil, []error{fmt.Errorf("invalid format for %s: %s", k, v)}
		}
		if len(timeParts) != 2 {
			return nil, []error{fmt.Errorf("invalid time format for %s: %s", k, v)}
		}

		if n, err := strconv.Atoi(timeParts[0]); err != nil || n < 0 || n > 23 {
			return nil, []error{fmt.Errorf("invalid hour for %s: %s", k, v)}
		}

		if n, err := strconv.Atoi(timeParts[1]); err != nil || n < 0 || n > 59 {
			return nil, []error{fmt.Errorf("invalid minute for %s: %s", k, v)}
		}

		return nil, nil
	}
}
