# Changelog

All notable changes to Antimoji will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- **Quality Tooling**: golangci-lint configuration and code quality standards
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
