package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"net/mail"
	"os"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// validateEmailAddress verifies that i contains a valid email address.
func validateEmailAddress(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	if _, err := mail.ParseAddress(v); err != nil {
		return nil, []error{fmt.Errorf("%q is not a valid email address", v)}
	}

	return nil, nil
}

// validateAwsAccountID verifies the account number is numeric, 12 digits.
func validateAwsAccountID(i interface{}, k string) ([]string, []error) {
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

// validatePermissions verifies that the permissions value is valid.
func validatePermissions(m interface{}, p cty.Path) diag.Diagnostics {
	if m.(string) != "update" {
		return diag.Errorf("invalid permissions value")
	}

	return nil
}

// validateRoleARN verifies that the role ARN is a valid AWS ARN.
func validateRoleARN(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := arn.Parse(m.(string)); err != nil {
		return diag.Errorf("failed to parse role ARN: %v", err)
	}

	return nil
}

// fileExists assumes m is a file path and returns nil if the file exists,
// otherwise a diagnostic message is returned.
func fileExists(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := os.Stat(m.(string)); err != nil {
		details := "unknown error"

		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			details = pathErr.Err.Error()
		}

		return diag.Errorf("failed to access file: %s", details)
	}

	return nil
}

func isExistingFile(i interface{}, k string) ([]string, []error) {
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

// validateHash verifies that m contains a valid base 16 encoded SHA-256 hash
// with two characters per byte.
func validateHash(m interface{}, p cty.Path) diag.Diagnostics {
	if hash, ok := m.(string); ok && len(hash) == 64 {
		return nil
	}

	return diag.Errorf("invalid hash value")
}

// validate numNodes verifies that the num_nodes value is valid. 2 is not allowed
func validateNumNodes(i interface{}, k string) ([]string, []error) {
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

var startAtRegexp = regexp.MustCompile(`^(Mon|Tue|Wed|Thu|Fri|Sat|Sun), (\d{2}):(\d{2})$`)

func validateStartAt(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	matches := startAtRegexp.FindAllStringSubmatch(v, -1)
	if len(matches) != 1 {
		return nil, []error{fmt.Errorf("invalid time format, expected \"DAY, HH:MM\"")}
	}

	match := matches[0]
	if len(match) != 4 {
		return nil, []error{fmt.Errorf("invalid time format, expected \"DAY, HH:MM\"")}
	}

	if n, err := strconv.Atoi(match[2]); err != nil || n < 0 || n > 23 {
		return nil, []error{fmt.Errorf("invalid time format, expected \"DAY, HH:MM\"")}
	}

	if n, err := strconv.Atoi(match[3]); err != nil || n < 0 || n > 59 {
		return nil, []error{fmt.Errorf("invalid time format, expected \"DAY, HH:MM\"")}
	}

	return nil, nil
}
