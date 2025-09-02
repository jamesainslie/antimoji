# Product Requirements Document: Antimoji

## Overview

Antimoji is a blazing-fast CLI tool written in Go that automatically detects and removes emojis from code files, markdown documents, and other text-based artifacts. The tool is designed for developers and teams who want to maintain clean, professional codebases free from emoji characters that can cause encoding issues, reduce readability, or violate coding standards. In a world with AI agents, plastering emojis all over the place, this tool is a must have.

## Product Vision

To provide the fastest, most reliable emoji detection and removal tool for software development workflows, seamlessly integrating into existing development processes while maintaining code integrity and performance.

## Target Users

- **Primary**: Software developers and engineering teams
- **Secondary**: Technical writers, documentation maintainers
- **Tertiary**: DevOps engineers managing CI/CD pipelines

## Core Features

### 1. Emoji Detection & Removal
- **Unicode Emoji Detection**: Detect all Unicode emoji characters (U+1F600-U+1F64F, U+1F300-U+1F5FF, U+1F680-U+1F6FF, U+1F1E0-U+1F1FF, etc.)
- **Text-based Emoticons**: Detect and handle ASCII emoticons (`:)`, `:(`, etc.)
- **Custom Emoji Patterns**: Support for platform-specific emoji syntax (`:smile:`, `:thumbs_up:`, etc.)
- **Safe Removal**: Remove emojis while preserving code structure and functionality
- **Replacement Options**: Replace with nothing, placeholder text, or custom strings

### 2. File System Operations
- **Recursive Directory Traversal**: Process entire repositories or directory trees
- **File Type Detection**: Smart detection of text-based files vs binary files
- **Glob Pattern Support**: Include/exclude files using glob patterns
- **Symlink Handling**: Configurable behavior for symbolic links
- **Large File Optimization**: Memory-efficient processing of large files

### 3. Git Integration
- **Git-Aware Processing**: Respect `.gitignore` files automatically
- **Commit History Analysis**: Scan and clean commit messages and diffs
- **Staged Files Only**: Option to process only git-staged files
- **Branch-Specific Processing**: Target specific branches or commit ranges
- **Pre-commit Hook Integration**: Easy integration with git hooks

### 4. Pipeline Integration & CI/CD
- **Exit Code Standards**: Return appropriate exit codes for pipeline integration
- **Linting Mode**: Fail-fast mode that exits with non-zero code when emojis are found
- **JSON Output**: Machine-readable output for CI/CD integration
- **Silent Mode**: Minimal output suitable for automated environments
- **Configurable Thresholds**: Set maximum allowed emoji count before failure

### 5. Performance Optimization
- **Concurrent Processing**: Multi-threaded file processing
- **Memory Efficiency**: Stream-based processing for large files
- **Caching**: Smart caching of file metadata and processing results
- **Progress Indicators**: Real-time progress reporting for large operations
- **Benchmark Mode**: Performance measurement and optimization tools

### 6. Configuration Management
- **XDG Base Directory**: Follow XDG specification for config file placement
- **YAML/JSON Config**: Flexible configuration file formats
- **Environment Variables**: Override config via environment variables
- **Profile Support**: Multiple configuration profiles for different projects
- **Config Validation**: Validate configuration files on startup

### 7. Filtering & Allowlist Management
- **Emoji Allowlist**: Configure specific emojis that should be ignored during scanning
- **File Ignore List**: Specify files and directories to exclude from processing
- **Pattern-Based Filtering**: Use regex patterns for advanced filtering rules
- **Context-Aware Allowlists**: Different allowlists for different file types or directories
- **Temporary Overrides**: Command-line flags to override configured filters

### 8. CLI Interface (Cobra Framework)
- **Intuitive Commands**: Clear, discoverable command structure
- **Rich Help System**: Comprehensive help and examples
- **Autocompletion**: Shell completion for commands and flags
- **Colored Output**: Syntax-highlighted, colored terminal output
- **Verbose/Quiet Modes**: Configurable output verbosity levels

## Technical Requirements

### Architecture
- **Language**: Go 1.24+ (latest stable version)
- **CLI Framework**: Cobra for command structure
- **Configuration**: Viper for configuration management
- **Concurrency**: Goroutines and channels for parallel processing
- **Unicode Support**: Full Unicode 15.0+ support
- **Versioning**: Semantic Versioning 2.0.0 (SemVer)

### Performance Targets
- **Processing Speed**: >100MB/s on modern hardware
- **Memory Usage**: <50MB for typical repository processing
- **Startup Time**: <100ms cold start
- **Scalability**: Handle repositories with 100k+ files

