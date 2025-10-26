# Changelog

All notable changes to Antimoji will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- **golangci-lint Integration**: Removed golangci.yml configuration from setup-lint command
  - Removed `--golangci` flag and `GolangCIConfig` option from setup-lint
  - Removed `ensureBasicGolangCIConfig()` and `generateGolangCIConfigForMode()` functions
  - Removed golangci.yml file generation and analysis functionality
  - Updated command help text and documentation to remove golangci references
  - antimoji is not an official golangci-lint linter and caused configuration errors
  - Focus remains on pre-commit hooks and standalone emoji detection

## [v0.9.16] - 2025-09-08

### Fixed
- **Critical Binary File Bug**: Fixed inconsistent binary file handling between scan and clean operations
  - Scan correctly skipped binary files, but clean processed them anyway
  - ModifyFile was missing the binary file detection check that ProcessFile had
  - Resolved anomalous behavior where clean reported emojis that scan didn't detect
- **OTEL Logging Deadlock**: Replaced BatchProcessor with SimpleProcessor to prevent CLI deadlocks
- **Logging Bypass Issues**: Eliminated direct stderr/stdout usage that bypassed the logging system

### Added
- **Comprehensive User Output System**: New `internal/ui` package for user-facing messages
  - Colored output with success/error/warning indicators
  - Separate user output from diagnostic logging
  - Multiple output levels (silent, normal, verbose, debug)
- **Enhanced Context Propagation**: New `internal/observability/context` package
  - Structured context keys for operation, component, file path tracking
  - Automatic context field extraction for logging
  - Proper context propagation throughout the application
- **Advanced Emoji Detection Debugging**: 
  - Detailed Unicode code point logging for detected emojis
  - Pattern application tracking and performance metrics
  - Debug information for emoji matches with range details
- **Enhanced Binary File Detection**:
  - Detailed logging of binary file detection reasoning
  - Consistent binary file handling across all operations
  - Better diagnostics for file type classification

### Changed
- **Logging Architecture**: Complete overhaul of logging system
  - OpenTelemetry compliant structured logging for diagnostics
  - Separate user-facing output system for CLI interaction
  - Context-aware logging with automatic field enhancement
- **CLI Output**: Replaced direct fmt.Printf calls with structured output
- **Error Handling**: Enhanced error correlation with context information
- **Performance Monitoring**: Added timing, content size, and pattern metrics

### Technical Details
- Fixed ModifyFile to include binary file detection check (critical consistency fix)
- Enhanced OTEL setup with proper resource configuration and semantic conventions
- Implemented comprehensive debugging system for troubleshooting detection issues
- Added structured logging with service name, version, operation, and component tracking
- Created thread-safe global logger and user output systems

### Impact
- **Consistent Behavior**: Scan and clean operations now behave identically
- **Better Diagnostics**: Complete visibility into emoji detection and file processing
- **Improved User Experience**: Clean, colored output separate from diagnostic logs
- **Production Ready**: Structured logs ready for monitoring and analysis

## [v0.9.15] - 2025-09-07

