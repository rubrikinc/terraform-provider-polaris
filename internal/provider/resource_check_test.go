// Copyright 2025 Rubrik, Inc.
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
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func checkResourceAttrIsUUID(name string, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rm := s.RootModule()

		rs, ok := rm.Resources[name]
		if !ok {
			return fmt.Errorf("resource %q not found in %s", name, rm.Path)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("no primary instance for resource %q in %s", name, rm.Path)
		}

		v, ok := is.Attributes[key]
		if !ok {
			return fmt.Errorf("attribute %q not found in %s", key, name)
		}

		_, err := uuid.Parse(v)
		if err != nil {
			return fmt.Errorf("attribute %q is not a valid UUID: %s", key, err)
		}

		return nil
	}
}
