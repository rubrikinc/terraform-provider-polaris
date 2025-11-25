# Code Review Guide for Augment AI

This guide provides a systematic approach for Augment AI to review code changes in the Terraform Provider for Rubrik Polaris project.

## Review Checklist

When reviewing code, systematically check the following areas:

### 1. Code Standards Compliance

- [ ] All exported types and functions have documentation comments
- [ ] Acronyms are fully uppercase (AWS, API, ID, UUID, etc.)
- [ ] Constants are defined in `names.go` for schema field keys
- [ ] Imports are organized correctly (stdlib, third-party, internal)
- [ ] Error messages follow conventions (lowercase, specific, contextual)

### 2. Terraform Provider Patterns

- [ ] Resources follow naming convention: `polaris_<provider>_<resource>`
- [ ] Files follow naming convention: `resource_<provider>_<resource>.go`
- [ ] Description constants are defined and use `description()` helper
- [ ] Schema uses constants from `names.go` for all keys
- [ ] All schema fields have clear descriptions
- [ ] Appropriate validators are used (built-in or custom)
- [ ] `ForceNew: true` is set for fields requiring resource recreation
- [ ] CRUD functions use `tflog.Trace()` at entry point
- [ ] CRUD functions return `diag.Diagnostics`
- [ ] Errors are converted using `diag.FromErr()` or `diag.Errorf()`

### 3. SDK Integration

- [ ] Region types are used instead of strings (`aws.Region`, `azure.Region`, `gcp.Region`)
- [ ] Region parsing uses `RegionFromName()` or similar functions
- [ ] Region conversion uses appropriate methods (`.ToRegionEnum()`, etc.)
- [ ] SDK wrapper functions are used (e.g., `access.Wrap(client)`)
- [ ] UUIDs are handled using `github.com/google/uuid`
- [ ] GraphQL types are imported from SDK packages

### 4. State Management

- [ ] Schema versioning is incremented for breaking changes
- [ ] State upgraders are provided for schema migrations
- [ ] Old schema versions are preserved in separate files
- [ ] Complex types (sets, maps) are set correctly
- [ ] All `d.Set()` calls check for errors

### 5. Documentation

- [ ] Description constants use proper formatting (backticks, arrows)
- [ ] Field descriptions specify if value is a UUID
- [ ] Field descriptions mention if changing forces new resource
- [ ] Notes use `->` prefix
- [ ] Warnings use `~>` prefix

### 6. Testing

- [ ] Test files follow naming convention: `resource_<name>_test.go`
- [ ] Tests cover create, read, update, delete operations
- [ ] Tests cover validation and error cases
- [ ] Test configurations are realistic

### 7. Error Handling

- [ ] All errors are properly handled
- [ ] Error messages are descriptive and include context
- [ ] Context is passed to all SDK calls
- [ ] Type assertions are safe

## Common Issues and Patterns

### Issue: Incorrect Acronym Capitalization

**Problem**:
```go
keyVpcId = "vpc_id"  // Wrong
keyAwsApi = "aws_api"  // Wrong
```

**Solution**:
```go
keyVPCID = "vpc_id"  // Correct
keyAWSAPI = "aws_api"  // Correct
```

### Issue: Using Strings for Regions

**Problem**:
```go
func addExocompute(ctx context.Context, region string) error {
    // No type safety
}
```

**Solution**:
```go
import gqlaws "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/regions/aws"

func addExocompute(ctx context.Context, region gqlaws.Region) error {
    regionEnum := region.ToRegionEnum()
    // Type-safe region handling
}
```

### Issue: Missing Error Checks on d.Set()

**Problem**:
```go
d.Set(keyName, resource.Name)  // Ignoring error
```

**Solution**:
```go
if err := d.Set(keyName, resource.Name); err != nil {
    return diag.FromErr(err)
}
```

### Issue: Missing tflog.Trace()

**Problem**:
```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    // Missing trace log
    client, err := m.(*client).polaris()
    // ...
}
```

**Solution**:
```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    tflog.Trace(ctx, "resourceRead")
    
    client, err := m.(*client).polaris()
    // ...
}
```

### Issue: Not Using Constants from names.go

**Problem**:
```go
Schema: map[string]*schema.Schema{
    "account_id": {  // Magic string
        Type: schema.TypeString,
        // ...
    },
}
```

**Solution**:
```go
Schema: map[string]*schema.Schema{
    keyAccountID: {  // Constant from names.go
        Type: schema.TypeString,
        // ...
    },
}
```

### Issue: Missing Documentation Comments

**Problem**:
```go
type ExocomputeConfig struct {
    Region string
    VPCID  string
}
```

**Solution**:
```go
// ExocomputeConfig represents the configuration for an Exocompute deployment.
type ExocomputeConfig struct {
    // Region is the AWS region where Exocompute is deployed.
    Region string
    // VPCID is the ID of the VPC where Exocompute is deployed.
    VPCID  string
}
```

## Feedback Format

When providing feedback, use this format:

### Severity Levels

1. **Critical** - Must be fixed (security issues, data loss, crashes)
2. **Important** - Should be fixed (violates standards, potential bugs)
3. **Minor** - Nice to have (style improvements, optimizations)
4. **Suggestion** - Optional (alternative approaches, enhancements)

### Feedback Template

```
[Severity]: [Issue Title]

Location: [file:line or function name]

Issue: [Description of the problem]

Suggestion: [How to fix it]

Example: [Code example if applicable]
```

## Review Examples

### Example 1: Good Code

```go
// resourceAwsExocompute creates an Exocompute configuration for AWS.
func resourceAwsExocompute() *schema.Resource {
    return &schema.Resource{
        CreateContext: awsCreateExocompute,
        ReadContext:   awsReadExocompute,
        DeleteContext: awsDeleteExocompute,

        Description: description(resourceAWSExocomputeDescription),
        Schema: map[string]*schema.Schema{
            keyID: {
                Type:        schema.TypeString,
                Computed:    true,
                Description: "Exocompute configuration ID (UUID).",
            },
            keyAccountID: {
                Type:         schema.TypeString,
                Required:     true,
                ForceNew:     true,
                Description:  "RSC cloud account ID (UUID). Changing this forces a new resource to be created.",
                ValidateFunc: validation.IsUUID,
            },
        },
    }
}
```

**Review**: ✅ Follows all standards - documentation, constants, validation, descriptions.

### Example 2: Code Needing Improvement

```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    client, err := m.(*client).polaris()
    if err != nil {
        return diag.FromErr(err)
    }

    id, _ := uuid.Parse(d.Id())  // Ignoring error
    resource, err := sdk.GetResource(ctx, id)
    if err != nil {
        return diag.FromErr(err)
    }

    d.Set("name", resource.Name)  // Not checking error, using magic string
    return nil
}
```

**Review**:

**Important**: Missing tflog.Trace() at function entry
- Add `tflog.Trace(ctx, "resourceRead")` at the start of the function

**Critical**: Ignoring error from uuid.Parse()
- Check and handle the error: `if err != nil { return diag.FromErr(err) }`

**Important**: Not checking error from d.Set()
- Wrap in error check: `if err := d.Set(...); err != nil { return diag.FromErr(err) }`

**Important**: Using magic string instead of constant
- Use `keyName` constant from `names.go` instead of `"name"`

