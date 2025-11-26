---
type: "always_apply"
---

# Terraform Provider Patterns

## Resource Schema Definition

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
        },
    }
}
```

**Key Points**:
- Always use `description()` helper function
- Use constants from `names.go` for all schema keys
- Include clear descriptions for all fields
- Use appropriate validators
- Mark computed fields appropriately
- Use `ForceNew: true` for fields requiring resource recreation

## CRUD Operation Patterns

### Create
```go
func resourceCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceCreate")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    name := d.Get(keyName).(string)
    
    id, err := someSDKFunction(ctx, name)
    if err != nil {
        return diag.FromErr(err)
    }

    d.SetId(id.String())
    return resourceRead(ctx, d, m)
}
```

### Read
```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceRead")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    id, err := uuid.Parse(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    resource, err := someSDKFunction(ctx, id)
    if err != nil {
        return diag.FromErr(err)
    }

    if err := d.Set(keyName, resource.Name); err != nil {
        return diag.FromErr(err)
    }

    return nil
}
```

### Update
```go
func resourceUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceUpdate")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    if d.HasChange(keyName) {
        // Handle the change
    }

    return resourceRead(ctx, d, m)
}
```

### Delete
```go
func resourceDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceDelete")

    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    id, err := uuid.Parse(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    if err := someSDKFunction(ctx, id); err != nil {
        return diag.FromErr(err)
    }

    return nil
}
```

## Common Patterns

### Error Handling

**Critical Rules**:
- CRUD functions MUST return `diag.Diagnostics`
- Use `diag.FromErr(err)` to wrap standard errors
- Use `diag.Errorf(format, args...)` for custom error messages
- ALWAYS check errors from `d.Set()` calls
- Handle `graphql.ErrNotFound` specially in Read operations

**Standard Error Patterns**:

```go
// 1. Wrap standard errors
if err != nil {
    return diag.FromErr(err)
}

// 2. Custom error messages
if condition {
    return diag.Errorf("invalid region: %s", regionName)
}

// 3. Check d.Set() errors
if err := d.Set(keyName, resource.Name); err != nil {
    return diag.FromErr(err)
}

// 4. Handle NotFound in Read (remove from state)
resource, err := someSDKFunction(ctx, id)
if errors.Is(err, graphql.ErrNotFound) {
    d.SetId("")
    return nil
}
if err != nil {
    return diag.FromErr(err)
}

// 5. Ignore NotFound in Delete
if err := someSDKFunction(ctx, id); err != nil && !errors.Is(err, graphql.ErrNotFound) {
    return diag.FromErr(err)
}
```

**Import Required**:
```go
import (
    "errors"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql"
)
```

### Logging
- Use `tflog.Trace(ctx, "functionName")` at the start of all CRUD functions
- Provider supports `TF_LOG_PROVIDER_POLARIS` and `TF_LOG_PROVIDER_POLARIS_API` env vars

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
Always use region types from SDK instead of strings:

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
2. Add a `StateUpgrader` to migrate old state
3. Keep old schema definitions (e.g., `resource_aws_account_v0.go`)


