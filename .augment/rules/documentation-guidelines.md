---
type: "always_apply"
---

# Documentation Guidelines

## Critical Rule: Never Manually Edit Generated Documentation

**❌ NEVER manually edit files in the `docs/` directory**

All documentation in `docs/` is auto-generated. **✅ ALWAYS run `go generate ./...` to update documentation**

## Documentation Sources

### 1. Resources/Data Sources WITHOUT Template Files

Documentation is generated from code in `internal/provider/`:

```go
const resourceExampleDescription = `
The ´polaris_example´ resource manages an example resource in RSC.

-> **Note:** Important information.
~> **Warning:** Warning about issues.
`

func resourceExample() *schema.Resource {
    return &schema.Resource{
        Description: description(resourceExampleDescription),
        Schema: map[string]*schema.Schema{
            keyID: {
                Type:        schema.TypeString,
                Computed:    true,
                Description: "Resource ID (UUID).",
            },
        },
    }
}
```

### 2. Resources/Data Sources WITH Template Files

**❌ DO NOT update schema descriptions in code**
**✅ DO update the template file in `templates/resources/` or `templates/data-sources/`**

Add comment in resource code:
```go
// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
```

Always check if the resource has a template file or not.

### 3. Guides

All guides must be in `templates/guides/*.md.tmpl`

## Workflow

### Adding New Resource/Data Source

1. Create code in `internal/provider/` with description constant and schema field descriptions
2. Decide if template needed (complex schema = yes, simple = no)
3. If template: Create in `templates/resources/` or `templates/data-sources/`, add comment in code
4. Create example file in `examples/` (recommended)
5. Run `go generate ./...`
6. Verify generated docs in `docs/`

### Updating Documentation

1. Check if template exists in `templates/`
2. Edit template OR resource code (not both)
3. Run `go generate ./...`
4. Verify changes in `docs/`

## Best Practices

- Always use `description()` helper (converts ´ to `)
- Use `->` for notes, `~>` for warnings
- Mention if field is UUID or forces new resource
- Keep templates in sync with code changes

## Upgrade Guides

### When to Create

Create when release contains breaking changes, significant changes, complex new features, or deprecations.

### Structure

File: `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`

**Required Sections**:
1. Front Matter with page title
2. Before Upgrading (link to changelog)
3. How to Upgrade (version constraint, init, plan, apply)
4. Significant Changes (with error messages and fixes)
5. New Features (with code examples)

### Breaking Changes

Provide: description, why it changed, error messages (console blocks), migration steps, example diffs, code examples.

See `templates/guides/upgrade_guide_v1.3.0.md.tmpl` for complete example.

### New Features

Provide: description, when to use, complete code examples, integration examples, links to docs.

See `templates/guides/upgrade_guide_v1.2.0.md.tmpl` for complete example.

### Deprecations

Provide: what's deprecated, replacement, warning messages, migration examples, removal timeline.

## Changelog Format

**Location**: `templates/guides/changelog.md.tmpl`

### Entry Types (in order)

1. **Breaking Changes**: `* **Breaking Change:** [description]. [why]. [link to upgrade guide].`
2. **Deprecations**: `* **Deprecated:** [field] in [resource] is deprecated. Use [new field] instead.`
3. **New Resources**: `* New resource added for [name] which [description].`
4. **New Data Sources**: `* New data source added for [name] which [description].`
5. **New Features**: `* Add support for [feature] using [resource]. [[docs](../resources/resource.md)]`
6. **Improvements**: `* Improve [what] in [resource].`
7. **Bug Fixes**: `* Fix a bug in [resource] where [description].`

### Breaking Change Rule

When release contains breaking changes:
1. Create upgrade guide in `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`
2. Link to upgrade guide from changelog entry
3. Document error messages and migration paths

## Verification

Before committing:
1. Run `go generate ./...`
2. Check `git diff` for `docs/` changes
3. Verify no manual edits to `docs/`
