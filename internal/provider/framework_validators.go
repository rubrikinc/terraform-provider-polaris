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

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// isUUID returns a validator that checks if a string value is a valid UUID.
func isUUID() validator.String {
	return isUUIDValidator{}
}

type isUUIDValidator struct{}

func (v isUUIDValidator) Description(_ context.Context) string {
	return "value must be a valid UUID"
}

func (v isUUIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v isUUIDValidator) ValidateString(_ context.Context, req validator.StringRequest, res *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if _, err := uuid.Parse(req.ConfigValue.ValueString()); err != nil {
		res.Diagnostics.AddAttributeError(req.Path, "Invalid UUID",
			fmt.Sprintf("%q is not a valid UUID: %s", req.ConfigValue.ValueString(), err))
	}
}

// isNotWhiteSpace returns a validator that checks if a string value is not
// empty or contains only whitespace.
func isNotWhiteSpace() validator.String {
	return isNotWhiteSpaceValidator{}
}

type isNotWhiteSpaceValidator struct{}

func (v isNotWhiteSpaceValidator) Description(_ context.Context) string {
	return "value must not be empty or contain only whitespace"
}

func (v isNotWhiteSpaceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v isNotWhiteSpaceValidator) ValidateString(_ context.Context, req validator.StringRequest, res *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		res.Diagnostics.AddAttributeError(req.Path, "Invalid Value",
			"value must not be empty or contain only whitespace")
	}
}

// setMustContain returns a validator that checks a set of strings contains the
// given value. A null or unknown set passes (nothing to validate yet).
func setMustContain(value string) validator.Set {
	return setMustContainValidator{value: value}
}

type setMustContainValidator struct {
	value string
}

func (v setMustContainValidator) Description(_ context.Context) string {
	return fmt.Sprintf("set must contain %q", v.value)
}

func (v setMustContainValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v setMustContainValidator) ValidateSet(ctx context.Context, req validator.SetRequest, res *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var values []string
	res.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &values, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	for _, value := range values {
		if value == v.value {
			return
		}
	}

	res.Diagnostics.AddAttributeError(req.Path, "Missing required value",
		fmt.Sprintf("%q must be included in the set", v.value))
}
