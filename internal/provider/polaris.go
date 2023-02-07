// Copyright 2021 Rubrik, Inc.
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
	"errors"
	"io/fs"
	"os"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

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
