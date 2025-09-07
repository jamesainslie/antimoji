# Release Process Documentation

This document outlines the complete release process for antimoji, including critical configuration details and common pitfalls to avoid.

## Overview

Antimoji uses an automated release process powered by GitHub Actions and GoReleaser. When a new tag is pushed, the system automatically:

1. Builds binaries for multiple platforms
2. Creates GitHub releases with artifacts
3. Updates the Homebrew tap formula
4. Builds and pushes Docker images
5. Runs comprehensive tests and security scans

## Critical Configuration Requirements

### 1. Homebrew Formula Placement

**⚠️ CRITICAL**: The GoReleaser configuration MUST include `folder: Formula` in the brews section.

```yaml
brews:
  - name: antimoji
    repository:
      owner: jamesainslie
      name: homebrew-antimoji
      token: "{{ .Env.PAT_TOKEN }}"
      branch: main
    folder: Formula  # ← CRITICAL: Without this, formula files go to repository root
```

**Why this matters:**
- Homebrew expects formula files in the `Formula/` directory
- Without `folder: Formula`, GoReleaser places `.rb` files at the repository root
- This causes silent failures where the formula appears to update but Homebrew can't find it
- Users will be unable to upgrade via `brew upgrade antimoji`

### 2. Personal Access Token (PAT)

**⚠️ CRITICAL**: Use `PAT_TOKEN`, not `GITHUB_TOKEN` for Homebrew updates.

```yaml
token: "{{ .Env.PAT_TOKEN }}"  # ← CORRECT
# token: "{{ .Env.GITHUB_TOKEN }}"  # ← WRONG: Insufficient permissions
```

**Why this matters:**
- `GITHUB_TOKEN` has limited permissions and cannot push to external repositories
- `PAT_TOKEN` must have `repo` and `public_repo` permissions
- Without proper token, Homebrew updates will silently fail

### 3. Repository Naming Convention

The Homebrew tap repository MUST follow the naming convention: `homebrew-{project}`

```yaml
repository:
  owner: jamesainslie
  name: homebrew-antimoji  # ← CORRECT: homebrew-{project}
```

## Release Checklist

### Pre-Release

- [ ] All tests passing on main branch
- [ ] Version updated in appropriate files
- [ ] CHANGELOG.md updated with new version
- [ ] All PRs merged and main branch is clean

### Creating a Release

1. **Update CHANGELOG.md** with the new version and changes
2. **Commit the changelog** with message: `docs: update CHANGELOG for vX.Y.Z release`
3. **Create and push tag**:
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```
4. **Monitor GitHub Actions** for successful completion

### Post-Release Verification

**⚠️ CRITICAL**: Always verify these steps after each release:

1. **Check Homebrew Formula Location**:
   - Visit: https://github.com/jamesainslie/homebrew-antimoji/blob/main/Formula/antimoji.rb
   - Verify the version number matches your release
   - Ensure file is in `Formula/` directory, not at root

2. **Test Homebrew Upgrade**:
   ```bash
   brew update
   brew upgrade antimoji
   antimoji --version  # Should show new version
   ```

3. **Verify GitHub Release**:
   - Check release was created with all artifacts
   - Verify release notes are populated
   - Confirm all platform binaries are present

4. **Test Docker Images**:
   ```bash
   docker run --rm ghcr.io/jamesainslie/antimoji:latest --version
   ```

## Common Issues and Solutions

### Issue: Homebrew Formula Not Updating

**Symptoms:**
- GoReleaser logs show successful push
- `brew upgrade antimoji` shows "already installed"
- Users report they can't upgrade to new version

**Root Cause:**
- Missing `folder: Formula` in GoReleaser config
- Formula file placed at repository root instead of `Formula/` directory

**Solution:**
1. Add `folder: Formula` to `.goreleaser.yaml` brews configuration
2. Manually move incorrectly placed formula file from root to `Formula/` directory
3. Delete any duplicate files at repository root

### Issue: Silent Homebrew Update Failures

**Symptoms:**
- GoReleaser completes without errors
- No commit appears in homebrew-antimoji repository
- Formula version remains unchanged

**Root Cause:**
- Using `GITHUB_TOKEN` instead of `PAT_TOKEN`
- Insufficient permissions for cross-repository operations

**Solution:**
1. Ensure `PAT_TOKEN` secret is configured in repository settings
2. Verify PAT has `repo` and `public_repo` permissions
3. Update `.goreleaser.yaml` to use `PAT_TOKEN`

### Issue: Homebrew Tap Not Found

**Symptoms:**
- `brew install antimoji` fails with "No available formula"
- Tap appears to exist but formulas not discoverable

**Root Cause:**
- Incorrect repository naming convention
- Repository not named `homebrew-{project}`

**Solution:**
1. Rename repository to follow `homebrew-{project}` convention
2. Update GoReleaser configuration with correct repository name

## Emergency Procedures

### Fixing a Broken Release

If a release is created but Homebrew formula is broken:

1. **Identify the Issue**:
   - Check formula location: root vs `Formula/` directory
   - Verify version numbers in formula file
   - Confirm repository structure

2. **Quick Fix**:
   ```bash
   # Clone homebrew tap
   git clone https://github.com/jamesainslie/homebrew-antimoji.git
   cd homebrew-antimoji
   
   # Move formula to correct location if needed
   mv antimoji.rb Formula/antimoji.rb
   
   # Update version if incorrect
   sed -i 's/version "OLD_VERSION"/version "NEW_VERSION"/g' Formula/antimoji.rb
   
   # Commit and push
   git add Formula/antimoji.rb
   git commit -m "fix: correct formula placement and version"
   git push origin main
   ```

3. **Verify Fix**:
   ```bash
   brew update
   brew upgrade antimoji
   ```

### Rolling Back a Release

If a release is fundamentally broken:

1. **Delete the GitHub Release** (if necessary)
2. **Delete the Git Tag**:
   ```bash
   git tag -d vX.Y.Z
   git push origin :refs/tags/vX.Y.Z
   ```
3. **Revert Homebrew Formula** to previous working version
4. **Create New Release** with fixes

## Monitoring and Alerts

- Monitor GitHub Actions for failed releases
- Check Homebrew tap repository for unexpected changes
- Verify Docker images are building and pushing correctly
- Test upgrade path on different platforms

## References

- [GoReleaser Homebrew Documentation](https://goreleaser.com/customization/homebrew/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

---

**Remember**: The key to successful releases is verification. Always check that Homebrew formulas are correctly placed in the `Formula/` directory and that users can actually upgrade to the new version.
