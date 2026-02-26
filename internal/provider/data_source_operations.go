// Copyright 2023 Rubrik, Inc.
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
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
)

const dataSourceOperationsDescription = `
The ´polaris_operations´ data source is used to access information about the
valid operations that can be performed by the RSC account.
`

func dataSourceOperations() *schema.Resource {
	return &schema.Resource{
		ReadContext: operationsRead,

		Description: description(dataSourceOperationsDescription),
		Schema: map[string]*schema.Schema{
			keyID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 hash of the operations.",
			},
			keyOperations: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Valid operations that can be performed by the RSC account.",
			},
		},
	}
}

func operationsRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	tflog.Trace(ctx, "operationsRead")

	client, err := m.(*client).polaris()
	if err != nil {
		return diag.FromErr(err)
	}

	coreClient := core.Wrap(client.GQL)

	operations, err := coreClient.ValuesByEnum(ctx, "Operation")
	if err != nil {
		return diag.FromErr(err)
	}

	slices.Sort(operations)

	var operationsAttr []string
	for _, operation := range operations {
		operationsAttr = append(operationsAttr, string(operation))
	}
	if err := d.Set(keyOperations, operationsAttr); err != nil {
		return diag.FromErr(err)
	}

	hash := sha256.New()
	for _, operation := range operations {
		hash.Write([]byte(operation))
	}
	d.SetId(fmt.Sprintf("%x", hash.Sum(nil)))

	return nil
}
