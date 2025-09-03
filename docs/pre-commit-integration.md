# Pre-commit Integration Guide

This guide explains how to integrate Antimoji with [pre-commit](https://pre-commit.com/) for automated emoji linting in your development workflow.

## Overview

Antimoji provides several pre-commit hooks that can automatically check your code for unwanted emojis before commits are made. This ensures your codebase maintains a professional appearance while allowing emojis in appropriate contexts like tests and documentation.

## Quick Setup

### For Antimoji Development (Local)

1. **Install pre-commit** (if not already installed):
   ```bash
   pip install pre-commit
   # or
   brew install pre-commit
   ```

2. **Copy the example configuration**:
   ```bash
   cp .pre-commit-config.yaml.example .pre-commit-config.yaml
   ```

3. **Install the hooks**:
   ```bash
   pre-commit install
   ```

4. **Generate antimoji configuration**:
   ```bash
   make generate-allowlist
   ```

### For External Projects

Add this to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0  # Use the latest release
    hooks:
      - id: antimoji-lint
        files: \.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$
        exclude: .*_test\.|.*/test/.*
```

## Available Hooks

### 1. `antimoji-lint` (Strict)
**Purpose**: Fail if any emojis are found in source code files
**Usage**: Best for enforcing strict emoji-free policies

```yaml
- id: antimoji-lint
  files: \.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$
  exclude: .*_test\.|.*/test/.*
```

**Behavior**:
- Scans source code files only
- Ignores test files, documentation, and vendor directories
- Fails (exit code 1) if any emojis are found
- Uses `--threshold=0 --ignore-allowlist` for strict checking

### 2. `antimoji-lint-config` (Configuration-based)
**Purpose**: Use custom antimoji configuration for linting
**Usage**: Best when you have specific emoji allowlists

```yaml
- id: antimoji-lint-config
  files: \.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$
```

**Behavior**:
- Requires `.antimoji.yaml` configuration file
- Respects your custom allowlists and ignore patterns
- Uses the `ci-lint` profile by default

### 3. `antimoji-docs` (Documentation)
**Purpose**: Check documentation for excessive emoji usage
**Usage**: Allow some emojis in docs but prevent overuse

```yaml
- id: antimoji-docs
  files: \.(md|rst|txt)$
```

**Behavior**:
- More permissive threshold (allows up to 5 emojis per file)
- Focuses on documentation files
- Helps maintain professional documentation

### 4. `antimoji-clean` (Auto-fix)
**Purpose**: Automatically remove emojis from source files
**Usage**: Auto-fix commits by removing emojis

```yaml
- id: antimoji-clean
  files: \.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$
  exclude: .*_test\.|.*/test/.*
```

**Behavior**:
- Automatically removes emojis from source files
- Creates backup files before modification
- Modifies files in-place

## Configuration Examples

### Strict Policy (No Emojis)
```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0
    hooks:
      - id: antimoji-lint
        files: \.(go|js|ts|py|java|c|cpp|rs)$
        exclude: .*_test\.|.*/test/.*
```

### Balanced Policy (Allow in Tests/Docs)
```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0
    hooks:
      - id: antimoji-lint-config  # Uses .antimoji.yaml
        files: \.(go|js|ts|py|java|c|cpp|rs)$
      - id: antimoji-docs
        files: \.(md|rst)$
```

### Auto-fix Policy (Remove Emojis Automatically)
```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0
    hooks:
      - id: antimoji-clean
        files: \.(go|js|ts|py|java|c|cpp|rs)$
        exclude: .*_test\.|.*/test/.*
```

## Setting Up Custom Configuration

### 1. Generate Configuration
```bash
# For strict CI linting
antimoji generate --type=ci-lint --output=.antimoji.yaml .

# For development-friendly config
antimoji generate --type=dev --output=.antimoji.yaml .

# For test-only emojis
antimoji generate --type=test-only --output=.antimoji.yaml .
```

### 2. Customize Configuration
Edit `.antimoji.yaml` to adjust:
- `emoji_allowlist`: Emojis to allow
- `file_ignore_list`: Files to skip
- `directory_ignore_list`: Directories to skip
- `max_emoji_threshold`: Maximum allowed emojis

### 3. Test Configuration
```bash
# Test the configuration
antimoji scan --config=.antimoji.yaml --profile=ci-lint --threshold=0 .

# Test pre-commit hook
pre-commit run antimoji-lint-config --all-files
```

## File Patterns

### Included Files (Source Code)
- `*.go`, `*.js`, `*.ts`, `*.jsx`, `*.tsx`
- `*.py`, `*.rb`, `*.java`
- `*.c`, `*.cpp`, `*.h`, `*.hpp`
- `*.rs`, `*.php`, `*.swift`, `*.kt`, `*.scala`

### Excluded Files (Automatically)
- Test files: `*_test.*`, `*/test/*`, `*/tests/*`
- Test data: `*/testdata/*`, `*/fixtures/*`, `*/mocks/*`
- Dependencies: `vendor/*`, `node_modules/*`
- Build artifacts: `dist/*`, `bin/*`, `build/*`
- Version control: `.git/*`

### Documentation Files
- `*.md`, `*.rst`, `*.txt`
- Excludes: `vendor/*`, `node_modules/*`, build artifacts

## Troubleshooting

### Hook Fails to Run
**Problem**: `antimoji: command not found`

**Solutions**:
1. **Local repo**: Use `repo: local` and ensure antimoji is built:
   ```bash
   make build
   ```

2. **External repo**: Ensure you're using the correct repository URL and version:
   ```yaml
   repo: https://github.com/jamesainslie/antimoji
   rev: v0.9.0  # Use latest release tag
   ```

### Too Many False Positives
**Problem**: Hook fails on legitimate emojis

**Solutions**:
1. **Generate allowlist**:
   ```bash
   antimoji generate --type=ci-lint --output=.antimoji.yaml .
   ```

2. **Use config-based hook**:
   ```yaml
   - id: antimoji-lint-config  # Uses .antimoji.yaml
   ```

3. **Adjust threshold**:
   ```yaml
   - id: antimoji-lint
     args: [--threshold=5]  # Allow up to 5 emojis
   ```

### Performance Issues
**Problem**: Hook is slow on large repositories

**Solutions**:
1. **Limit file patterns**:
   ```yaml
   files: \.go$  # Only Go files
   ```

2. **Add more excludes**:
   ```yaml
   exclude: |
     (?x)^(
       vendor/.*|
       .*_generated\..*|
       .*\.pb\.go
     )$
   ```

3. **Use parallel processing**:
   ```yaml
   require_serial: false
   ```

## Integration with Antimoji Development

For the Antimoji project itself, the setup includes:

### Makefile Targets
```bash
make install-pre-commit-hook    # Install git pre-commit hook
make uninstall-pre-commit-hook  # Remove git pre-commit hook
make test-pre-commit-hook       # Test pre-commit hook
make generate-allowlist         # Generate CI allowlist
make antimoji-lint             # Run antimoji on itself
```

### CI Integration
The GitHub Actions workflow includes an `antimoji-lint` job that:
1. Builds antimoji from source
2. Runs antimoji on its own codebase
3. Fails the build if emojis are found in restricted locations
4. Uses the generated `.antimoji.yaml` configuration

### Development Workflow
```bash
# Set up development environment with pre-commit
make dev-setup

# Generate strict allowlist
make generate-allowlist

# Test the linting
make antimoji-lint

# Run pre-commit on all files
pre-commit run --all-files
```

## Best Practices

### 1. Configuration Management
- **Generate initial config**: Use `antimoji generate` to create baseline
- **Version control config**: Commit `.antimoji.yaml` to repository
- **Regular updates**: Regenerate config when adding legitimate emojis

### 2. Hook Selection
- **Strict projects**: Use `antimoji-lint` for zero-emoji policy
- **Balanced projects**: Use `antimoji-lint-config` with custom allowlists
- **Documentation-heavy**: Add `antimoji-docs` for documentation checking

### 3. Performance Optimization
- **File patterns**: Be specific about which files to check
- **Exclusions**: Exclude vendor, test, and generated files
- **Parallel execution**: Use `require_serial: false` for better performance

### 4. Team Adoption
- **Gradual rollout**: Start with `antimoji-docs` then add `antimoji-lint`
- **Clear guidelines**: Document which emojis are acceptable where
- **Auto-fix option**: Consider `antimoji-clean` for automatic cleanup

## Example Configurations

### Minimal Setup (Zero Emojis)
```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0
    hooks:
      - id: antimoji-lint
```

### Comprehensive Setup
```yaml
repos:
  - repo: https://github.com/jamesainslie/antimoji
    rev: v0.9.0
    hooks:
      - id: antimoji-lint-config  # Source code with allowlist
      - id: antimoji-docs         # Documentation checking
      - id: antimoji-clean        # Auto-fix (optional)
```

### Custom Local Setup
```yaml
repos:
  - repo: local
    hooks:
      - id: antimoji-custom
        name: Custom Antimoji Check
        entry: bin/antimoji scan --config=custom-config.yaml --threshold=0
        language: system
        files: \.go$
        pass_filenames: true
```

This integration provides a seamless way to maintain emoji-free codebases while preserving flexibility for appropriate emoji usage in tests and documentation.
