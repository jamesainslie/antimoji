<!-- markdownlint-disable MD033 MD041 -->
<div align="center">
  <img src="pics/antimoji.png" alt="Antimoji Logo" width="200" height="200">
  <h1>Antimoji</h1>
  
  <p>
    <a href="https://github.com/jamesainslie/antimoji/actions/workflows/ci.yml"><img src="https://github.com/jamesainslie/antimoji/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="https://github.com/jamesainslie/antimoji/actions/workflows/build.yml"><img src="https://github.com/jamesainslie/antimoji/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
    <a href="https://github.com/jamesainslie/antimoji/actions/workflows/nightly.yml"><img src="https://github.com/jamesainslie/antimoji/actions/workflows/nightly.yml/badge.svg" alt="Nightly Build"></a>
    <br/>
    <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/jamesainslie/antimoji?color=00ADD8&logo=go" alt="Go Version"></a>
    <a href="https://github.com/jamesainslie/antimoji/releases"><img src="https://img.shields.io/github/v/release/jamesainslie/antimoji?color=blue&logo=github&cache=1756852585" alt="Latest Release"></a>
    <a href="https://github.com/jamesainslie/antimoji"><img src="https://img.shields.io/github/stars/jamesainslie/antimoji?color=yellow&logo=github" alt="GitHub Stars"></a>
    <br/>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/jamesainslie/antimoji?color=blue" alt="License"></a>
    <a href="https://github.com/jamesainslie/antimoji/commits/main"><img src="https://img.shields.io/github/commit-activity/m/jamesainslie/antimoji?color=blue&logo=github" alt="Commit Activity"></a>
    <a href="https://goreportcard.com/report/github.com/jamesainslie/antimoji"><img src="https://goreportcard.com/badge/github.com/jamesainslie/antimoji" alt="Go Report Card"></a>
    <a href="https://pkg.go.dev/github.com/jamesainslie/antimoji"><img src="https://pkg.go.dev/badge/github.com/jamesainslie/antimoji.svg" alt="Go Reference"></a>
  </p>
</div>
<!-- markdownlint-enable MD033 MD041 -->

> A blazing-fast CLI tool for detecting and removing emojis from code files and documentation.

Antimoji is a high-performance emoji detection and removal tool built with Go using functional programming principles. It provides comprehensive emoji scanning and cleaning capabilities for maintaining professional, emoji-free codebases.

## Features

### Core Capabilities
- **Unicode Emoji Detection**: Comprehensive support for Unicode 15.0+ emojis
- **Text Emoticon Detection**: Recognizes `:)`, `:(`, `:D` and other emoticons  
- **Custom Pattern Detection**: Supports `:smile:`, `:thumbs_up:` style patterns
- **Multi-Rune Support**: Handles skin tone modifiers and ZWJ sequences
- **Allowlist Filtering**: Configurable patterns to preserve specific emojis

### File Operations
- **Safe File Modification**: Atomic operations prevent data corruption
- **Backup Creation**: Automatic backups with timestamp naming
- **Permission Preservation**: Maintains original file permissions
- **Streaming Processing**: Memory-efficient handling of large files
- **Binary File Detection**: Automatically skips non-text files

### CLI Interface
- **Multiple Commands**: `scan` for detection, `clean` for removal
- **Output Formats**: Table, JSON, and CSV formats
- **Configuration Profiles**: Default, strict, and CI/CD profiles
- **Performance Statistics**: Built-in benchmarking and metrics
- **Dry-Run Mode**: Preview changes without file modification

## Installation

### From Source
```bash
git clone https://github.com/jamesainslie/antimoji.git
cd antimoji
make build
sudo cp bin/antimoji /usr/local/bin/
```

### Using Go Install
```bash
go install github.com/jamesainslie/antimoji/cmd/antimoji@latest
```

## Quick Start

### Scan for Emojis
```bash
# Scan current directory
antimoji scan .

# Scan specific files
antimoji scan file.go README.md

# Recursive scan with statistics
antimoji scan --recursive --stats src/

# JSON output for automation
antimoji scan --format json .
```

### Remove Emojis
```bash
# Preview changes (safe)
antimoji clean --dry-run .

# Remove emojis with backup
antimoji clean --backup --in-place .

# Custom replacement text
antimoji clean --replace "[EMOJI]" --in-place .

# Respect allowlist configuration
antimoji clean --respect-allowlist --in-place .
```

