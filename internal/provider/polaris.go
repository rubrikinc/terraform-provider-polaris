package provider

import (
	"errors"
	"io/fs"
	"os"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// credentialsFileExists assumes m is a file path and returns nil if the file
// exists, otherwise an diagnostic message is returned.
func credentialsFileExists(m interface{}, p cty.Path) diag.Diagnostics {
	if _, err := os.Stat(m.(string)); err != nil {
		details := "unknown error"

		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			details = pathErr.Err.Error()
		}

		return diag.Errorf("failed to access the credentials file: %s", details)
	}

	return nil
}
