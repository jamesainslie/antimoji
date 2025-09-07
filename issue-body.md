## Summary

The pre-commit integration for Antimoji had a critical bug where the clean step would report "0 modified" while the lint/scan step would still find emojis, causing zero-tolerance enforcement to fail.

## Problem Description

Users reported that pre-commit hooks configured with Antimoji would show:
```
Auto-clean Emojis (zero-tolerance).................Failed
- hook id: antimoji-clean
- files were modified by this hook
Summary: removed 0 emojis from 157 files (0 modified, 0 errors)

Zero-Tolerance Emoji Verification..................Failed
- hook id: antimoji-verify
- exit code: 1
Error: emoji threshold exceeded: found 106 emojis (limit: 0)
```

## Root Causes Identified

1. **Missing `--ignore-allowlist` flag in clean command** - Only available in scan
2. **Profile inconsistency** - Clean using `ci-lint` profile, scan using `zero` profile  
3. **Different allowlist handling logic** between clean and scan commands
4. **Inconsistent flag naming** - `--respect-allowlist` vs `--ignore-allowlist`

## Solution Implemented

### 1. Standardized Allowlist Flags
- Added `--ignore-allowlist` flag to clean command
- Made flag behavior consistent across both commands
- Maintained backward compatibility

### 2. Unified Allowlist Processing
- Created `internal/core/allowlist/processor.go` with shared logic
- Both commands now use identical allowlist processing
- Consistent error handling and logging

### 3. Enhanced Pre-commit Examples
- Added `examples/pre-commit/` with working configurations
- Created comprehensive troubleshooting guide
- Provided clear migration paths

### 4. Updated Setup-Lint Command
- Now generates improved two-step workflow (clean + verify)
- Uses consistent profiles between steps
- Uses `entry: antimoji` with `args:` pattern instead of problematic `bash -c`

## Files Changed

### Core Implementation
- `internal/cli/clean.go` - Added `--ignore-allowlist` flag and unified processing
- `internal/cli/scan.go` - Updated to use unified allowlist processing  
- `internal/core/allowlist/processor.go` - New unified allowlist processing logic
- `internal/cli/setup_lint.go` - Updated to generate improved configurations

### Testing
- `internal/cli/clean_precommit_test.go` - Comprehensive integration tests

### Documentation and Examples
- `examples/pre-commit/zero-tolerance.yaml` - Zero-tolerance configuration
- `examples/pre-commit/ci-friendly.yaml` - CI-friendly configuration
- `examples/pre-commit/README.md` - Comprehensive guide
- `PRECOMMIT_REFACTOR_SUMMARY.md` - Detailed technical summary

## Validation Results

✅ Fixed `--ignore-allowlist` flag availability in clean command
✅ Fixed profile inconsistency issues with unified allowlist processing  
✅ Argument forwarding works correctly
✅ Both `bash -c` and direct invocation patterns work
✅ Comprehensive solution addresses all root causes
✅ All existing functionality preserved (backward compatible)

## Recommended Configuration

```yaml
repos:
  - repo: local
    hooks:
      - id: antimoji-clean
        entry: antimoji
        args: [clean, --config=.antimoji.yaml, --profile=zero, --in-place, --quiet]
        language: system
        types: [text]
        pass_filenames: true

      - id: antimoji-verify  
        entry: antimoji
        args: [scan, --config=.antimoji.yaml, --profile=zero, --threshold=0, --quiet]
        language: system
        types: [text]
        pass_filenames: true
```

This fix ensures reliable zero-tolerance emoji enforcement in pre-commit workflows.
