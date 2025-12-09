---
type: "agent_requested"
description: "Rules when releasing a new terraform provider"
---

# Release Process

## Pre-Release Checklist

### 1. Update Changelog
- [ ] Update `templates/guides/changelog.md.tmpl` with all changes
- [ ] Follow changelog guidelines for proper formatting
- [ ] Order entries correctly (breaking changes first, then features, then fixes)

### 2. Create Upgrade Guide (if needed)
- [ ] If breaking changes, create `templates/guides/upgrade_guide_v<VERSION>.md.tmpl`
- [ ] Link to upgrade guide from changelog
- [ ] Document error messages and migration paths

### 3. Update Documentation
- [ ] Run `go generate ./...` to regenerate all documentation
- [ ] Verify generated files in `docs/` are correct
- [ ] Check `git diff` to verify documentation changes

### 4. Run Tests
- [ ] Run unit tests: `go test ./...`
- [ ] Run acceptance tests: `TF_ACC=1 go test -count=1 -timeout=120m -v ./...`
- [ ] Ensure all tests pass

### 5. Code Quality Checks
- [ ] Run `go mod tidy`
- [ ] Run `go vet ./...`
- [ ] Run `go generate ./...` and verify no files changed
- [ ] Run `gofmt -d .`
- [ ] Run static analysis: `go run honnef.co/go/tools/cmd/staticcheck@latest ./...`

### 6. Verify CI Pipeline
- [ ] Ensure latest commit on `main` has passed all CI checks
- [ ] Verify no pending PRs that should be included

## Release Steps

### 1. Checkout and Update Main
```bash
git checkout main
git pull origin main
```

### 2. Create Version Tag
```bash
git tag v<MAJOR>.<MINOR>.<PATCH>
```

**Version Numbering**:
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.4.0): New features, backward compatible
- **Patch** (v1.4.1): Bug fixes, backward compatible

### 3. Verify Tag (Dry Run)
```bash
git push --dry-run --tags
```

### 4. Push Tag to GitHub
```bash
git push --tags
```

**What happens**: GitHub Actions triggers, GoReleaser builds binaries, signs artifacts, creates release, publishes to Terraform Registry.

### 5. Verify Tag
```bash
git tag -l
git show v<MAJOR>.<MINOR>.<PATCH>
```

### 6. Monitor Release
1. **GitHub Actions**: Watch `release` workflow
2. **GitHub Releases**: Verify binaries, signatures, checksums
3. **Terraform Registry**: Verify version appears and documentation updated

## Post-Release Tasks

- [ ] Check GitHub release page for completeness
- [ ] Verify all platform binaries present
- [ ] Test downloading new provider version
- [ ] Verify Terraform Registry shows new version
- [ ] Notify team
- [ ] Monitor for issues

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

### Release Workflow Failed

If the GitHub Actions workflow fails:

1. Check the workflow logs for error messages
2. Fix the underlying issue
3. Delete the tag (if release wasn't created)
4. Fix the issue in the code
5. Create a new tag with a patch version

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