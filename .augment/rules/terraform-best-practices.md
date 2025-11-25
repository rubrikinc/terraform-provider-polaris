# Terraform Provider Development Best Practices

This document outlines best practices for developing the Terraform Provider for Rubrik Polaris, following HashiCorp's recommended patterns and conventions.

## Resource and Data Source Structure

### Standard File Organization

Each resource and data source should follow this structure:

1. **Copyright header** - MIT license header at the top of every file
2. **Package declaration** - `package provider`
3. **Imports** - Organized with standard library, then third-party, then internal
4. **Description constant** - Multi-line description with examples and notes
5. **Resource/Data Source function** - Returns `*schema.Resource`
6. **CRUD functions** - Create, Read, Update, Delete operations
7. **Helper functions** - Any supporting functions

### Resource Naming Conventions

- **Resource files**: `resource_<provider>_<resource_name>.go` (e.g., `resource_aws_exocompute.go`)
- **Data source files**: `data_source_<resource_name>.go` (e.g., `data_source_user.go`)
- **Resource keys**: Use constants from `names.go` (e.g., `keyAccountID`, `keyRegion`)
- **Resource names**: Use `polaris_<provider>_<resource>` format (e.g., `polaris_aws_account`)

### Schema Definition Best Practices

```go
func resourceExample() *schema.Resource {
    return &schema.Resource{
        CreateContext: exampleCreate,
        ReadContext:   exampleRead,
        UpdateContext: exampleUpdate,
        DeleteContext: exampleDelete,

        Description: description(resourceExampleDescription),
        Schema: map[string]*schema.Schema{
            keyID: {
                Type:        schema.TypeString,
                Computed:    true,
                Description: "Resource ID (UUID).",
            },
            keyName: {
                Type:         schema.TypeString,
                Required:     true,
                Description:  "Resource name.",
                ValidateFunc: validation.StringIsNotWhiteSpace,
            },
            // More fields...
        },
    }
}
```

**Key Points**:
- Always use `description()` helper function for descriptions
- Use constants from `names.go` for all schema keys
- Include clear, concise descriptions for all fields
- Use appropriate validators from `validation` package or custom validators
- Mark computed fields appropriately
- Use `ForceNew: true` for fields that require resource recreation

## CRUD Operation Patterns

### Create Operations

```go
func resourceCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceCreate")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    // Extract parameters from schema
    name := d.Get(keyName).(string)
    
    // Call SDK function
    id, err := someSDKFunction(ctx, name)
    if err != nil {
        return diag.FromErr(err)
    }

    // Set the resource ID
    d.SetId(id.String())

    return resourceRead(ctx, d, m)
}
```

### Read Operations

```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceRead")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    // Parse resource ID
    id, err := uuid.Parse(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    // Fetch resource from API
    resource, err := someSDKFunction(ctx, id)
    if err != nil {
        return diag.FromErr(err)
    }

    // Set all schema fields
    if err := d.Set(keyName, resource.Name); err != nil {
        return diag.FromErr(err)
    }

    return nil
}
```

### Update Operations

```go
func resourceUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceUpdate")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    // Check what changed
    if d.HasChange(keyName) {
        // Handle the change
    }

    return resourceRead(ctx, d, m)
}
```

### Delete Operations

```go
func resourceDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceDelete")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    // Parse resource ID
    id, err := uuid.Parse(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    // Delete the resource
    if err := someSDKFunction(ctx, id); err != nil {
        return diag.FromErr(err)
    }

    return nil
}
```

## Common Patterns

### Logging

- Use `tflog.Trace(ctx, "functionName")` at the start of all CRUD functions
- Use structured logging with `tflog` for debugging information
- The provider supports two log levels via environment variables:
  - `TF_LOG_PROVIDER_POLARIS` - Controls provider-level logging
  - `TF_LOG_PROVIDER_POLARIS_API` - Controls API-level logging

### Error Handling

- Always use `diag.FromErr(err)` to convert errors to diagnostics
- Provide context in error messages when possible
- Use `diag.Errorf()` for formatted error messages

### Client Access

