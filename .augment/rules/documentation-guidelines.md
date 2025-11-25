---
type: "always_apply"
---

# Documentation Guidelines

This document defines the standards for documenting resources, data sources, and guides in the Terraform Provider for Rubrik Polaris project.

## Critical Rule: Never Manually Edit Generated Documentation

**❌ NEVER manually edit files in the `docs/` directory**

All documentation in the `docs/` directory is auto-generated from:
1. Template files in `templates/`
2. Resource/data source code in `internal/provider/`
3. Example files in `examples/`

**✅ ALWAYS run `go generate ./...` to update documentation**

## Documentation Generation

### How Documentation is Generated

The project uses `terraform-plugin-docs` to automatically generate documentation. The generation is triggered by:

```bash
go generate ./...
```

This command is defined in `main.go`:
```go
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name terraform-provider-polaris
```

### What Gets Generated

The tool generates documentation in the `docs/` directory:
- `docs/index.md` - Provider documentation (from `templates/index.md.tmpl`)
- `docs/resources/*.md` - Resource documentation
- `docs/data-sources/*.md` - Data source documentation
- `docs/guides/*.md` - Guide documentation (from `templates/guides/*.md.tmpl`)

## Documentation Sources

### 1. Resources and Data Sources WITHOUT Template Files

For resources/data sources that **do not** have a template file in `templates/`, the documentation is generated from:

**Source**: The resource/data source code in `internal/provider/`

**What to document in code**:
- Description constant (e.g., `resourceAWSExocomputeDescription`)
- Schema field descriptions
- Example files in `examples/resources/` or `examples/data-sources/`

**Example** (resource without template):
```go
const resourceExampleDescription = `
The ´polaris_example´ resource manages an example resource in RSC.

-> **Note:** Important information about the resource.

~> **Warning:** Warning about potential issues.
`

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

### 2. Resources and Data Sources WITH Template Files

For resources/data sources that **have** a template file in `templates/resources/` or `templates/data-sources/`:

**❌ DO NOT update schema descriptions in the resource code**
**✅ DO update the template file instead**

**Template files that exist**:
- `templates/resources/aws_cloud_cluster.md.tmpl`
- `templates/resources/aws_cnp_account.md.tmpl`
- `templates/resources/aws_cnp_account_trust_policy.md.tmpl`
- `templates/resources/aws_custom_tags.md.tmpl`
- `templates/resources/aws_exocompute.md.tmpl`
- `templates/resources/azure_cloud_cluster.md.tmpl`
- `templates/resources/azure_custom_tags.md.tmpl`
- `templates/resources/azure_exocompute.md.tmpl`
- `templates/resources/cdm_bootstrap.md.tmpl`
- `templates/resources/cdm_bootstrap_cces_aws.md.tmpl`
- `templates/resources/cdm_bootstrap_cces_azure.md.tmpl`
- `templates/resources/gcp_custom_labels.md.tmpl`
- `templates/data-sources/aws_cnp_permissions.md.tmpl`
- `templates/data-sources/azure_archival_location.md.tmpl`
- `templates/data-sources/gcp_archival_location.md.tmpl`
- `templates/data-sources/gcp_project.md.tmpl`
- `templates/data-sources/role.md.tmpl`
- `templates/data-sources/role_template.md.tmpl`
- `templates/data-sources/sso_group.md.tmpl`
- `templates/data-sources/user.md.tmpl`

**Example comment in resource code**:
```go
// This resource uses a template for its documentation, remember to update the
// template if the documentation for any field changes.
func resourceAwsExocompute() *schema.Resource {
    // ...
}
```

**Template file structure**:
```markdown
---
page_title: "{{.Name}} {{.Type}} - {{.ProviderName}}"
subcategory: ""
description: |-
  {{.Description}}
---

# {{.Name}} ({{.Type}})

{{.Description}}

{{if .HasExample}}
## Example Usage

{{tffile .ExampleFile}}
{{end}}

## Schema

### Required

- `account_id` (String) RSC cloud account ID (UUID). Changing this forces a new resource to be created.

### Optional

- `region` (String) AWS region.

### Read-Only