## Configuration

Antimoji uses XDG-compliant configuration files:

```yaml
# ~/.config/antimoji/config.yaml
version: "0.5.0"
profiles:
  default:
    # File processing
    recursive: true
    follow_symlinks: false
    backup_files: false
    
    # Emoji detection
    unicode_emojis: true
    text_emoticons: true
    custom_patterns: [":smile:", ":frown:", ":thumbs_up:"]
    
    # Allowlist (emojis to preserve)
    emoji_allowlist:
      - "✅"  # Checkmark for task completion
      - "❌"  # Cross mark for failures
      - "⚠️"  # Warning symbol
    
    # File filters
    include_patterns: ["*.go", "*.md", "*.js", "*.py", "*.ts"]
    exclude_patterns: ["vendor/*", "node_modules/*", ".git/*"]
    
    # Output
    output_format: "table"
    show_progress: true
    colored_output: true
```

### Configuration Profiles

#### Default Profile
Balanced settings for general development use with common allowlisted emojis.

#### Strict Profile
Zero-tolerance policy - removes all emojis regardless of type.

#### CI Profile
Optimized for CI/CD pipelines with JSON output and specific error codes.

## Usage Examples

### Development Workflow
```bash
# Check for emojis before commit
antimoji scan . --threshold 0 --fail-on-found

# Clean codebase maintaining status emojis
antimoji clean --respect-allowlist --backup --in-place .

# Generate report for code review
antimoji scan . --format json > emoji-report.json
```

### CI/CD Integration
```bash
# Fail build if emojis found
antimoji scan . --threshold 0 --format json --quiet

# Clean with strict policy
antimoji clean --profile strict --in-place .
```

### Large Repository Processing
```bash
# High-performance scanning
antimoji scan --recursive --stats --workers 8 .

# Memory-efficient cleaning
antimoji clean --stream --in-place large-repo/
```

## Performance

Antimoji is optimized for high-performance processing:

- **Small files (<1KB)**: >10,000 files/second
- **Medium files (1-100KB)**: >1,000 files/second  
- **Large files (>1MB)**: >100MB/second throughput
- **Memory usage**: <50MB for typical repositories
- **Startup time**: <100ms cold start

## Architecture

Antimoji follows clean architecture principles with functional programming:

```
CLI Layer          → Cobra commands, Viper config
Application Layer  → Command handlers, Config manager
Business Logic     → Emoji detector, File processor, Allowlist manager
Infrastructure     → File system, Concurrency, Memory management
```

### Key Design Principles
- **Functional Programming**: Pure functions and immutable data
- **Performance First**: Zero-copy operations and memory pooling
- **Safety Emphasis**: Atomic operations and comprehensive error handling
- **Test-Driven**: 85% minimum test coverage requirement

## Development

### Prerequisites
- Go 1.21 or later
- Make (for build automation)

### Development Setup
```bash
git clone https://github.com/jamesainslie/antimoji.git
cd antimoji
make dev-setup
make test-watch
```

### Running Tests
```bash
# Run all tests
make test

# Test with coverage
make test-coverage

# Run benchmarks
make benchmark

# Quality checks
make check-all
```

### Build
```bash
# Development build
make build

# Release build
make build-release
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests first (TDD approach)
4. Implement feature maintaining 85% test coverage
5. Ensure all linting passes (`make lint`)
6. Commit changes (`git commit -m 'feat: add amazing feature'`)
7. Push to branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Standards
- 85% minimum test coverage
- Zero golangci-lint warnings
- Functional programming principles
- Comprehensive documentation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Configuration powered by [Viper](https://github.com/spf13/viper)
- Testing with [Testify](https://github.com/stretchr/testify)
- Follows [Semantic Versioning](https://semver.org/)
- Inspired by functional programming principles

## Support

- **Documentation**: See [docs/](docs/) directory
- **Issues**: [GitHub Issues](https://github.com/jamesainslie/antimoji/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jamesainslie/antimoji/discussions)
- **Email**: james@ainslies.us

---

**Antimoji** - Keeping your codebase clean and professional, one emoji at a time.