### File Format Support
- **Programming Languages**: All major languages (.go, .js, .py, .java, .c, .cpp, .rs, etc.)
- **Markup Languages**: Markdown, HTML, XML, YAML, JSON
- **Documentation**: README files, comments, docstrings
- **Configuration Files**: Various config formats

## Command Structure

```
antimoji [global flags] <command> [command flags] [arguments]

Global Flags:
  --config string       Config file path
  --profile string      Configuration profile
  --verbose, -v         Verbose output
  --quiet, -q           Quiet mode
  --silent             Silent mode for CI/CD (minimal output)
  --dry-run            Show what would be changed without modifying files
  --fail-on-found      Exit with non-zero code if emojis are found (linting mode)
  --allow-emoji string  Temporarily allow specific emojis (comma-separated)
  --ignore-file string  Temporarily ignore specific files (comma-separated)

Commands:
  scan               Scan files for emojis without modifying
  clean              Remove emojis from files
  git                Git-specific operations
  config             Configuration management
  stats              Show performance and usage statistics
  benchmark          Run performance benchmarks
  version            Show version information
```

### Subcommands

#### `antimoji scan`
```bash
antimoji scan [flags] [path...]
  --recursive, -r       Scan directories recursively
  --include string      Include file patterns (glob)
  --exclude string      Exclude file patterns (glob)
  --format string       Output format (table, json, csv)
  --count-only          Show only emoji counts
  --threshold int       Maximum allowed emoji count (for linting)
  --ignore-allowlist    Ignore configured emoji allowlist
  --stats              Show performance statistics
  --benchmark          Run in benchmark mode with detailed metrics
```

#### `antimoji clean`
```bash
antimoji clean [flags] [path...]
  --recursive, -r       Clean directories recursively
  --backup             Create backup files
  --replace string     Replacement text for emojis
  --in-place, -i       Modify files in place
  --respect-allowlist  Respect configured emoji allowlist during cleaning
  --stats              Show performance statistics
  --benchmark          Run in benchmark mode with detailed metrics
```

#### `antimoji git`
```bash
antimoji git [subcommand] [flags]
  commits            Clean commit messages
  staged             Clean staged files
  branch             Clean specific branch
  history            Clean commit history (dangerous)
```

#### `antimoji stats`
```bash
antimoji stats [flags]
  --format string    Output format (table, json, csv)
  --since string     Show stats since date (YYYY-MM-DD)
  --export string    Export stats to file
```

#### `antimoji benchmark`
```bash
antimoji benchmark [flags] [path...]
  --iterations int   Number of benchmark iterations (default: 10)
  --warmup int       Number of warmup runs (default: 3)
  --format string    Output format (table, json, csv)
  --export string    Export results to file
  --compare string   Compare with previous benchmark results
```

## Configuration Schema

```yaml
# ~/.config/antimoji/config.yaml
version: "0.1.0"  # Semantic versioning
profiles:
  default:
    # File processing
    recursive: true
    follow_symlinks: false
    backup_files: false
    
    # Emoji detection
    unicode_emojis: true
    text_emoticons: true
    custom_patterns: [":smile:", ":thumbs_up:"]
    
    # Allowlist and ignore functionality
    emoji_allowlist:
      - "✅"  # Checkmark for task completion
      - "❌"  # Cross mark for failures
      - "⚠️"  # Warning symbol
    file_ignore_list:
      - "*.min.js"
      - "*.min.css" 
      - "vendor/**/*"
      - "node_modules/**/*"
      - ".git/**/*"
      - "**/*.generated.*"
    directory_ignore_list:
      - ".git"
      - "node_modules" 
      - "vendor"
      - "dist"
      - "build"
    
    # Replacement behavior
    replacement: ""  # Empty string removes emojis
    preserve_whitespace: true
    
    # File filters
    include_patterns: ["*.go", "*.md", "*.js", "*.py", "*.ts", "*.jsx", "*.tsx"]
    exclude_patterns: ["vendor/*", "node_modules/*", ".git/*"]
    
    # CI/CD and linting
    fail_on_found: false
    max_emoji_threshold: 0  # Fail if more than this many emojis found
    exit_code_on_found: 1   # Exit code when emojis are found in linting mode
    
    # Performance
    max_workers: 0  # Auto-detect CPU cores
    buffer_size: 64KB
    
    # Output
    output_format: "table"
    show_progress: true
    colored_output: true
    
    # Performance and Observability
    enable_metrics: true
    enable_tracing: false  # Disabled by default for privacy
    metrics_endpoint: ""   # Local collection by default
    telemetry_opt_out: false
    performance_stats: false
    
  strict:
    # Stricter profile for production environments
    unicode_emojis: true
    text_emoticons: true
    custom_patterns: ["*emoji*", "*smiley*"]
    emoji_allowlist: []  # No emojis allowed in strict mode
    fail_on_found: true
    max_emoji_threshold: 0
    exit_code_on_found: 2  # Different exit code for strict mode
    
  ci:
    # Optimized for CI/CD pipelines
    recursive: true
    follow_symlinks: false
    backup_files: false
    unicode_emojis: true
    text_emoticons: true
    emoji_allowlist:
      - "✅"  # Common in CI/CD contexts
      - "❌"
      - "⚠️"
    fail_on_found: true
    max_emoji_threshold: 5  # Allow some emojis but limit excess
    output_format: "json"
    show_progress: false
    colored_output: false
    
    # Enhanced observability for CI/CD
    enable_metrics: true
    enable_tracing: true
    performance_stats: true
    telemetry_opt_out: false  # Analytics helpful for CI/CD optimization
```

