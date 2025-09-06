# Setup-Lint Regression Tests Documentation

This document explains the comprehensive regression tests created to prevent the critical bugs that were discovered in the `antimoji setup-lint` command from recurring.

## Background

The original bugs in `setup-lint` included:
1. **YAML Indentation Bug**: Generated invalid YAML with excessive indentation
2. **Invalid Command Flags**: Used non-existent `--profile` and `--fail-on-found` flags
3. **Binary Detection Issues**: Incorrectly assumed local build vs global installation
4. **Missing Multi-Document YAML Support**: Didn't handle Kubernetes YAML files
5. **Go Module Context Issues**: Included Go hooks inappropriately
6. **No Configuration Validation**: Generated configs weren't validated

## Test Categories

### 1. YAML Syntax Validation Tests (`TestYAMLSyntaxValidation`)

**What it tests:**
- Generated YAML is syntactically correct and parseable
- No excessive indentation (specifically catches the 16-space vs 8-space bug)
- YAML structure is valid across all linting modes
- Works correctly with and without go.mod files

**Why it's important:**
This test would have immediately caught the critical YAML indentation bug that caused "mapping values are not allowed in this context" errors. The original bug generated YAML with 16 spaces of indentation on the `entry:` field instead of 8 spaces.

**Test scenarios:**
- Zero-tolerance mode with/without go.mod
- Allow-list mode with/without go.mod  
- Permissive mode with/without go.mod

### 2. Command Flag Validation Tests (`TestCommandFlagValidation`)

**What it tests:**
- Only valid antimoji scan flags are used in generated configs
- Invalid flags like `--profile` and `--fail-on-found` are not present
- Generated commands can actually be executed
- Flag validation works across all linting modes

**Why it's important:**
This test prevents the use of non-existent command flags that would cause pre-commit hooks to fail with "unknown flag" errors. It dynamically validates flags against the actual antimoji binary.

**Test scenarios:**
- Validates flags for zero-tolerance, allow-list, and permissive modes
- Checks against actual antimoji help output when available
- Explicitly blocks known invalid flags

### 3. Binary Detection Logic Tests (`TestBinaryDetectionLogic`)

**What it tests:**
- Correct detection of globally installed vs local antimoji binary
- Proper fallback behavior when global binary isn't available
- Makefile-based local build detection
- PATH environment variable handling

**Why it's important:**
This test ensures the tool correctly detects how to invoke antimoji, preventing "Executable `bin/antimoji` not found" errors. It validates the smart detection logic that prefers global installation but falls back to local builds when appropriate.

**Test scenarios:**
- Global antimoji available in PATH
- No global antimoji but Makefile exists (local build)
- Neither global nor local build available (fallback behavior)

### 4. Multi-Document YAML Support Tests (`TestMultiDocumentYAMLSupport`)

**What it tests:**
- `--allow-multiple-documents` flag is included in check-yaml hook
- Multi-document YAML files (common in Kubernetes) are supported
- Flag is present across all linting modes
- Proper hook configuration structure

**Why it's important:**
This test prevents the "expected a single document in the stream" errors that occur with Kubernetes YAML files using `---` separators. Modern DevOps workflows commonly use multi-document YAML files.

**Test scenarios:**
- Creates actual multi-document YAML files
- Validates flag presence in generated configs
- Tests across all linting modes

### 5. Go Module Detection Tests (`TestGoModuleDetection`)

**What it tests:**
- Go-specific hooks are only included when go.mod exists
- No Go hooks when go.mod is absent
- Proper detection of Go project context
- Conditional hook inclusion logic

**Why it's important:**
This test prevents "cannot find main module" errors that occur when Go hooks are included in non-Go projects. It ensures the tool is context-aware and only includes relevant hooks.

**Test scenarios:**
- Project with go.mod file (should include Go hooks)
- Project without go.mod file (should not include Go hooks)
- Validates specific Go hook presence/absence

### 6. Build Hook Conditional Inclusion Tests (`TestBuildHookConditionalInclusion`)

**What it tests:**
- Build hooks are only included when local builds are needed
- No unnecessary build hooks when using global antimoji
- Proper environment-based decision making
- Makefile and PATH interaction

**Why it's important:**
This test prevents unnecessary build hooks that can fail when no Makefile exists, and ensures efficient configuration generation based on the actual deployment scenario.

**Test scenarios:**
- Global antimoji available (no build hook needed)
- Local build required (build hook included)
- Environment manipulation to test different scenarios

### 7. Configuration Validation Tests (`TestConfigurationValidation`)

**What it tests:**
- The `validateConfiguration()` function correctly identifies issues
- YAML syntax errors are caught
- Invalid flags are detected in generated configs
- Validation logic works correctly

**Why it's important:**
This test validates the validation layer itself, ensuring that the safety net catches configuration issues before they cause runtime failures.

**Test scenarios:**
- Valid YAML configurations (should pass)
- Invalid YAML syntax (should fail)
- Configurations with invalid flags (should fail)
- Edge cases and error conditions

### 8. End-to-End Regression Prevention Tests (`TestEndToEndRegressionPrevention`)

**What it tests:**
- Complete setup-lint workflow from start to finish
- Realistic project environment simulation
- All bug fixes working together
- Generated files are actually usable

**Why it's important:**
This comprehensive test simulates the exact conditions that caused the original bugs, ensuring they can never regress. It validates the entire system working together, not just individual components.

**Test scenarios:**
- Creates realistic project with go.mod and Kubernetes YAML
- Runs complete setup-lint workflow
- Validates all generated files
- Checks all specific bug fixes are working

## Test Design Principles

### 1. **Fail-Fast Detection**
Each test is designed to fail immediately if the corresponding bug is reintroduced, making it impossible for these issues to slip through CI/CD.

### 2. **Environment Independence**
Tests create their own isolated environments and don't depend on external tools being installed, making them reliable across different development environments.

### 3. **Realistic Scenarios**
Tests use realistic project structures and configurations that mirror actual user environments where the bugs occurred.

### 4. **Comprehensive Coverage**
Each major bug category has dedicated tests, and the end-to-end test validates all fixes working together.

### 5. **Self-Documenting**
Each test includes detailed comments explaining what bug it prevents and why it's necessary.

## Running the Tests

```bash
# Run all regression tests
go test ./internal/cli -v -run "Regression|YAML|Flag|Binary|MultiDocument|GoModule|BuildHook|Configuration"

# Run specific test category
go test ./internal/cli -v -run TestYAMLSyntaxValidation
go test ./internal/cli -v -run TestCommandFlagValidation
go test ./internal/cli -v -run TestEndToEndRegressionPrevention

# Run all CLI tests (includes regression tests)
go test ./internal/cli -v
```

## Maintenance

These tests should be:
1. **Never removed** - They prevent critical regressions
2. **Updated carefully** - Changes should only make tests more comprehensive
3. **Extended when new bugs are found** - Add new test cases for any new issues
4. **Kept in sync** - Update if the setup-lint functionality legitimately changes

## Integration with CI/CD

These regression tests are automatically run as part of the standard test suite, ensuring that any code changes that would reintroduce these bugs will be caught immediately in the CI/CD pipeline.

The tests are designed to be fast and reliable, with minimal external dependencies, making them suitable for frequent execution in automated testing environments.