### Fixed
- **Pre-commit Integration Bug**: Fixed critical issue where clean step would report "0 modified" while scan step would still find emojis, causing zero-tolerance enforcement to fail (#6)
- **Missing Flag Support**: Added `--ignore-allowlist` flag to clean command for consistency with scan command
- **Profile Inconsistency**: Resolved issues where different profiles between clean and scan commands caused unreliable behavior
- **Setup-Lint Configuration**: Updated `setup-lint` command to generate improved two-step workflow (clean + verify) with consistent profiles

### Added
- **Unified Allowlist Processing**: New `internal/core/allowlist/processor.go` provides consistent allowlist handling across all commands
- **Pre-commit Examples**: Added `examples/pre-commit/` directory with working zero-tolerance and CI-friendly configurations
- **Comprehensive Documentation**: Added troubleshooting guide and best practices for pre-commit integration
- **Integration Tests**: Added `internal/cli/clean_precommit_test.go` with comprehensive pre-commit workflow testing

### Changed
- **Setup-Lint Output**: Now generates improved pre-commit configurations using `entry: antimoji` with `args:` instead of problematic `bash -c` patterns
- **CI Workflow**: Updated antimoji linting threshold to be compatible with ci-lint profile design
- **Flag Consistency**: Made `--ignore-allowlist` take precedence over `--respect-allowlist` in both commands for consistent behavior

### Technical Details
- Resolved the "clean reports 0 modified but still finds emojis" issue by ensuring consistent profile usage
- Fixed argument forwarding issues in pre-commit hook configurations  
- Added validation to detect common pre-commit configuration mistakes
- Maintained full backward compatibility with existing configurations

## [v0.9.14] - 2025-07-09

### Fixed
- Fixed ERROR status when using configuration files - files now process correctly with custom configs
- Resolved issue where minimal config files caused `MaxFileSize: 0` and `BufferSize: 0`, breaking file processing
- Added sensible defaults for processing configuration (100MB max file size, 64KB buffer)
- Enabled emoji detection by default when not explicitly specified in config files
- Fixed zero-tolerance emoji policies and CI/CD linting with custom configurations

## [v0.9.13] - 2025-09-07

### Fixed
- **File Inclusion Logic**: Fixed `shouldIncludeFile()` to include all files when `IncludePatterns` is empty, resolving restrictive default behavior that excluded files like `.txt`, `.yaml`, `.json`
- **Generate Command Consistency**: Unified file discovery logic between `scan` and `generate` commands, ensuring consistent behavior across all antimoji commands
- **Default Configuration**: Updated default config to use empty include patterns, enabling processing of all file types by default

### Removed
- **Config Version Field**: Eliminated useless version field from configuration structures that provided zero functional value and was never used by runtime code
- **Duplicate Code**: Removed `discoverAllFiles()` function and hardcoded extension lists in favor of consistent file discovery logic

### Improved
- **Test Coverage**: Added comprehensive file inclusion tests with 13 different scenarios
- **Code Cleanliness**: Removed 50+ lines of ceremonial code that served no functional purpose
- **Documentation**: Enhanced code comments explaining file inclusion behavior

## [v0.9.7] - 2025-09-06

### Fixed
- **Token Permissions**: Updated PAT_TOKEN with enhanced permissions including write:packages
- **GoReleaser Integration**: Should now properly push Homebrew formula updates

## [v0.9.6] - 2025-09-06

### Fixed
- **GoReleaser Homebrew Integration**: Properly configured GoReleaser to handle Homebrew formula updates
- **Formula Structure**: Added skip_upload: auto and proper commit message template
- **Branch Specification**: Explicitly set branch: main for Homebrew repository updates

### Improved
- **Homebrew Testing**: Enhanced formula test to include emoji functionality validation
- **Documentation**: Added setup-lint command to Homebrew formula caveats

## [v0.9.5] - 2025-09-06

### Fixed
- **Homebrew Integration**: Fixed GoReleaser Homebrew tap integration with proper PAT token
- **Release Workflow**: Streamlined workflow to use GoReleaser for all distribution channels
- **Docker Build**: Resolved all Docker build issues with proper multi-stage build

### Changed
- **Distribution**: Unified distribution through GoReleaser for consistency
- **Homebrew Updates**: Now handled automatically by GoReleaser with proper permissions

## [v0.9.4] - 2025-09-06

### Added
- **New `setup-lint` Command**: Comprehensive automated linting configuration setup with three tolerance modes
  - **Zero-tolerance mode**: Disallows ALL emojis in source code (strictest policy)
  - **Allow-list mode**: Allows only specified emojis (default: ✅, ❌) with customizable allowlist
  - **Permissive mode**: Warns about excessive emoji usage but doesn't fail builds
- **Automated Configuration Generation**: 
  - Creates `.antimoji.yaml` with mode-specific profiles and settings
  - Updates `.pre-commit-config.yaml` with antimoji linting hooks
- **Flexible Setup Options**:
  - `--allowed-emojis` flag for custom emoji allowlists
  - `--force` flag to overwrite existing configurations  
  - `--skip-precommit` flag to skip pre-commit hook installation
  - `--output-dir` to specify target directory
- **Smart Default Configurations**: Mode-specific profiles with appropriate thresholds, file patterns, and behaviors
- **Comprehensive Help Documentation**: Detailed usage examples and configuration guidance

### Fixed
- **Pre-commit Hook Integration**: Fixed go-vet hook compatibility issues with Go modules
- **Local Hook Implementation**: Replaced external pre-commit golang hooks with reliable local implementations
- **Code Formatting**: Resolved trailing whitespace and end-of-file formatting issues across codebase

### Changed
- **Pre-commit Configuration**: Migrated to local go-fmt, go-vet, and go-mod-tidy hooks for better reliability
- **CLI Command Structure**: Added setup-lint as a core subcommand alongside scan, clean, and generate

### Technical Details
- **Comprehensive Test Suite**: Added extensive unit and integration tests for setup-lint functionality
- **Configuration Management**: Leverages existing config.Config and Profile structures for consistency
- **Mode-specific Profiles**: Each linting mode generates appropriate emoji detection settings, thresholds, and file filters
- **Pre-commit Integration**: Seamless integration with existing pre-commit workflow and antimoji auto-cleaning

### Impact
- **Developer Experience**: One-command setup for emoji linting in any project
- **CI/CD Integration**: Automated configuration for continuous integration emoji policies  
- **Flexibility**: Support for different team preferences from strict to permissive emoji usage
- **Consistency**: Standardized emoji linting configuration across projects and teams

## [v0.9.3] - 2025-09-06

### Fixed
- **Critical UTF-8 Emoji Detection Bug**: Fixed issue where files containing emojis were incorrectly classified as binary files, causing emoji detection to be completely skipped
- **Multi-byte Unicode Handling**: Resolved bug where UTF-8 continuation bytes were counted as "non-printable" characters, triggering false binary file detection
- **Text File Classification**: Replaced byte-by-byte analysis with proper UTF-8 rune-aware processing for accurate text/binary classification

### Changed
- **UTF-8 Processing**: Enhanced `isTextContent()` function to use `utf8.DecodeRune()` for proper Unicode character processing
- **Performance**: Maintained performance while fixing Unicode handling - no regression in processing speed
- **Testing**: Added comprehensive emoji edge case testing with 13 new test scenarios covering various emoji densities and Unicode combinations

### Technical Details
- Rewrote text content detection to process complete Unicode runes instead of individual bytes
- Only count actual control characters as non-printable, not valid UTF-8 continuation bytes
- Added extensive test coverage for emoji-heavy content, mixed Unicode text, and binary detection edge cases
- Maintains robust binary file detection while properly handling all Unicode text content

### Impact
- Files with emojis now correctly identified as text and processed for emoji detection
- Zero-tolerance emoji policies can now be properly enforced on all text files
- Fixes critical issue where antimoji appeared to work but wasn't detecting emojis due to incorrect file classification

## [v0.9.2] - 2025-09-03

### Fixed
- **Docker Build Issues**: Resolved Docker build context problems in GoReleaser release process
- **Cross-Platform Compatibility**: Added comprehensive Windows support for all tests and builds
- **CI/CD Pipeline**: Fixed linting configuration compatibility and workflow issues
- **File Path Handling**: Improved cross-platform file path categorization using `filepath.ToSlash()`
- **Security Scanning**: Resolved gosec security scan integration in CI environment
- **Build System**: Added proper Windows `.exe` binary generation in CI workflows
- **Test Reliability**: Fixed timing-sensitive concurrency tests with platform-specific tolerances
- **Error Handling**: Enhanced error message assertions for Windows vs Unix compatibility

### Changed
- **Docker Base Image**: Switched to scratch image for minimal size and better reliability
- **CI Workflows**: Improved Windows PowerShell syntax and cross-platform shell handling
- **Test Coverage**: Maintained 87.3% coverage across all platforms

### Security
- **File Permissions**: Fixed configuration file permissions from 0644 to 0600
- **Path Validation**: Added proper nosec annotations for intentional file operations
- **Build Security**: Ensured all security scans pass with zero issues

## [v0.6.0] - 2024-12-02

### Added
- **Enhanced Concurrency System**: Complete worker pool integration with intelligent processing decisions
- **Performance Control**: `--workers` flag for user control over concurrent processing
- **Smart Processing Logic**: Automatic fallback to sequential processing for optimal performance
- **Build System**: LD flags for version, build time, and git commit information
- **Installation System**: Cross-platform `make install/uninstall` targets for system deployment
- **Version Command**: Detailed version information with build metadata
- **Comprehensive Benchmarks**: Performance comparison between sequential and concurrent processing

### Changed
- Enhanced file processing pipeline with concurrent worker pool integration
- Improved CLI with worker count control and performance tuning options
- Professional build system with automated version detection from git tags

### Performance
- **45% Performance Improvement**: 1,285 files/second (up from ~880 files/second)
- **Intelligent Concurrency**: Sequential for small workloads, concurrent for large workloads
- **Resource Optimization**: Configurable worker pools with automatic CPU detection
- **Benchmark Results**: Comprehensive performance analysis with memory usage tracking

### Technical Details
- Implemented production-ready worker pool with lifecycle management
- Added context-aware processing with graceful shutdown
- Created comprehensive concurrency testing with 79.2% coverage
- Integrated atomic operations for thread-safe metrics tracking

## [v0.5.0] - 2024-12-02

### Added
- **File Modification Engine**: Complete emoji removal functionality with atomic operations
- **Clean Command**: Full-featured `antimoji clean` command with comprehensive options
- **Backup System**: Automatic backup creation with timestamp-based naming
- **Atomic File Operations**: Corruption-free file updates using temporary files + rename
- **Permission Preservation**: Maintains original file permissions and metadata
- **Custom Replacement**: `--replace` flag for custom emoji replacement text
- **Dry-Run Mode**: Preview changes without modifying files
- **Safety Validation**: Prevents accidental file modification without explicit flags

### Changed
- Enhanced CLI with `clean` subcommand alongside existing `scan` command
- Improved error handling with detailed modification result reporting
- Extended processor package with file modification capabilities

### Technical Details
- Implemented atomic file operations to prevent data corruption
- Added comprehensive backup functionality with timestamp naming
- Created ModifyConfig for flexible modification behavior
- Integrated allowlist filtering into file modification pipeline
- Added performance benchmarks for file modification operations

## [v0.4.0] - 2024-12-02

### Added
- **Allowlist System**: Configurable emoji allowlists with pattern matching
- **Unicode Normalization**: Consistent emoji matching across different representations
- **Pattern-Based Filtering**: Support for exact emoji pattern matching
- **Allowlist Merging**: Combine multiple allowlists efficiently
- **Default Allowlists**: Pre-configured patterns for common use cases
- **CLI Integration**: `--ignore-allowlist` flag for testing and overrides

### Changed
- Enhanced scan command to respect configured allowlists
- Improved emoji detection accuracy with normalized matching
- Extended configuration system to support allowlist patterns

### Performance
- O(1) allowlist lookup performance using hash map optimization
- Sub-millisecond lookup times for large allowlists (1000+ patterns)
- Property-based testing ensures deterministic and monotonic behavior

### Technical Details
- Implemented Unicode normalization for variation selectors
- Added comprehensive allowlist engine with 94.0% test coverage
- Created functional allowlist filtering with zero false positives
- Integrated allowlist support throughout the processing pipeline

## [v0.3.0] - 2024-12-02

### Added
- **CLI Foundation**: Complete Cobra-based command-line interface
- **Configuration Management**: Viper-powered YAML configuration with XDG compliance
- **Scan Command**: Full-featured `antimoji scan` with multiple output formats
- **Profile System**: Multiple configuration profiles (default, strict, ci)
- **Output Formats**: Table, JSON, and CSV output modes
- **Performance Statistics**: `--stats` flag for processing metrics
- **File Discovery**: Recursive directory traversal with pattern filtering
- **Threshold Mode**: `--threshold` flag for CI/CD integration

### Changed
- Restructured project with proper CLI architecture
- Added comprehensive configuration validation and loading
- Enhanced file processing with configurable patterns

### Performance
- File discovery with efficient pattern matching
- Configurable worker pools and buffer sizes
- Statistics reporting with throughput metrics

### Technical Details
- Implemented Cobra command structure with global and local flags
- Added Viper configuration loading with environment variable support
- Created comprehensive CLI testing with 87.3% coverage
- Integrated all previous phases into cohesive CLI experience

## [v0.2.0] - 2024-12-02

### Added
- **File Processing Pipeline**: Streaming file processing with memory efficiency
- **File System Operations**: Safe file reading with binary detection
- **Processing Configuration**: Configurable emoji detection behavior
- **Batch Processing**: Multi-file processing with error resilience
- **Text File Detection**: Heuristic-based binary vs text file identification
- **Large File Support**: Streaming architecture for memory-efficient processing

### Changed
- Enhanced emoji detection with configurable pattern filtering
- Improved error handling throughout the processing pipeline
- Added comprehensive file metadata handling

### Performance
- Streaming file processing for large files (>100MB)
- Memory-efficient chunk-based reading
- Concurrent processing foundation

### Technical Details
- Implemented functional processing pipeline with pure functions
- Added comprehensive file system abstraction layer
- Created processing configuration system with validation
- Achieved 86.7% test coverage with integration tests

## [v0.1.0] - 2024-12-02

### Added
- **Core Emoji Detection Engine**: Unicode emoji detection across all major ranges
- **Multi-Pattern Support**: Unicode emojis, text emoticons, and custom patterns
- **Position Tracking**: Accurate line/column position calculation for detected emojis
- **Multi-Rune Support**: Skin tone modifiers and Zero-Width Joiner sequences
- **Pattern Compilation**: Optimized emoji pattern matching
- **Comprehensive Testing**: Property-based testing with 93.1% coverage

### Performance
- Unicode range detection with efficient lookup algorithms
- Multi-rune emoji handling (skin tone modifiers, ZWJ sequences)
- Performance benchmarks showing >10MB/s processing speed for typical files

### Technical Details
- Implemented functional emoji detection with pure functions
- Added comprehensive Unicode range definitions for all emoji categories
- Created robust position calculation system with byte/character mapping
- Developed property-based tests ensuring deterministic behavior
- Established performance benchmarking framework

## [v0.0.1] - 2024-12-02

### Added
- **Project Foundation**: Complete project structure and development tooling
- **Functional Programming Core**: Result[T] type for functional error handling
- **Development Infrastructure**: Comprehensive testing framework and CI/CD
- **Quality Tooling**: Linting configuration and code quality standards
- **Documentation**: Complete PRD, architecture documentation, and implementation plan

### Technical Details
- Established Go 1.21 project with modern dependency management
- Implemented functional Result[T] type with Map, FlatMap operations
- Set up comprehensive testing framework with 100% coverage requirement
- Created development workflow with make-based automation
- Established GitHub Actions CI/CD pipeline

---

## Development Philosophy

Throughout all releases, Antimoji has maintained:

- **Test-Driven Development**: 85% minimum test coverage enforced
- **Functional Programming**: Pure functions and immutable data structures
- **Performance-First**: Blazing-fast processing with comprehensive benchmarks
- **Quality Standards**: Zero-tolerance for linting issues and technical debt
- **User Safety**: Atomic operations, backups, and comprehensive error handling

## Migration Guide

### Upgrading from v0.4.x to v0.5.0
- New `clean` command available alongside existing `scan` command
- File modification requires explicit `--in-place` or `--dry-run` flag for safety
- Backup functionality available with `--backup` flag
- All existing `scan` functionality remains unchanged

### Upgrading from v0.3.x to v0.4.0
- Allowlist functionality now integrated into scan results
- Use `--ignore-allowlist` flag to see all detected emojis
- Configuration files can now include `emoji_allowlist` sections
- All existing functionality remains backward compatible

### Upgrading from v0.2.x to v0.3.0
- CLI interface now available with `antimoji scan` command
- Configuration files follow XDG Base Directory specification
- Environment variables supported with `ANTIMOJI_` prefix
- Library usage unchanged, new CLI layer added

### Upgrading from v0.1.x to v0.2.0
- File processing now available through processor package
- Configuration-driven emoji detection behavior
- Binary file detection and automatic skipping
- All detection functionality remains unchanged

## Contributors

- James Ainslie <james@ainslies.us> - Primary developer and maintainer

## License

This project is licensed under the MIT License - see the LICENSE file for details.