## User Stories

### As a Developer
- **Story 1**: "I want to quickly scan my codebase to identify files containing emojis so I can assess the scope of cleanup needed."
- **Story 2**: "I want to automatically remove emojis from all Go files in my project while preserving code functionality."
- **Story 3**: "I want to integrate emoji removal into my pre-commit workflow to prevent emojis from entering the repository."

### As a Team Lead
- **Story 4**: "I want to enforce emoji-free code standards across all team repositories with a single configuration."
- **Story 5**: "I want to clean up legacy codebases that accumulated emojis over time without breaking existing functionality."

### As a DevOps Engineer
- **Story 6**: "I want to integrate antimoji into CI/CD pipelines to automatically reject commits containing emojis."
- **Story 7**: "I want to monitor and report on emoji usage across multiple repositories."
- **Story 8**: "I want to configure different emoji allowlists for different environments (dev vs prod)."
- **Story 9**: "I want the tool to return specific exit codes so my pipeline can handle emoji violations appropriately."

### As a Project Maintainer
- **Story 10**: "I want to allow certain functional emojis (like checkmarks) while blocking decorative ones."
- **Story 11**: "I want to ignore generated files and vendor dependencies during emoji scanning."
- **Story 12**: "I want to track performance metrics to optimize our emoji scanning processes."
- **Story 13**: "I want to understand usage patterns to improve the tool's effectiveness."

### As a Tool Developer
- **Story 14**: "I want to collect anonymous usage metrics to understand how the tool is being used."
- **Story 15**: "I want to track performance regressions across versions using benchmarking data."

## Success Metrics

### Performance Metrics
- **Processing Speed**: Measure MB/s throughput on various file sizes
- **Memory Efficiency**: Track memory usage during large repository processing
- **Accuracy**: 99.9%+ emoji detection rate with zero false positives

### User Experience Metrics
- **Time to First Value**: <5 minutes from installation to first successful scan
- **Error Rate**: <1% of operations result in user-facing errors
- **Documentation Coverage**: 100% of features documented with examples

### Pipeline Integration Metrics
- **CI/CD Adoption**: Track integration into major CI/CD platforms (GitHub Actions, GitLab CI, Jenkins)
- **Exit Code Reliability**: 100% consistent exit codes for pipeline decision making
- **Configuration Accuracy**: Zero false positives with properly configured allowlists

### Observability Metrics
- **Telemetry Coverage**: Track percentage of users with telemetry enabled
- **Performance Trend Analysis**: Monitor performance improvements across versions
- **Usage Pattern Insights**: Understand most common use cases and configurations
- **Error Rate Tracking**: Monitor and reduce error rates across different environments

## Risk Assessment

### Technical Risks
- **Unicode Complexity**: Emoji standards evolve; tool must stay updated
- **File Corruption**: Risk of damaging files during modification
- **Performance Degradation**: Large repositories might cause memory issues
- **Pipeline Breaking Changes**: SemVer violations could break existing CI/CD integrations
- **Allowlist Management**: Complex allowlist configurations could lead to user confusion
- **Privacy Concerns**: Users may be concerned about telemetry and usage tracking
- **Telemetry Infrastructure**: Dependency on external services for metrics collection

### Mitigation Strategies
- **Comprehensive Testing**: Unit tests for all emoji patterns and edge cases
- **Backup Functionality**: Always offer backup options for destructive operations
- **Gradual Rollout**: Beta testing with select users before general release
- **Semantic Versioning**: Strict adherence to SemVer to prevent breaking changes
- **Configuration Validation**: Built-in validation and helpful error messages for configurations
- **Pipeline Documentation**: Comprehensive CI/CD integration guides and examples
- **Privacy-First Design**: Telemetry disabled by default, clear opt-in mechanisms
- **Local-First Analytics**: Prefer local metrics collection with optional remote export
- **Transparent Data Practices**: Clear documentation of what data is collected and how it's used

