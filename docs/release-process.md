# Release Process

This document describes the complete release process for Antimoji, from preparation to distribution.

## Prerequisites

### 1. Repository Setup
Ensure your GitHub repository has the following configured:

#### Required Permissions
- Actions: Read and write permissions
- Contents: Write permissions  
- Packages: Write permissions
- Pull requests: Write permissions

#### Optional Secrets (for enhanced features)
- `CODECOV_TOKEN`: For code coverage reporting
- `HOMEBREW_TOKEN`: For automated Homebrew formula updates

### 2. Local Development Setup
```bash
# Install development tools
make install-tools

# Set up development environment
make dev-setup

# Verify everything works
make check-all
```

## Release Types

### 1. Development Builds
Triggered automatically on pushes to `main` and `develop`:

```bash
# Manual snapshot build
make release-snapshot

# Local testing
make release-local
```

### 2. Official Releases
Triggered by pushing version tags:

```bash
# Prepare release
make release-prepare VERSION=v1.0.0

# Push tag to trigger release
git push origin v1.0.0
```

## Step-by-Step Release Process

### Step 1: Pre-Release Validation

```bash
# 1. Ensure clean working directory
git status

# 2. Pull latest changes
git pull origin main

# 3. Run comprehensive quality checks
make check-all

# 4. Test local release build
make release-local

# 5. Review changelog and version
git log --oneline --since="last release"
```

### Step 2: Version and Changelog

```bash
# 1. Update CHANGELOG.md with new version
# 2. Update version references if needed
# 3. Commit changes
git add CHANGELOG.md
git commit -m "chore: prepare for release v1.0.0"
```

### Step 3: Create Release

```bash
# 1. Create and push tag
make release-prepare VERSION=v1.0.0
git push origin v1.0.0

# 2. Monitor GitHub Actions workflow
# 3. Verify release completion
```

### Step 4: Post-Release Verification

```bash
# 1. Check GitHub release page
# 2. Test binary downloads
# 3. Verify Docker images
docker pull ghcr.io/antimoji/antimoji:v1.0.0

# 4. Test Homebrew installation (if configured)
brew install antimoji/tap/antimoji
```

## Release Checklist

### Pre-Release
- [ ] All CI checks passing on main branch
- [ ] Documentation is up to date
- [ ] CHANGELOG.md updated
- [ ] Version number follows semantic versioning
- [ ] Local release test successful (`make release-local`)
- [ ] No open critical issues

### During Release
- [ ] Tag created and pushed
- [ ] GitHub Actions workflow triggered
- [ ] All workflow jobs completed successfully
- [ ] Release artifacts generated
- [ ] Docker images published
- [ ] Homebrew formula updated (if configured)

### Post-Release
- [ ] Release notes reviewed and published
- [ ] Binary downloads tested
- [ ] Docker images verified
- [ ] Package manager installations tested
- [ ] Documentation updated with new version
- [ ] Announcement prepared (if needed)

## Distribution Channels

### 1. GitHub Releases
- **Location**: https://github.com/antimoji/antimoji/releases
- **Content**: 
  - Binaries for Linux (amd64, arm64)
  - Binaries for macOS (amd64, arm64)  
  - Binaries for Windows (amd64)
  - Archive files with documentation
  - Checksums file
  - Automated changelog

### 2. Docker Images
- **Registry**: GitHub Container Registry (ghcr.io)
- **Images**:
  - `ghcr.io/antimoji/antimoji:latest`
  - `ghcr.io/antimoji/antimoji:v1.0.0`
- **Platforms**: linux/amd64
- **Base**: Scratch (minimal security footprint)

### 3. Package Managers

#### Homebrew (macOS)
```bash
brew install antimoji/tap/antimoji
```

#### Go Install
```bash
go install github.com/antimoji/antimoji/cmd/antimoji@latest
```

## Versioning Strategy

### Semantic Versioning
- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible functionality additions  
- **PATCH**: Backward-compatible bug fixes

### Tag Format
- Production releases: `v1.0.0`, `v1.1.0`, `v1.0.1`
- Pre-releases: `v1.0.0-rc.1`, `v1.0.0-beta.1`
- Development: Automatic snapshot versioning

### Branch Strategy
- **main**: Production-ready code
- **develop**: Integration branch for features
- **feature/***: Feature development branches
- **hotfix/***: Critical bug fixes

## Rollback Process

### If Release Fails
1. **Delete the tag**:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Fix issues and re-release**:
   ```bash
   # Fix issues
   git commit -m "fix: release issues"
   
   # Re-create tag
   make release-prepare VERSION=v1.0.0
   git push origin v1.0.0
   ```

### If Released Version Has Issues
1. **Create hotfix**:
   ```bash
   git checkout -b hotfix/v1.0.1
   # Fix issues
   git commit -m "fix: critical issue"
   git checkout main
   git merge hotfix/v1.0.1
   ```

2. **Release hotfix**:
   ```bash
   make release-prepare VERSION=v1.0.1
   git push origin v1.0.1
   ```

## Monitoring and Analytics

### GitHub Actions
- Monitor workflow execution times
- Track artifact download counts
- Review security scan results
- Monitor dependency update notifications

### Release Metrics
- Download counts per platform
- Docker image pull statistics
- Package manager installation metrics
- User feedback and issue reports

## Maintenance Schedule

### Weekly (Automated)
- Dependency update checks
- Vulnerability scanning
- Code quality audits

### Monthly (Manual)
- Review CI/CD performance
- Update workflow configurations
- Rotate security tokens if needed
- Review and update documentation

### Quarterly (Manual)
- Major dependency updates
- Workflow optimization
- Security audit
- Performance baseline review