```go
client, err := m.(*client).polaris()
if err != nil {
    return diag.FromErr(err)
}
```

### UUID Handling

```go
import "github.com/google/uuid"

// Parse UUID from string
id, err := uuid.Parse(d.Id())
if err != nil {
    return diag.FromErr(err)
}

// Set UUID as string
d.SetId(id.String())
```

### Region Handling

Always use region types from the SDK instead of strings:

```go
import gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"

// Parse region from user input
region := gqlaws.RegionFromName(d.Get(keyRegion).(string))
if region == gqlaws.RegionUnknown {
    return diag.Errorf("invalid region: %s", d.Get(keyRegion).(string))
}

// Convert to GraphQL enum
regionEnum := region.ToRegionEnum()
```

### SDK Wrapper Functions

Use SDK wrapper functions for cleaner code:

```go
import (
    "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
    gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)

// Use wrapper
user, err := access.Wrap(client).UserByID(ctx, userID)
```

## State Management

### Schema Versioning

When making breaking changes to a resource schema:

1. Increment `SchemaVersion` field
2. Add a `StateUpgrader` to migrate old state to new schema
3. Keep old schema definitions (e.g., `resource_aws_account_v0.go`)

Example:
```go
SchemaVersion: 2,
StateUpgraders: []schema.StateUpgrader{{
    Type:    resourceAwsAccountV0().CoreConfigSchema().ImpliedType(),
    Upgrade: resourceAwsAccountStateUpgradeV0,
    Version: 0,
}, {
    Type:    resourceAwsAccountV1().CoreConfigSchema().ImpliedType(),
    Upgrade: resourceAwsAccountStateUpgradeV1,
    Version: 1,
}},
```

### Setting Complex Types

For sets:
```go
items := &schema.Set{F: schema.HashString}
for _, item := range itemList {
    items.Add(item)
}
if err := d.Set(keyItems, items); err != nil {
    return diag.FromErr(err)
}
```

For maps:
```go
if err := d.Set(keyConfig, map[string]interface{}{
    keyName:   value.Name,
    keyStatus: value.Status,
}); err != nil {
    return diag.FromErr(err)
}
```

## Documentation

### Description Constants

All resources and data sources must have a description constant:

```go
const resourceExampleDescription = `
The ´polaris_example´ resource manages an example resource in RSC.

-> **Note:** Important information about the resource.

~> **Warning:** Warning about potential issues.
`
```

**Formatting**:
- Use backticks for code/resource names (´polaris_example´)
- Use `->` for notes
- Use `~>` for warnings
- Include usage examples when helpful

### Field Descriptions

- Be concise but clear
- Specify if the field is a UUID
- Mention if changing the field forces a new resource
- Include valid values or ranges when applicable

Example:
```go
keyAccountID: {
    Type:         schema.TypeString,
    Required:     true,
    ForceNew:     true,
    Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
    ValidateFunc: validation.IsUUID,
},
```

## Validation

### Built-in Validators

Use validators from `github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation`:

- `validation.StringIsNotWhiteSpace` - Ensures string is not empty or whitespace
- `validation.IsUUID` - Validates UUID format
- `validation.StringInSlice()` - Validates string is in allowed list

### Custom Validators

Define custom validators in `validators.go`:

```go
func validateExample(i any, k string) ([]string, []error) {
    v, ok := i.(string)
    if !ok {
        return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
    }

    // Validation logic
    if !isValid(v) {
        return nil, []error{fmt.Errorf("%q is not valid", v)}
    }

    return nil, nil
}
```

For diagnostic-based validators:
```go
func validateExampleDiag(m any, p cty.Path) diag.Diagnostics {
    if !isValid(m.(string)) {
        return diag.Errorf("invalid value")
    }
    return nil
}
```

## Testing

### Acceptance Tests

- Test files should be named `resource_<name>_test.go`
- Use `resource.Test()` for acceptance tests
- Test create, read, update, and delete operations
- Test validation and error cases

### Test Patterns

```go
func TestAccResourceExample_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProviderFactories: providerFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccResourceExampleConfig(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("polaris_example.test", "name", "test"),
                ),
            },
        },
    })
}
```

