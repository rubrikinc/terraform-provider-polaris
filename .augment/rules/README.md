# Augment AI Guidelines for Terraform Provider Polaris

This directory contains guidelines and coding standards for the Terraform Provider for Rubrik Polaris project, specifically formatted for Augment AI to understand and enforce.

## Overview

These guidelines ensure consistency, type safety, and maintainability across the Terraform provider codebase. All contributors and AI assistants should follow these standards when making changes to the project.

## Guidelines Documents

### 1. [Terraform Best Practices](./terraform-best-practices.md)

Defines Terraform provider development best practices including:
- **Resource and Data Source Structure**: Standard patterns for implementing resources and data sources
- **Schema Definitions**: How to properly define and document schemas
- **CRUD Operations**: Patterns for Create, Read, Update, Delete operations
- **State Management**: Proper handling of Terraform state
- **Error Handling**: Consistent error handling patterns

### 2. [Coding Standards](./coding-standards.md)

Defines general coding conventions including:
- **Documentation Requirements**: All types and functions must have documentation comments
- **Acronym Capitalization**: All acronyms must be fully uppercase (e.g., `AWS`, `CDM`, `API`, `ID`)
- **Naming Conventions**: Consistent naming patterns for resources, data sources, and fields
- General Go best practices

### 3. [Code Review Guide](./code-review-guide.md)

Comprehensive guide for Augment AI to act as a code reviewer:
- **Review Checklist**: Systematic approach to reviewing code
- **Common Issues**: Patterns from historical code reviews
- **Feedback Format**: How to provide constructive feedback
- **Severity Levels**: Critical, Important, Minor, Suggestion

### 4. [Changelog Guidelines](./changelog-guidelines.md)

Defines standards for maintaining the changelog:
- **Changelog Format**: Version headers and entry formatting
- **Entry Types**: Breaking changes, new resources, features, improvements, bug fixes
- **Ordering Rules**: How to order different types of changes
- **Documentation Links**: Proper linking to resource documentation
- **Best Practices**: When and how to update the changelog

### 5. [Documentation Guidelines](./documentation-guidelines.md)

Defines standards for documenting resources, data sources, and guides:
- **Critical Rule**: Never manually edit files in `docs/` - always run `go generate ./...`
- **Documentation Sources**: Where to update documentation (templates vs. resource code)
- **Template Files**: Which resources/data sources use template files
- **Documentation Workflow**: How to add and update documentation
- **Best Practices**: Formatting, field descriptions, examples
- **Verification**: How to verify documentation changes before committing

### 6. [Release Process](./release-process.md)

Defines the process for releasing new provider versions:
- **Pre-Release Checklist**: Changelog, upgrade guides, documentation, tests, code quality
- **Release Steps**: Git commands for tagging and pushing releases
- **Automation**: GitHub Actions workflow and GoReleaser configuration
- **Post-Release Tasks**: Verification, announcements, monitoring
- **Troubleshooting**: Handling common release issues
- **Best Practices**: Testing, versioning, communication

## Quick Reference

### When Adding New Resources or Data Sources

1. **Follow existing patterns** - Look at similar resources/data sources for structure
2. **Use consistent naming** - Follow the `keyXxx` constant pattern in `names.go`
3. **Add proper documentation** - Include description constants with examples
4. **Use proper validation** - Leverage built-in validators or create custom ones
5. **Handle errors consistently** - Use `diag.FromErr()` for error handling
6. **Add logging** - Use `tflog.Trace()` for function entry points

### When Working with the SDK

1. **Use region types** - Import and use `aws.Region`, `azure.Region`, `gcp.Region` types
2. **Follow SDK patterns** - Use the SDK's wrapper functions (e.g., `access.Wrap(client)`)
3. **Handle UUIDs properly** - Use `github.com/google/uuid` for UUID handling
4. **Use GraphQL types** - Import GraphQL types from the SDK (e.g., `gqlaccess.User`)

## Enforcement

These guidelines should be enforced by:
1. Code review
2. Augment AI when making code suggestions or changes
3. Linting and static analysis tools where applicable

## Questions?

If you're unsure about how to apply these guidelines, refer to existing code in the repository for examples, or consult the detailed guideline documents in this directory.