- `id` (String) Exocompute configuration ID (UUID).
```

### 3. Guides

All guide documentation must be in template files:

**Location**: `templates/guides/*.md.tmpl`

**Examples**:
- `templates/guides/changelog.md.tmpl`
- `templates/guides/aws_cnp_account.md.tmpl`
- `templates/guides/permissions.md.tmpl`
- `templates/guides/users_and_roles.md.tmpl`
- `templates/guides/upgrade_guide_v*.md.tmpl`

## Documentation Workflow

### When Adding a New Resource or Data Source

1. **Create the resource/data source code** in `internal/provider/`
   - Include description constant
   - Include schema field descriptions
   - Add comment if using template (see below)

2. **Decide if a template is needed**:
   - ✅ Use template if: Complex schema, custom formatting, special sections
   - ❌ No template if: Standard resource with simple schema

3. **If using a template**:
   - Create template file in `templates/resources/` or `templates/data-sources/`
   - Add comment in resource code: `// This resource uses a template for its documentation...`
   - Document schema fields in the template

4. **Create example file** (optional but recommended):
   - Create file in `examples/resources/<resource_name>/resource.tf`
   - Or `examples/data-sources/<data_source_name>/data-source.tf`

5. **Generate documentation**:
   ```bash
   go generate ./...
   ```

6. **Verify generated documentation**:
   - Check `docs/resources/<resource_name>.md` or `docs/data-sources/<data_source_name>.md`
   - Ensure all fields are documented
   - Ensure examples are included

### When Updating Existing Documentation

1. **Identify the source**:
   - Check if template exists in `templates/`
   - If template exists, edit the template
   - If no template, edit the resource/data source code

2. **Make changes**:
   - Update description constants
   - Update schema field descriptions
   - Update template files if applicable

3. **Regenerate documentation**:
   ```bash
   go generate ./...
   ```

4. **Verify changes**:
   - Check generated files in `docs/`
   - Ensure changes are reflected correctly

## Best Practices

1. **Always use the `description()` helper** for description constants
   - Converts acute accents (´) to backticks (`)
   - Example: `Description: description(resourceExampleDescription)`

2. **Use proper Markdown formatting** in descriptions:
   - Use backticks for code/resource names: ´polaris_example´
   - Use `->` for notes
   - Use `~>` for warnings

3. **Be specific in field descriptions**:
   - Mention if field is a UUID
   - Mention if changing forces new resource
   - Include valid values or ranges

4. **Add examples** for complex resources:
   - Create example files in `examples/`
   - Show common use cases
   - Include comments explaining the configuration

5. **Keep templates in sync** with code:
   - When adding/removing schema fields, update templates
   - When changing field types, update templates
   - Add reminder comments in code

## Common Mistakes to Avoid

❌ **Editing files in `docs/` directly** - These are auto-generated and will be overwritten

❌ **Forgetting to run `go generate ./...`** - Documentation won't be updated

❌ **Not updating templates** when schema changes - Documentation will be out of sync

❌ **Inconsistent formatting** - Use the same patterns as existing documentation

❌ **Missing field descriptions** - All schema fields should have clear descriptions

## Upgrade Guides

### When to Create an Upgrade Guide

Create an upgrade guide when a release contains:
- ✅ **Breaking changes** that require user action
- ✅ **Significant changes** that may cause unexpected diffs or errors
- ✅ **New features** that require complex configuration
- ✅ **Deprecations** that users need to migrate away from
- ❌ Minor bug fixes or internal changes

### Upgrade Guide Structure

All upgrade guides must be created as template files in `templates/guides/` with the naming pattern:
- `upgrade_guide_v<MAJOR>.<MINOR>.<PATCH>.md.tmpl`

**Example**: `upgrade_guide_v1.4.0.md.tmpl`

### Standard Upgrade Guide Format

Every upgrade guide should follow this structure (see existing upgrade guides in `templates/guides/` for complete examples):

**File**: `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`

**Required Sections**:

1. **Front Matter**: YAML header with page title
2. **Before Upgrading**: Link to changelog and deprecation warnings
3. **How to Upgrade**: Step-by-step upgrade instructions
   - Configure version constraint using `~>` operator
   - Run `terraform init -upgrade`
   - Run `terraform plan` to validate
   - Run `terraform apply -refresh-only` to migrate state
4. **Significant Changes**: Breaking changes and significant changes
5. **New Features**: New features requiring explanation

**Template Structure**:
- Start with YAML front matter: `page_title: "Upgrade Guide: v<VERSION>"`
- Include "Before Upgrading" section linking to changelog
- Provide "How to Upgrade" with terraform version configuration example
- Document "Significant Changes" with error messages and fixes
- Document "New Features" with code examples

### Documenting Breaking Changes

For each breaking change, provide:

1. **Clear description** of what changed
2. **Why it changed** (if relevant)
3. **Error messages** users might see (in console blocks)
4. **Step-by-step migration instructions**
5. **Example diffs** showing before/after state
6. **Code examples** showing the fix

**Example Pattern**:

See `templates/guides/upgrade_guide_v1.3.0.md.tmpl` for a complete example. The pattern includes:
- Section heading describing the change (e.g., "### Resource Field Now Required")
- Description of what changed and why
- Error message in a console code block showing what users will see
- Step-by-step instructions to resolve the error
- Command examples (e.g., `terraform state show`) to help users migrate

### Documenting New Features

For each new feature that requires explanation, provide:

1. **Feature description** and purpose
2. **When to use it**
3. **Complete code examples** showing usage
4. **Integration examples** with other resources
5. **Links to related documentation**

**Example Pattern**:

See `templates/guides/upgrade_guide_v1.2.0.md.tmpl` for complete examples. The pattern includes:
- Section heading with feature name (e.g., "### Data Scanning Cyber Assisted Recovery")
- Description of what the feature does and when to use it
- Complete Terraform code examples showing the feature configuration
- Integration with other resources if applicable
- Links to resource documentation

### Documenting Deprecations

For deprecations, provide:

1. **What is deprecated**
2. **What to use instead** (if applicable)
3. **Warning messages** users will see
4. **Migration examples** showing before/after
5. **Timeline** for removal (if known)

**Example Pattern**:

See `templates/guides/upgrade_guide_v1.2.0.md.tmpl` for a complete example. The pattern includes:
- Section heading with what is deprecated (e.g., "### Trust Policy Features Field Deprecated")
- Description of what is deprecated and what to use instead
- Warning message in a console code block showing what users will see
- Migration instructions showing the safe removal process
- Expected diff output showing the in-place update

### Best Practices for Upgrade Guides

1. **Be comprehensive** - Cover all breaking changes and significant changes
2. **Show actual errors** - Include real error messages users will see
3. **Provide examples** - Show complete, working code examples
4. **Use console blocks** - Format error messages and command output in console blocks
5. **Link to changelog** - Always reference the changelog for full details
6. **Test instructions** - Verify all migration steps actually work
7. **Be empathetic** - Understand users may have large configurations to update

### Upgrade Guide Checklist

When creating an upgrade guide:

- [ ] Created template file in `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`
- [ ] Included standard structure (Before Upgrading, How to Upgrade, Significant Changes, New Features)
- [ ] Documented all breaking changes with error messages and fixes
- [ ] Documented significant changes that may cause unexpected diffs
- [ ] Documented new features that require complex configuration
- [ ] Documented all deprecations with migration paths
- [ ] Included complete, tested code examples
- [ ] Included example error messages and diffs
- [ ] Referenced the changelog
- [ ] Ran `go generate ./...` to generate the documentation
- [ ] Verified generated documentation in `docs/guides/`
- [ ] Updated changelog to reference the upgrade guide (if breaking changes exist)

## Verification

Before committing documentation changes:

1. Run `go generate ./...`
2. Check `git diff` to see what changed in `docs/`
3. Verify all changes are intentional
4. Ensure no manual edits were made to `docs/`

The CI pipeline will fail if generated files are out of sync:

```bash
go generate ./... >/dev/null
git diff --exit-code || (echo "Generated files are out of sync. Please run go generate and commit the changes." && exit 1)
```
