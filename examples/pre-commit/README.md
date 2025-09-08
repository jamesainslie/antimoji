# Antimoji Pre-commit Integration Examples

This directory contains working examples of pre-commit configurations for Antimoji that avoid common pitfalls and ensure reliable zero-tolerance emoji enforcement.

## Quick Start

1. Choose the appropriate configuration for your needs:
   - `zero-tolerance.yaml` - Strict no-emoji policy
   - `ci-friendly.yaml` - Allows certain emojis in documentation

2. Copy the relevant configuration to your `.pre-commit-config.yaml`

3. Create the corresponding `.antimoji.yaml` configuration file (examples included in each file)

4. Install and run:
   ```bash
   pre-commit install
   pre-commit run --all-files
   ```

## Configuration Explanations

### zero-tolerance.yaml

**Use case**: Projects that want absolutely no emojis anywhere.

**Key features**:
- Uses `zero` profile with empty allowlist
- Both clean and verify steps use the same profile (consistent behavior)
- Uses `entry: antimoji` with `args:` for reliable argument passing
- Comprehensive exclude patterns for common directories

**Why it works**:
- Consistent profile usage eliminates the "0 modified but still finds emojis" issue
- Direct `entry: antimoji` avoids bash shell argument forwarding complexity
- Zero-tolerance profile ensures clean removes everything that verify would find

### ci-friendly.yaml

**Use case**: Projects that allow certain emojis in documentation but want strict enforcement in code.

**Key features**:
- Uses `ci-lint` profile with carefully curated allowlist
- Both clean and verify steps use the same profile (consistent behavior)
- Allows common CI/documentation emojis (✅, ❌, ⚠️, ✨)
- File-specific ignore patterns for test files and documentation

**Why it works**:
- Consistent profile usage ensures clean and verify have the same understanding of "allowed"
- Allowlist is carefully designed for common CI/documentation use cases
- File ignore patterns prevent enforcement in appropriate contexts

## Common Issues and Solutions

### Issue 1: "clean reports 0 modified, verify still finds emojis"

**Root cause**: Different profiles or allowlist settings between clean and verify steps.

**Solution**: Use the same profile and flags for both steps.

```yaml
# ❌ Wrong - inconsistent profiles
- args: [clean, --profile=ci-lint, --in-place]
- args: [scan, --profile=zero, --threshold=0]

# ✅ Correct - consistent profiles
- args: [clean, --profile=zero, --in-place]
- args: [scan, --profile=zero, --threshold=0]
```

### Issue 2: "unknown flag: --ignore-allowlist" on clean command

**Root cause**: Using flags that only exist on scan command.

**Solution**: Use consistent flags that exist on both commands, or use different flag approaches.

```yaml
# ❌ Wrong - --ignore-allowlist only exists on scan
- args: [clean, --ignore-allowlist, --in-place]

# ✅ Correct - use zero profile instead
- args: [clean, --profile=zero, --in-place]

# ✅ Also correct - now --ignore-allowlist works on clean too (after fix)
- args: [clean, --ignore-allowlist, --in-place]
```

### Issue 3: Bash argument forwarding issues

**Root cause**: Complex shell escaping with `bash -c` pattern.

**Solution**: Use direct `entry: antimoji` with `args:` instead.

```yaml
# ❌ Complex and error-prone
entry: bash -c 'antimoji clean --config=.antimoji.yaml "$@"'

# ✅ Simple and reliable
entry: antimoji
args: [clean, --config=.antimoji.yaml]
```

## Testing Your Configuration

Use this test script to validate your pre-commit setup:

```bash
#!/bin/bash
# Create test file with emojis
echo "Test file with emojis 😀 🎉 ✅" > test_emoji.txt

# Run pre-commit on the test file
pre-commit run --files test_emoji.txt

# Check results
if [ $? -eq 0 ]; then
    echo "✅ Pre-commit configuration working correctly"
    echo "File content after processing: $(cat test_emoji.txt)"
else
    echo "❌ Pre-commit configuration has issues"
fi

# Cleanup
rm -f test_emoji.txt
```

## Advanced Configuration

### Custom Profiles

You can create custom profiles in your `.antimoji.yaml`:

```yaml
version: 0.5.0
profiles:
  strict-code:
    emoji_allowlist: []
    file_ignore_list: ["README.md", "CHANGELOG.md"]
    description: "No emojis in code, but allow in docs"
    
  docs-only:
    emoji_allowlist: ["✅", "❌", "⚠️", "📝", "🔧"]
    file_ignore_list: ["**/*.go", "**/*.js", "**/*.py"]
    description: "Allow emojis only in documentation"
```

### Multiple Hook Strategy

For complex projects, you might want different rules for different file types:

```yaml
repos:
  - repo: local
    hooks:
      # Strict enforcement for code files
      - id: antimoji-code
        name: "Clean Code Files (zero-tolerance)"
        entry: antimoji
        args: [clean, --profile=strict-code, --in-place, --quiet]
        files: \.(go|js|py|java|cpp|c|h)$
        
      # Lenient enforcement for documentation
      - id: antimoji-docs
        name: "Clean Documentation (allow some emojis)"
        entry: antimoji
        args: [clean, --profile=docs-friendly, --in-place, --quiet]
        files: \.(md|rst|txt)$
```

## Troubleshooting

### Enable Debug Output

Add logging flags to troubleshoot issues:

```yaml
# Basic diagnostic information
args: [clean, --config=.antimoji.yaml, --profile=zero, --log-level=info, --in-place]

# Detailed debugging with Unicode code points
args: [clean, --config=.antimoji.yaml, --profile=zero, --log-level=debug, --verbose, --in-place]

# Human-readable text logs instead of JSON
args: [clean, --config=.antimoji.yaml, --profile=zero, --log-level=info, --log-format=text, --in-place]
```

### Check Configuration

Validate your antimoji configuration:

```bash
antimoji scan --config=.antimoji.yaml --profile=zero --dry-run .
```

### Common File Patterns

Make sure your exclude patterns match your project structure:

```yaml
exclude: |
  (?x)^(
    # Version control
    \.git/.*|
    \.svn/.*|
    
    # Dependencies
    vendor/.*|
    node_modules/.*|
    venv/.*|
    \.venv/.*|
    
    # Build artifacts
    bin/.*|
    dist/.*|
    build/.*|
    target/.*|
    
    # Configuration
    \.antimoji\.yaml|
    \.pre-commit-config\.yaml
  )$
```

## Support

If you encounter issues:

1. Check this README for common solutions
2. Test your configuration with the provided test script
3. Enable debug logging to see what's happening
4. Create a minimal reproduction case
5. File an issue with the Antimoji project

## Contributing

Found a configuration pattern that works well? Please contribute it back to help other users!

