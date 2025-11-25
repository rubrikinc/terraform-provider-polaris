---
type: "always_apply"
---

# Release Process Guidelines

This document defines the process for releasing new versions of the Terraform Provider for Rubrik Polaris.

## Overview

The release process is automated through GitHub Actions. When a version tag is pushed to the repository, the GitHub Actions workflow automatically:
1. Builds the provider for multiple platforms (Linux, macOS, Windows, FreeBSD)
2. Signs the release artifacts with GPG
3. Creates a GitHub release
4. Publishes the provider to the Terraform Registry

## Pre-Release Checklist

Before creating a new release, ensure the following tasks are completed:

### 1. Update Changelog

- [ ] Update `templates/guides/changelog.md.tmpl` with all changes for the new version
- [ ] Follow the [Changelog Guidelines](./changelog-guidelines.md) for proper formatting
- [ ] Include all breaking changes, new features, improvements, and bug fixes
- [ ] Order entries correctly (breaking changes first, then features, then fixes)

### 2. Create Upgrade Guide (if needed)

- [ ] If the release contains breaking changes, create an upgrade guide
- [ ] Create `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`
- [ ] Follow the [Documentation Guidelines](./documentation-guidelines.md#upgrade-guides)
- [ ] Link to the upgrade guide from breaking change entries in the changelog
- [ ] Document all breaking changes with error messages and migration paths
- [ ] Document significant changes that may cause unexpected diffs

### 3. Update Documentation

- [ ] Run `go generate ./...` to regenerate all documentation
- [ ] Verify generated files in `docs/` are correct
- [ ] Ensure no manual edits were made to `docs/` directory
- [ ] Check `git diff` to verify documentation changes

### 4. Run Tests

- [ ] Run unit tests: `go test ./...`
- [ ] Run acceptance tests: `TF_ACC=1 go test -count=1 -timeout=120m -v ./...`
- [ ] Ensure all tests pass
- [ ] Fix any failing tests before proceeding

### 5. Code Quality Checks

- [ ] Run `go mod tidy` to clean up dependencies
- [ ] Run `go vet ./...` to check for common errors
- [ ] Run `go generate ./...` and verify no files changed
- [ ] Run `gofmt -d .` to check formatting
- [ ] Run static analysis: `go run honnef.co/go/tools/cmd/staticcheck@latest ./...`

### 6. Verify CI Pipeline

- [ ] Ensure the latest commit on `main` branch has passed all CI checks
- [ ] Check Jenkins pipeline status (if applicable)
- [ ] Verify no pending pull requests that should be included

## Release Process

Follow these steps to create a new release:

### Step 1: Checkout and Update Main Branch

```bash
git checkout main
git pull origin main
```

**Important**: Always ensure you're on the latest `main` branch before creating a release tag.

### Step 2: Create Version Tag

```bash
git tag v<MAJOR>.<MINOR>.<PATCH>
```

**Examples**:
- `git tag v1.4.0` - New minor version with features
- `git tag v1.4.1` - Patch version with bug fixes
- `git tag v2.0.0` - Major version with breaking changes

**Version Numbering**:
- **Major** (v2.0.0): Breaking changes that require user action
- **Minor** (v1.4.0): New features, backward compatible
- **Patch** (v1.4.1): Bug fixes, backward compatible

### Step 3: Verify Tag (Dry Run)

```bash
git push --dry-run --tags
```

**What to check**:
- Verify the tag name is correct
- Ensure you're pushing to the correct remote (`origin`)
- Check that only the intended tag will be pushed

### Step 4: Push Tag to GitHub

```bash
git push --tags
```

**What happens next**:
1. GitHub Actions workflow is triggered automatically
2. GoReleaser builds binaries for all platforms
3. Artifacts are signed with GPG
4. GitHub release is created
5. Provider is published to Terraform Registry

### Step 5: Verify Tag Was Created

```bash
git tag -l
```

**Expected output**: List of all tags including the newly created tag

### Step 6: Inspect Tag Details

```bash
git show v<MAJOR>.<MINOR>.<PATCH>
```

**Example**:
```bash
git show v1.4.0
```

**What to verify**:
- Tag points to the correct commit
- Commit message is appropriate
- Commit includes all intended changes

### Step 7: Monitor Release Process

1. **GitHub Actions**: Navigate to https://github.com/rubrikinc/terraform-provider-polaris/actions
   - Watch the `release` workflow
   - Ensure all steps complete successfully
   - Check for any errors in the build or signing process

2. **GitHub Releases**: Navigate to https://github.com/rubrikinc/terraform-provider-polaris/releases
   - Verify the release was created
   - Check that all platform binaries are attached
   - Verify GPG signatures are present
   - Ensure checksums file is included

3. **Terraform Registry**: Navigate to https://registry.terraform.io/providers/rubrikinc/polaris
   - Wait for the new version to appear (may take a few minutes)
   - Verify the version is listed
   - Check that documentation is updated
   - Test downloading the provider

## Post-Release Tasks

After the release is published:

### 1. Verify Release

- [ ] Check GitHub release page for completeness
- [ ] Verify all platform binaries are present
- [ ] Test downloading and using the new provider version
- [ ] Verify Terraform Registry shows the new version

### 2. Announce Release

- [ ] Update internal documentation (if applicable)
- [ ] Notify team via Slack or other communication channels
- [ ] Update any external documentation or blog posts

### 3. Monitor for Issues

- [ ] Watch for GitHub issues related to the new release
- [ ] Monitor Slack channels for user feedback
- [ ] Be prepared to create a patch release if critical issues are found

## Troubleshooting

### Tag Already Exists

If you need to recreate a tag:

```bash
# Delete local tag
git tag -d v1.4.0

# Delete remote tag (use with caution!)
git push origin :refs/tags/v1.4.0

# Create new tag
git tag v1.4.0

# Push new tag
git push --tags
```

**Warning**: Only delete and recreate tags if the release hasn't been published yet. Never delete tags for published releases.

### Release Workflow Failed

If the GitHub Actions workflow fails:

1. Check the workflow logs for error messages
2. Fix the underlying issue
3. Delete the tag (if release wasn't created)
4. Fix the issue in the code
5. Create a new tag with a patch version

### Wrong Commit Tagged

If you tagged the wrong commit:

```bash
# Delete the incorrect tag
git tag -d v1.4.0
git push origin :refs/tags/v1.4.0

# Checkout the correct commit
git checkout <correct-commit-hash>

# Create the tag on the correct commit
git tag v1.4.0

# Push the corrected tag
git push --tags
```

## Best Practices

1. **Always test before releasing**: Run full test suite including acceptance tests
2. **Update documentation first**: Ensure changelog and upgrade guides are complete
3. **Use semantic versioning**: Follow semver principles for version numbers
4. **Verify before pushing**: Use `--dry-run` to check what will be pushed
5. **Monitor the release**: Watch the GitHub Actions workflow to completion
6. **Communicate clearly**: Update changelog with clear, user-focused descriptions
7. **Plan for rollback**: Know how to handle issues if they arise
8. **Release during business hours**: Easier to handle issues if they occur

## Release Automation

The release process is automated through:

- **GitHub Actions**: `.github/workflows/release.yml`
- **GoReleaser**: `.goreleaser.yml`
- **Trigger**: Pushing a tag matching `v*` pattern

The workflow automatically:
- Builds for multiple platforms (Linux, macOS, Windows, FreeBSD)
- Builds for multiple architectures (amd64, 386, arm, arm64)
- Signs artifacts with GPG
- Creates GitHub release with changelog
- Publishes to Terraform Registry

## Emergency Procedures

### Critical Bug in Released Version

If a critical bug is found in a released version:

1. **Assess severity**: Determine if immediate patch is needed
2. **Create hotfix branch**: Branch from the release tag
3. **Fix the bug**: Make minimal changes to fix the issue
4. **Test thoroughly**: Run full test suite
5. **Create patch release**: Follow normal release process with patch version
6. **Communicate urgency**: Notify users of the critical fix

### Rollback a Release

**Note**: Terraform Registry does not support deleting published versions. If a release has critical issues:

1. **Do not delete the release**: This breaks existing users
2. **Create a patch release**: Fix the issue and release a new version
3. **Update documentation**: Add notes about the problematic version
4. **Communicate clearly**: Inform users to skip the problematic version

