---
type: "always_apply"
---

# Coding Standards

This document defines general coding conventions for the Terraform Provider for Rubrik Polaris project.

## Documentation Requirements

All exported types and functions must have documentation comments following Go conventions:

```go
// ResourceExample represents an example resource in RSC.
type ResourceExample struct {
    // ID is the unique identifier for the resource.
    ID string
    // Name is the human-readable name of the resource.
    Name string
}

// CreateExample creates a new example resource in RSC.
func CreateExample(ctx context.Context, name string) (*ResourceExample, error) {
    // implementation
}
```

**Key Rules**:
- Documentation comments should start with the name of the thing being documented
- Use complete sentences with proper punctuation
- Explain what the type/function does, not how it does it
- For complex functions, include parameter and return value descriptions

## Acronym Capitalization

All acronyms must be fully uppercase to maintain consistency across the codebase.

**Correct Examples**:
- ✅ `AWSAPI` not `AwsApi`
- ✅ `CloudAccountID` not `CloudAccountId`
- ✅ `VPCID` not `VpcId`
- ✅ `CDMURL` not `CdmUrl`
- ✅ `HTTPClient` not `HttpClient`
- ✅ `JSONData` not `JsonData`
- ✅ `UUIDString` not `UuidString`

**Common Acronyms**:
- AWS (Amazon Web Services)
- API (Application Programming Interface)
- ARN (Amazon Resource Name)
- CDM (Cloud Data Management)
- CNP (Cloud Native Protection)
- DSPM (Data Security Posture Management)
- EBS (Elastic Block Store)
- EC2 (Elastic Compute Cloud)
- EKS (Elastic Kubernetes Service)
- GCP (Google Cloud Platform)
- HTTP/HTTPS (HyperText Transfer Protocol)
- ID (Identifier)
- IP (Internet Protocol)
- JSON (JavaScript Object Notation)
- KMS (Key Management Service)
- NTP (Network Time Protocol)
- RSC (Rubrik Security Cloud)
- S3 (Simple Storage Service)
- SLA (Service Level Agreement)
- SQL (Structured Query Language)
- SSO (Single Sign-On)
- URL (Uniform Resource Locator)
- UUID (Universally Unique Identifier)
- VPC (Virtual Private Cloud)
- YAML (YAML Ain't Markup Language)

## Naming Conventions

### Constants

All schema field keys must be defined as constants in `names.go`:

```go
const (
    keyAccountID    = "account_id"
    keyRegion       = "region"
    keyVPCID        = "vpc_id"
)
```

**Key Rules**:
- Use `key` prefix for all schema field constants
- Use camelCase after the `key` prefix
- Maintain alphabetical ordering within logical groups
- Use full uppercase for acronyms (e.g., `keyVPCID`, not `keyVpcId`)

### Resource and Data Source Names

- **Resources**: `polaris_<provider>_<resource>` (e.g., `polaris_aws_account`)
- **Data Sources**: `polaris_<resource>` (e.g., `polaris_user`)
- **Files**: `resource_<provider>_<resource>.go` or `data_source_<resource>.go`

### Function Names

- **CRUD functions**: Use provider prefix (e.g., `awsCreateAccount`, `azureReadSubscription`)
- **Helper functions**: Use descriptive names (e.g., `parseRegion`, `validateAccountID`)
- **Exported functions**: Start with uppercase letter
- **Internal functions**: Start with lowercase letter

## Import Organization

Organize imports in three groups, separated by blank lines:

1. Standard library imports
2. Third-party imports
3. Internal/project imports

```go
import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/hashicorp/terraform-plugin-log/tflog"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

    "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/access"
    gqlaccess "github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/access"
)
```

## Error Handling

### Terraform Provider Errors

Always use `diag.Diagnostics` for error handling in Terraform provider code:

```go
// Convert standard errors
if err != nil {
    return diag.FromErr(err)
}

// Create formatted errors
if invalid {
    return diag.Errorf("invalid configuration: %s", reason)
}
```

### Error Messages

- Use lowercase for error messages (unless starting with a proper noun or acronym)
- Be specific about what went wrong
- Include relevant context (e.g., resource ID, field name)
- Don't include "error:" prefix (it's added automatically)

**Examples**:
- ✅ `"failed to parse account ID: %v"`
- ✅ `"invalid region: %s"`
- ❌ `"Error: Failed to parse account ID"`
- ❌ `"something went wrong"`

## Code Organization

### File Structure

Each resource or data source file should follow this structure:

1. Copyright header (MIT license)
2. Package declaration
3. Imports
4. Description constant
5. Main resource/data source function
6. CRUD operation functions
7. Helper functions

### Function Ordering

Within a file, organize functions in this order:

1. Resource/data source definition function
2. Create function
3. Read function
4. Update function
5. Delete function
6. Helper functions (alphabetically)

## Best Practices

### Use Context

Always pass and use `context.Context` for operations that may be long-running or need cancellation:

```go
func resourceRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
    // Use ctx for SDK calls
    resource, err := sdk.GetResource(ctx, id)
}
```

### Avoid Magic Numbers and Strings

Define constants for magic values:

```go
const (
    defaultTimeout = 30 * time.Minute
    maxRetries     = 3
)
```

### Use Type Assertions Safely

Always check type assertions:

```go
// Correct
client, err := m.(*client).polaris()
if err != nil {
    return diag.FromErr(err)
}

// Also correct for simple cases
name := d.Get(keyName).(string)  // Safe because schema guarantees type
```

