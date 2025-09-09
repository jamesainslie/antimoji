# Technical Debt

This document tracks technical debt, stubbed functionality, and items that need revisiting in the Antimoji codebase.

## 🔄 Refactoring in Progress (2025-09-09)

### PR #8: Dependency Injection Foundation ✅ ALL CHECKS PASSING
- **GitHub PR**: https://github.com/jamesainslie/antimoji/pull/8
- **Status**: ✅ 14/17 checks successful, 3 skipped, 0 failing
- **Fixes Applied**:
  - ✅ Fixed --config and --profile flag handling in scan command
  - ✅ Added comprehensive tests for commands package (scan_test.go)
  - ✅ Test coverage increased from 62.5% to 65.3% (above required 65%)
  - ✅ CI/Antimoji Lint: PASS (was failing)
  - ✅ CI/Quality Gate: PASS (was failing)
- **Achievements**: DI foundation + scan command refactored + CI fixes
- **Ready**: Ready for review and merge

### PR #9: Remaining Commands Refactoring 🚧 PLANNED
- **Branch**: `refactor/remaining-commands` (will be created after PR #8 fixes)
- **Scope**: Clean, Generate, Setup-lint commands

### Global State Elimination

#### ✅ Completed
- Created dependency injection foundation (`internal/app/`)
- Implemented mock logger for testing
- Updated main.go to use dependency injection
- Created Application and Dependencies structs

#### 🚧 In Progress - Global Logger Removal
**Priority: HIGH**
- **Issue**: Global logger still exists in `internal/observability/logging/global.go`
- **Status**: ✅ Scan command refactored, other commands pending
- **Progress**:
  - ✅ Created `internal/app/commands/scan.go` with full DI
  - ✅ Scan command working correctly with dependency injection
  - 🚧 Clean, Generate, Setup-lint commands still need refactoring
- **Next Steps**: 
  1. ✅ ~~Refactor scan command~~ 
  2. 🚧 Refactor clean, generate, setup-lint commands
  3. Remove global logger functions and variables
  4. Update all import statements
- **Files Affected**: 
  - `internal/observability/logging/global.go` (to be removed/refactored)
  - `internal/cli/clean.go`, `internal/cli/generate.go` (need refactoring)
  - Various other files using global logging calls
- **Validation**: `grep -r "globalLogger" internal/` should return no results

#### 🚧 Pending - Global UI Output Removal  
**Priority: HIGH**
- **Issue**: Global UI output singleton still exists in `internal/ui/output.go`
- **Status**: Not started
- **Dependencies**: Should be done after global logger removal
- **Files Affected**:
  - `internal/ui/output.go` (lines 189-230 contain global state)
  - All files using `ui.Info()`, `ui.Success()`, etc.

### Command Structure Refactoring

#### 🚧 Stubbed Commands Need Implementation
**Priority: MEDIUM**
- **Issue**: Most CLI commands are currently placeholders that return errors
- **Location**: `internal/app/application.go` lines 95-145
- **Stubbed Commands**:
  - ✅ ~~`createScanCommand()` - refactored with full DI implementation~~
  - `createCleanCommand()` - returns "clean command not yet refactored for dependency injection"  
  - `createGenerateCommand()` - returns "generate command not yet refactored for dependency injection"
  - `createSetupLintCommand()` - returns "setup-lint command not yet refactored for dependency injection"
- **Next Steps**: Refactor remaining commands to use dependency injection instead of global state
- **Dependencies**: Global state removal must be completed first

### Configuration Management

#### 🚧 CLI Flag Handling Not Implemented
**Priority: MEDIUM**
- **Issue**: Current DI implementation uses hardcoded default configuration
- **Location**: `cmd/antimoji/main.go` lines 22-37
- **Problem**: No CLI flag parsing for log levels, output formats, etc.
- **Next Steps**: 
  1. Add CLI flag parsing to main.go
  2. Map CLI flags to app.Config
  3. Preserve backward compatibility with existing flags

#### 🚧 Build Information Not Passed Through
**Priority: LOW**
- **Issue**: Build version hardcoded in `internal/app/application.go:89`
- **Current**: Returns "0.9.16-refactor" 
- **Expected**: Should use build-time variables (version, buildTime, gitCommit)
- **Next Steps**: Pass build info through Dependencies or Config

### Testing Infrastructure

#### 🚧 Integration Tests Missing
**Priority: MEDIUM**
- **Issue**: Only unit tests exist for DI foundation
- **Missing**: End-to-end tests for refactored application
- **Next Steps**: 
  1. Create integration test suite
  2. Test complete command workflows with DI
  3. Verify no regression in functionality

### Error Handling

#### 🚧 Result Type Panic Still Exists
**Priority: HIGH**
- **Issue**: `internal/types/result.go:37` still contains panic in Unwrap()
- **Status**: Not yet addressed
- **Risk**: Production code could panic on error conditions
- **Next Steps**: Implement safe unwrap methods as planned in refactoring plan

### Legacy Code Cleanup

#### 🚧 Old CLI Package Still Exists
**Priority: LOW**
- **Issue**: `internal/cli/` package still exists with old implementation
- **Status**: Not yet removed, but no longer used by main.go
- **Risk**: Confusion about which implementation is active
- **Next Steps**: Remove after all commands are refactored to use DI

---

## 📋 General Technical Debt

### Code Quality Issues

#### Missing Documentation
- **Issue**: Many functions lack proper godoc comments
- **Priority**: LOW
- **Files**: Various throughout codebase

#### Test Coverage Gaps
- **Issue**: Some packages have insufficient test coverage
- **Priority**: MEDIUM  
- **Target**: Maintain >85% coverage during refactoring

### Performance Concerns

#### Memory Allocations in Emoji Detection
- **Issue**: Excessive memory allocations during Unicode processing
- **Priority**: MEDIUM
- **Status**: Identified in refactoring plan, not yet addressed

---

## 🎯 Completion Criteria

### Task 1.1 - Global Logger Removal
- [ ] No `globalLogger` variables in codebase
- [ ] No `logging.(Info|Error|Debug|Warn)(` calls outside of global.go
- [ ] All commands accept logger via dependency injection
- [ ] All tests pass with injected mock logger
- [ ] Performance maintained (±5%)

### Task 1.2 - Global UI Output Removal
- [ ] No `globalUserOutput` variables in codebase  
- [ ] No `ui.(Info|Success|Error)(` global calls
- [ ] All output goes through injected interface
- [ ] All output formats work correctly

---

## 📝 Notes

- This file should be updated whenever stubbed functionality is added
- All TODO comments in code should also be tracked here
- Items should be removed when completed and verified
- Priority levels: HIGH (blocks other work), MEDIUM (should be done soon), LOW (nice to have)

## 🔧 Development Standards (Added 2025-09-09)

**Always run before committing:**
```bash
# Lint and test on every change
golangci-lint run    # Must pass with 0 issues
go test ./... -short # Must pass all tests

# Quality gate checks
make test-coverage-check  # Must be ≥65%
make antimoji-lint       # Must pass project linting
```

**Last Updated**: 2025-09-09  
**Next Review**: Weekly during refactoring phase
