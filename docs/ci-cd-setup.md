# CI/CD Setup Guide

This document outlines the complete CI/CD setup for the Antimoji project, including required secrets, workflow configurations, and release processes.

## Overview

The Antimoji project uses GitHub Actions for CI/CD with the following workflows:

- **CI (`ci.yml`)** - Continuous integration for every push/PR
- **Release (`release.yml`)** - Automated releases when tags are pushed
- **Build (`build.yml`)** - Development builds and snapshots
- **Maintenance (`maintenance.yml`)** - Weekly dependency and security checks
- **Nightly (`nightly.yml`)** - Comprehensive nightly testing

## Required GitHub Secrets

To fully utilize all CI/CD features, configure these secrets in your repository settings:

### Essential Secrets

1. **`GITHUB_TOKEN`** (Automatic)
   - Automatically provided by GitHub
   - Used for: Creating releases, uploading artifacts, Docker registry access

### Optional Secrets (for enhanced features)

2. **`CODECOV_TOKEN`** (Optional)
   - For: Code coverage reporting to Codecov
   - Get from: https://codecov.io/
   - Used in: `ci.yml` workflow

3. **`HOMEBREW_TOKEN`** (Optional)
   - For: Automated Homebrew formula updates
   - Requirements: Personal access token with repo permissions to your homebrew-tap repository
   - Used in: `release.yml` workflow
   - Note: Workflow continues even if this fails

## Workflow Triggers

### CI Workflow (`ci.yml`)
- **Triggers**: Push to `main`/`develop`, PRs to `main`/`develop`, manual dispatch
- **Jobs**: Test, Lint, Security scan, Benchmark (PR only), Build, Cross-platform test, Quality gate
- **Artifacts**: Build artifacts (30 days), Benchmark results

### Release Workflow (`release.yml`)
- **Triggers**: Tag push (pattern: `v*`)
- **Jobs**: Test, GoReleaser release, Docker image build, Homebrew update, Notifications
- **Artifacts**: Release binaries, Docker images, Homebrew formula update

### Build Workflow (`build.yml`)
- **Triggers**: Push to `main`/`develop`, PRs to `main`
- **Jobs**: Snapshot builds, Multi-platform testing, Performance testing
- **Artifacts**: Development snapshots (7-14 days)

### Maintenance Workflow (`maintenance.yml`)
- **Triggers**: Weekly schedule (Sundays 3 AM UTC), manual dispatch
- **Jobs**: Dependency updates, Vulnerability scanning, Code quality audit

### Nightly Workflow (`nightly.yml`)
- **Triggers**: Daily schedule (2 AM UTC), manual dispatch
- **Jobs**: Multi-platform comprehensive testing, Performance regression testing, Security scanning, Integration testing

## Release Process

### 1. Prepare Release
```bash
# Ensure all tests pass
make check-all

# Create and test local release
make release-local

# Prepare release with version
make release-prepare VERSION=v1.0.0
```

### 2. Push Release Tag
```bash
# Push the tag to trigger release workflow
git push origin v1.0.0
```

### 3. Monitor Release
- Watch the GitHub Actions workflow
- Check that all jobs complete successfully
- Verify release artifacts are created
- Confirm Docker images are published
- Check Homebrew formula update (if configured)

## Build Targets

The project includes comprehensive Makefile targets:

### Development
- `make build` - Build development binary
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage
- `make lint` - Run linter
- `make check-all` - Run all quality checks

### Release
- `make release-prepare VERSION=v1.0.0` - Prepare release
- `make release-check` - Validate release readiness
- `make release-local` - Test release locally
- `make release-snapshot` - Create snapshot release

### CI/CD Helpers
- `make ci-test` - CI test target
- `make ci-lint` - CI lint target
- `make ci-security` - CI security scan
- `make ci-build-all` - Build all platform binaries

## Quality Gates

### CI Quality Requirements
- All tests must pass
- Code coverage ≥ 85%
- Linting must pass (golangci-lint)
- Security scan must pass (gosec)
- Cross-platform builds must succeed

### Release Quality Requirements
- All CI quality gates must pass
- Integration tests must pass
- Performance benchmarks must complete
- Multi-platform binaries must be buildable

## Artifacts and Distribution

### GitHub Releases
- Multi-platform binaries (Linux, macOS, Windows)
- Archive files (.tar.gz for Unix, .zip for Windows)
- Checksums file
- Automated changelog generation

### Docker Images
- Published to GitHub Container Registry (`ghcr.io`)
- Multi-architecture support (amd64)
- Semantic version tagging
- Latest tag for newest release

### Package Managers
- **Homebrew**: Automated formula updates (requires `HOMEBREW_TOKEN`)
- **Go Install**: Direct installation from GitHub

## Monitoring and Maintenance

### Automated Checks
- **Weekly**: Dependency updates, vulnerability scans
- **Nightly**: Comprehensive testing, performance regression testing
- **Per-commit**: Full CI pipeline

### Manual Operations
- Release preparation and tagging
- Homebrew tap repository management
- Security token rotation

## Troubleshooting

### Common Issues

1. **Coverage Upload Fails**
   - Ensure `CODECOV_TOKEN` is set (or remove codecov step)
   - Check coverage file is generated correctly

2. **Homebrew Update Fails**
   - Verify `HOMEBREW_TOKEN` has correct permissions
   - Check homebrew-tap repository exists
   - Workflow continues even if this fails

3. **Docker Push Fails**
   - Ensure GitHub Container Registry permissions are correct
   - Check `GITHUB_TOKEN` has package write permissions

4. **Release Fails**
   - Verify tag follows semantic versioning (`v1.0.0`)
   - Ensure all CI checks pass before tagging
   - Check GoReleaser configuration syntax

### Testing Workflows Locally

```bash
# Test GoReleaser configuration
goreleaser check

# Create local snapshot
goreleaser build --snapshot --clean

# Test all quality checks
make check-all

# Test release preparation
make release-check
```

## Security Considerations

- All secrets are properly scoped to required workflows
- Docker images use minimal scratch base for security
- Security scanning runs on every CI build
- Vulnerability checking runs weekly
- SARIF reports uploaded to GitHub Security tab

## Performance Monitoring

- Benchmark tests run on every PR
- Nightly performance regression testing
- Build time and artifact size tracking
- Multi-platform performance validation
