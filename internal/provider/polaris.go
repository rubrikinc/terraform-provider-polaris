package provider

import (
	"errors"
	"io/fs"
	"os"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

// fileExists assumes m is a file path and returns nil if the file exists,
// otherwise an diagnostic message is returned.
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

// validateFeature verifies that m contains a valid Polaris feature name.
func validateFeature(m interface{}, p cty.Path) diag.Diagnostics {
	_, err := core.ParseFeature(m.(string))
	return diag.FromErr(err)
}

// validateHash verifies that m contains a valid SHA-256 hash.
func validateHash(m interface{}, p cty.Path) diag.Diagnostics {
	if hash, ok := m.(string); ok && len(hash) == 64 {
		return nil
	}

	return diag.Errorf("invalid hash value")
}
