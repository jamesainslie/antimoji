# GitHub Repository Setup

This guide walks you through setting up your GitHub repository to work with the Antimoji CI/CD system.

## 1. Create GitHub Repository

1. Go to GitHub and create a new repository named `antimoji`
2. Make it public (for GitHub Actions and package registry)
3. Don't initialize with README (since you already have one)

## 2. Configure Local Git Remote

```bash
# Add GitHub remote
git remote add origin https://github.com/YOUR_USERNAME/antimoji.git

# Or if you prefer SSH
git remote add origin git@github.com:YOUR_USERNAME/antimoji.git

# Push your code
git push -u origin main
```

## 3. Repository Settings

### Actions Permissions
Go to Settings → Actions → General and ensure:
- **Actions permissions**: "Allow all actions and reusable workflows"
- **Workflow permissions**: "Read and write permissions"
- **Allow GitHub Actions to create and approve pull requests**: Checked

### Package Registry
Go to Settings → Actions → General:
- Under "Workflow permissions", ensure "Read and write permissions" is selected
- This enables publishing to GitHub Container Registry (ghcr.io)

## 4. Required Secrets (Optional)

Go to Settings → Secrets and variables → Actions:

### For Code Coverage (Optional)
- **Secret name**: `CODECOV_TOKEN`
- **Value**: Get from https://codecov.io after connecting your repository
- **Purpose**: Upload test coverage reports

### For Homebrew Updates (Optional)
- **Secret name**: `HOMEBREW_TOKEN`
- **Value**: GitHub Personal Access Token with repo permissions
- **Purpose**: Automatically update Homebrew formula on releases
- **Setup**:
  1. Create a homebrew-tap repository: `https://github.com/YOUR_USERNAME/homebrew-tap`
  2. Generate a Personal Access Token with `repo` scope
  3. Add the token as `HOMEBREW_TOKEN` secret

## 5. Branch Protection (Recommended)

Go to Settings → Branches and add protection rules for `main`:

- **Require a pull request before merging**
- **Require status checks to pass before merging**
  - Add required checks: `Test`, `Lint`, `Security Scan`
- **Require branches to be up to date before merging**
- **Restrict pushes that create files over 100MB**

## 6. Test Your Setup

### Initial Push
```bash
# Push your code
git push origin main
```

This should trigger the CI workflow. Check:
- Go to Actions tab in your repository
- Verify the CI workflow runs
- Check that all jobs complete successfully

### Test Release Process
```bash
# Create a test release
git tag v0.1.0-test
git push origin v0.1.0-test
```

This should trigger the release workflow. Verify:
- Release workflow completes
- GitHub release is created
- Binaries are uploaded
- Docker image is published (if Docker daemon available)

## 7. Repository Configuration

### Topics/Tags
Add these topics to your repository for discoverability:
- `cli`
- `emoji`
- `go`
- `performance`
- `text-processing`

### Description
"High-performance emoji detection and removal CLI tool"

### Website
"https://github.com/antimoji/antimoji"

## 8. Verify Everything Works

After setup, verify these work:

### CI Workflow
- [ ] Pushes to main trigger CI
- [ ] PRs trigger CI with benchmarks
- [ ] All quality gates pass
- [ ] Artifacts are uploaded

### Release Workflow  
- [ ] Tag pushes trigger releases
- [ ] Binaries are built for all platforms
- [ ] GitHub release is created
- [ ] Docker images are published
- [ ] Homebrew formula updates (if configured)

### Scheduled Workflows
- [ ] Maintenance workflow runs weekly
- [ ] Nightly workflow runs daily
- [ ] Build workflow runs on pushes

## 9. Maintenance

### Regular Tasks
- Monitor workflow runs for failures
- Update dependencies when maintenance workflow reports updates
- Review security scan results
- Check performance regression reports

### Token Rotation
- Rotate `HOMEBREW_TOKEN` annually
- Update `CODECOV_TOKEN` if needed
- Review repository access permissions quarterly

## Troubleshooting

### Common Issues

1. **Workflows don't trigger**
   - Check Actions permissions are enabled
   - Verify workflow files are in `.github/workflows/`
   - Check YAML syntax is valid

2. **Docker builds fail**
   - Ensure Docker daemon is available in CI
   - Check network connectivity issues
   - Verify base image availability

3. **Release fails**
   - Check tag format follows semantic versioning
   - Verify all CI checks pass first
   - Check GoReleaser configuration syntax

4. **Homebrew update fails**
   - Verify homebrew-tap repository exists
   - Check `HOMEBREW_TOKEN` permissions
   - Ensure token has repo scope

### Getting Help
- Check workflow logs in GitHub Actions tab
- Review error messages in job outputs
- Consult the CI/CD setup documentation
- Check GoReleaser documentation for configuration issues
