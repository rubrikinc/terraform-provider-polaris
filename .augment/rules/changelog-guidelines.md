# Changelog Guidelines

This document defines the standards for maintaining the changelog in the Terraform Provider for Rubrik Polaris project.

## Changelog Location

The changelog is maintained in the template file:
- **File**: `templates/guides/changelog.md.tmpl`
- **Generated file**: `docs/guides/changelog.md` (auto-generated, do not edit directly)

## Changelog Format

### Version Header

Each release should start with a version header in the format:

```markdown
## v<MAJOR>.<MINOR>.<PATCH>
```

Example:
```markdown
## v1.4.0
```

### Entry Format

Each changelog entry should be a bullet point describing a single change. Entries should be:
- Clear and concise
- Written in present tense
- Focused on user-facing changes
- Ordered by importance (breaking changes first, then new features, then improvements, then bug fixes)

### Entry Types and Examples

#### 1. Breaking Changes

Breaking changes **MUST** be listed first and clearly marked with `**Breaking Change:**` prefix.

**Format**:
```markdown
* **Breaking Change:** [Description of what changed]. [Explanation of why]. [Link to upgrade guide if applicable].
```

**Example**:
```markdown
* **Breaking Change:** The `kms_master_key` field of the `polaris_aws_archival_location` resource is now required and
  no longer has a default value. Previously, the field was optional with a default value of `aws/s3`. This default
  value was only valid for source region archival locations. Due to a bug fix in RSC, the default value is no longer
  accepted for specific region archival locations. See the [upgrade guide](upgrade_guide_v1.4.0.md) for migration
  instructions.
```

**Important**: When a release contains breaking changes, **ALWAYS**:
1. Create an upgrade guide in `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`
2. Link to the upgrade guide from the breaking change entry in the changelog
3. See [Documentation Guidelines](./documentation-guidelines.md#upgrade-guides) for upgrade guide format

#### 2. New Resources

New resources should be clearly identified with "New resource added" or similar language.

**Format**:
```markdown
* New resource added for `<resource_name>` which [description of what it does].
```

**Examples**:
```markdown
* New resource added for `azure_cces_cloud_cluster` which deploys new cloud clusters in Azure and manages resources through RSC.
* New resource `polaris_aws_exocompute` for managing AWS Exocompute configurations.
```

#### 3. New Data Sources

**Format**:
```markdown
* New data source added for `<data_source_name>` which [description of what it provides].
```

**Example**:
```markdown
* New data source `polaris_deployment` for accessing RSC deployment information.
```

#### 4. New Features

Features added to existing resources should describe what capability was added.

**Format**:
```markdown
* Add support for [feature] using the `<resource_name>` resource. [[docs](../resources/<resource>.md)]
```

**Examples**:
```markdown
* Add support for adding AWS Cloud Cluster with Elastic Storage. [[docs](../resources/aws_cloud_cluster.md)]
* Add support for onboarding the RSC feature `SERVERS_AND_APPS` using the `polaris_cnp_aws_account` resource.
  [[docs](../resources/aws_cnp_account.md)]
* Add support for creating DSPM, Data Scanning and Outpost features under `polaris_aws_account`.
  [[docs](../resources/aws_account.md)]
```

#### 5. Improvements

Improvements to existing functionality.

**Format**:
```markdown
* Improve [what was improved] in `<resource_name>` resource.
* Update [what was updated] to [new behavior].
```

**Example**:
```markdown
* Improve error messages when AWS account onboarding fails.
* Update region validation to support new AWS regions.
```

#### 6. Bug Fixes

Bug fixes should describe what was fixed and the impact.

**Format**:
```markdown
* Fix a bug in the `<resource_name>` resource where [description of the bug].
```

**Example**:
```markdown
* Fix a bug in the `polaris_azure_subscription` resource where the wrong mutation was used to update the subscription
  when the subscription was updated to use permission groups and a resource group at the same time.
```

#### 7. Deprecations

Deprecations should be clearly marked.

**Format**:
```markdown
* **Deprecated:** The `<field_name>` field in `<resource_name>` is deprecated and will be removed in v<version>.
  Use `<new_field_name>` instead.
```

## Documentation Links

When referencing documentation, use relative links in the format:

```markdown
[[docs](../resources/<resource_name>.md)]
[[docs](../data-sources/<data_source_name>.md)]
[[docs](<upgrade_guide_file>.md)]
```

## Ordering Rules

Within each version section, order entries as follows:

1. **Breaking Changes** (always first, marked with `**Breaking Change:**`)
2. **Deprecations** (marked with `**Deprecated:**`)
3. **New Resources** (start with "New resource")
4. **New Data Sources** (start with "New data source")
5. **New Features** (start with "Add support for")
6. **Improvements** (start with "Improve" or "Update")
7. **Bug Fixes** (start with "Fix")

## Complete Example

```markdown
## v1.4.0
* **Breaking Change:** The `kms_master_key` field of the `polaris_aws_archival_location` resource is now required and
  no longer has a default value. Previously, the field was optional with a default value of `aws/s3`. This default
  value was only valid for source region archival locations. Due to a bug fix in RSC, the default value is no longer
  accepted for specific region archival locations. See the [upgrade guide](upgrade_guide_v1.4.0.md) for migration
  instructions.
* New resource added for `azure_cces_cloud_cluster` which deploys new cloud clusters in Azure and manages resources through RSC.
* New data source `polaris_features` for accessing enabled RSC features.
* Add support for DSPM feature in `polaris_aws_account` resource. [[docs](../resources/aws_account.md)]
* Improve error handling when cloud account onboarding fails.
* Fix a bug in the `polaris_azure_subscription` resource where region validation was too strict.

## v1.3.0
* New resource `polaris_role_assignment` for managing user and group role assignments.
* Add support for custom tags in AWS resources. [[docs](../resources/aws_custom_tags.md)]
* Fix a bug where UUID validation was not applied to all ID fields.
```

## Best Practices

1. **Be Specific**: Describe exactly what changed, not just that something changed
2. **User-Focused**: Write from the perspective of someone using the provider
3. **Include Context**: Explain why a change was made if it's not obvious
4. **Link Documentation**: Always link to relevant documentation for new features
5. **Breaking Changes**: Always provide migration guidance for breaking changes
6. **Consistent Formatting**: Follow the exact format patterns shown above
7. **Chronological Order**: Newest versions at the top of the file

## When to Update

Update the changelog:
- ✅ When adding a new resource or data source
- ✅ When adding a new feature to an existing resource
- ✅ When making a breaking change
- ✅ When fixing a user-visible bug
- ✅ When deprecating functionality
- ❌ For internal refactoring that doesn't affect users
- ❌ For documentation-only changes (unless significant)
- ❌ For test-only changes

