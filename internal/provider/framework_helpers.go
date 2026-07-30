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

// This file holds small, resource-agnostic helpers shared across the Terraform
// Plugin Framework resources and data sources in this package.

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// decodeStringSetOrNil decodes a string types.Set into a []string, returning nil
// when the set is null or unknown.
//
// A bare set.ElementsAs errors on an unknown set, which happens for an
// Optional+Computed attribute the practitioner left unset (it is "known after
// apply" in the plan). Guarding for null/unknown here lets callers decode a set
// straight from the plan or config without repeating that check. Any decode
// errors are appended to diags.
func decodeStringSetOrNil(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(set.ElementsAs(ctx, &out, false)...)
	return out
}

// possibleValues returns a sentence describing the allowed values for use in an
// attribute description. Each value is backtick-quoted and the last two are
// joined by "and". An empty slice returns the empty string.
func possibleValues[T ~string](values []T) string {
	switch l := len(values); l {
	case 0:
		return ""
	case 1:
		return "Possible value is `" + string(values[0]) + "`"
	default:
		var sb strings.Builder
		for i := range values {
			if i == l-1 {
				sb.WriteString(" and `" + string(values[i]) + "`")
			} else {
				sb.WriteString(", `" + string(values[i]) + "`")
			}
		}

		return fmt.Sprintf("Possible values are %s", sb.String()[2:])
	}
}
