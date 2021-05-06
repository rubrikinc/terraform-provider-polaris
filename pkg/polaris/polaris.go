package polaris

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

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

// fromResourceID converts a resource id to a Polaris cloud account id and
// a native id. Native id in this context means a CSP specific id.
func fromResourceID(resourceID string) (cloudAccountID string, nativeID string, err error) {
	parts := strings.Split(resourceID, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("polaris: invalid resource id: %s", resourceID)
	}

	return parts[0], parts[1], nil
}

// toResourceID converts a Polaris cloud account id and a native id to a
// resource id. Native id in this context means a CSP specific id.
func toResourceID(cloudAccountID, nativeID string) string {
	return fmt.Sprintf("%s:%s", cloudAccountID, nativeID)
}