## Implementation Phases

### Phase 1: Core Functionality (MVP) - v0.1.0
- Basic emoji detection and removal
- File system traversal
- Simple CLI interface
- XDG configuration support
- Basic allowlist functionality
- File ignore list implementation

### Phase 2: Pipeline Integration - v0.2.0
- Exit code standardization for CI/CD
- JSON output format
- Silent/quiet modes
- Linting mode with configurable thresholds
- Basic CI/CD documentation
- Basic performance statistics
- Local metrics collection

### Phase 3: Git Integration - v0.3.0
- Git-aware file processing
- Commit message cleaning
- Pre-commit hook support
- Staged files processing

### Phase 4: Performance Optimization - v0.4.0
- Concurrent processing
- Memory optimization
- Caching mechanisms
- Benchmark mode
- OpenTelemetry integration
- Advanced performance metrics
- Usage analytics (opt-in)

### Phase 5: Advanced Features - v1.0.0
- Context-aware allowlists
- Advanced pattern matching
- Multiple configuration profiles
- Enhanced pipeline integrations
- Breaking changes if needed for better API design

## Dependencies

### Core Dependencies
- **github.com/spf13/cobra**: CLI framework
- **github.com/spf13/viper**: Configuration management
- **golang.org/x/text/unicode**: Unicode handling
- **github.com/go-git/go-git**: Git operations

### Observability Dependencies
- **go.opentelemetry.io/otel**: OpenTelemetry SDK
- **go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc**: OTLP metrics exporter
- **go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp**: OTLP trace exporter
- **go.opentelemetry.io/otel/exporters/prometheus**: Prometheus metrics exporter
- **go.opentelemetry.io/otel/exporters/jaeger**: Jaeger trace exporter
- **go.opentelemetry.io/otel/sdk/metric**: Metrics SDK
- **go.opentelemetry.io/otel/sdk/trace**: Tracing SDK

### Development Dependencies
- **github.com/stretchr/testify**: Testing framework
- **github.com/golangci/golangci-lint**: Code linting
- **github.com/goreleaser/goreleaser**: Release automation

## Acceptance Criteria

### Functional Requirements
- [ ] Successfully detects all Unicode emoji categories
- [ ] Processes files without corrupting content
- [ ] Respects .gitignore files when processing repositories
- [ ] Provides accurate progress reporting for long operations
- [ ] Supports all major text file formats
- [ ] Implements configurable emoji allowlists with zero false positives
- [ ] Returns appropriate exit codes for CI/CD pipeline integration
- [ ] Supports file and directory ignore lists with glob patterns
- [ ] Follows semantic versioning for all releases

### Non-Functional Requirements
- [ ] Processes 1GB repository in under 10 seconds
- [ ] Uses less than 100MB memory for typical operations
- [ ] Starts up in under 100ms
- [ ] Provides clear error messages for all failure modes
- [ ] Maintains 100% backwards compatibility within major versions
- [ ] Achieves <1ms latency for allowlist lookups
- [ ] Supports repositories with 1M+ files without performance degradation

## Future Enhancements

- **IDE Integration**: Plugins for VS Code, GoLand, etc.
- **Web Interface**: Browser-based interface for non-CLI users
- **API Mode**: HTTP API for integration with other tools
- **Machine Learning**: AI-powered context-aware emoji detection
- **Internationalization**: Support for non-English emoji descriptions
- **Advanced Pipeline Features**: Integration with GitHub Actions marketplace, GitLab CI templates
- **Allowlist Sharing**: Community-maintained allowlist presets for different project types
- **Real-time Monitoring**: Dashboard for tracking emoji usage trends across projects

## Versioning Strategy

Antimoji follows [Semantic Versioning 2.0.0](https://semver.org/):

### Version Format: MAJOR.MINOR.PATCH

- **MAJOR**: Incompatible API changes, breaking configuration format changes
- **MINOR**: New functionality in a backwards-compatible manner
- **PATCH**: Backwards-compatible bug fixes, emoji database updates

### Release Schedule
- **Patch releases**: Monthly or as needed for critical bugs
- **Minor releases**: Quarterly for new features
- **Major releases**: Annually or when breaking changes are necessary

### Backwards Compatibility Promise
- Configuration files remain compatible within major versions
- CLI interface maintains backwards compatibility within major versions
- Exit codes and output formats stable within major versions
- Deprecation warnings provided one minor version before removal

## Conclusion

Antimoji addresses a real need in software development for maintaining clean, professional codebases. By focusing on performance, reliability, and ease of use, it will become an essential tool in developers' workflows for ensuring code quality and consistency.
