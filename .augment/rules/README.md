---
type: "always_apply"
---

# Augment AI Guidelines for Terraform Provider Polaris

This directory contains guidelines and coding standards for the Terraform Provider for Rubrik Polaris project.

## Core Guidelines (Always Loaded)

### [Core Standards](./core-standards.md)
Essential coding standards that apply to all code:
- **Documentation Requirements**: All exported types and functions must have documentation comments
- **Naming Conventions**: Constants, resources, data sources, and functions
- **Error Handling**: Consistent error handling patterns
- **Import Organization**: Standard library, third-party, internal
- **File Structure**: Standard file organization

### [Terraform Patterns](./terraform-patterns.md)
Terraform provider development patterns:
- **Resource Schema Definition**: How to properly define schemas
- **CRUD Operations**: Standard patterns for Create, Read, Update, Delete
- **Common Patterns**: Logging, client access, UUID handling, region handling
- **SDK Integration**: Using SDK wrapper functions and types
- **State Management**: Schema versioning and state upgraders

## Additional Guidelines (Request When Needed)

When working on specific tasks, request the relevant guideline:

### Documentation Work
**Request**: "Show me the documentation guidelines"
**File**: `documentation-guidelines.md`
**Contains**:
- Critical rule: Never manually edit `docs/` directory
- Documentation generation with `go generate ./...`
- Template files vs code-based documentation
- Upgrade guide creation and format
- Changelog format and entry types

### Code Review
**Request**: "Show me the code review guidelines"
**File**: `code-review-guidelines.md`
**Contains**:
- Systematic review checklist
- Common issues and solutions
- Feedback format and severity levels
- Review examples

### Release Process
**Request**: "Show me the release process"
**File**: `release-process.md`
**Contains**:
- Pre-release checklist
- Step-by-step release instructions
- Post-release tasks
- Troubleshooting and emergency procedures

## Quick Reference

### When Adding New Resources or Data Sources

1. **Follow existing patterns** - Look at similar resources/data sources
2. **Use consistent naming** - Follow the `keyXxx` constant pattern in `names.go`
3. **Add proper documentation** - Include description constants
4. **Use proper validation** - Leverage built-in validators or create custom ones
5. **Handle errors consistently** - Use `diag.FromErr()` for error handling
6. **Add logging** - Use `tflog.Trace()` for function entry points

### When Working with the SDK

1. **Use region types** - Import and use `aws.Region`, `azure.Region`, `gcp.Region` types
2. **Follow SDK patterns** - Use the SDK's wrapper functions (e.g., `access.Wrap(client)`)
3. **Handle UUIDs properly** - Use `github.com/google/uuid` for UUID handling
4. **Use GraphQL types** - Import GraphQL types from the SDK

## How to Request Additional Guidelines

When you need specific guidelines, simply ask:
- "Show me the documentation guidelines" - For documentation work
- "Show me the code review guidelines" - For reviewing code
- "Show me the release process" - For creating releases

This keeps the context window small while giving you access to detailed guidelines when needed.

