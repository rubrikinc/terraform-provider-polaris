---
type: "agent_requested"
description: "When reviewing code for a PR or acting as a reviewer for another developer"
---

# Code Review Guidelines

## Review Checklist

### 1. Code Standards
- [ ] All exported types/functions have documentation comments
- [ ] Constants defined in `names.go` for schema field keys
- [ ] Always use `secret.String` when adding sensitive fields

### 2. Terraform Patterns
- [ ] Resources: `polaris_<provider>_<resource>`
- [ ] Files: `resource_<provider>_<resource>.go`
- [ ] Description constants use `description()` helper
- [ ] Schema uses constants from `names.go`
- [ ] All schema fields have clear descriptions
- [ ] Appropriate validators used
- [ ] `ForceNew: true` for fields requiring recreation
- [ ] CRUD functions use `tflog.Trace()` at entry
- [ ] CRUD functions return `diag.Diagnostics`
- [ ] Errors use `diag.FromErr()` or `diag.Errorf()`

### 3. SDK Integration
- [ ] Region types used (not strings)
- [ ] SDK wrapper functions used
- [ ] UUIDs use `github.com/google/uuid`
- [ ] GraphQL types imported from SDK

### 4. State Management
- [ ] Schema versioning incremented for breaking changes
- [ ] State upgraders provided
- [ ] All `d.Set()` calls check errors

### 5. Documentation
- [ ] Description constants use proper formatting
- [ ] Field descriptions specify UUID/ForceNew
- [ ] Notes use `->`, warnings use `~>`

### 6. Testing
- [ ] Tests cover CRUD operations
- [ ] Tests cover validation/error cases

## Common Issues

### Using Strings for Regions
❌ `func addExocompute(ctx context.Context, region string)`
✅ `func addExocompute(ctx context.Context, region gqlaws.Region)`

### Missing Error Checks on d.Set()
❌ `d.Set(keyName, resource.Name)`
✅ `if err := d.Set(keyName, resource.Name); err != nil { return diag.FromErr(err) }`

### Missing tflog.Trace()
❌ Missing at function entry
✅ `tflog.Trace(ctx, "resourceRead")` at start of CRUD functions

### Not Using Constants from names.go
❌ `"account_id": {`
✅ `keyAccountID: {`

### Missing Documentation Comments
❌ No comment on exported type
✅ `// ExocomputeConfig represents the configuration for an Exocompute deployment.`

## Feedback Format

**Severity Levels**:
1. **Critical** - Must be fixed (security, data loss, crashes)
2. **Important** - Should be fixed (violates standards, potential bugs)
3. **Minor** - Nice to have (style improvements)
4. **Suggestion** - Optional (alternative approaches)

