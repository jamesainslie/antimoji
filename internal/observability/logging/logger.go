// Package logging provides OpenTelemetry compliant structured logging for Antimoji.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	ctxutil "github.com/antimoji/antimoji/internal/observability/context"
)

// LogLevel represents the available log levels.
type LogLevel string

const (
	// LevelSilent disables all logging (default)
	LevelSilent LogLevel = "silent"
	// LevelDebug enables debug and all higher level logs
	LevelDebug LogLevel = "debug"
	// LevelInfo enables info and all higher level logs
	LevelInfo LogLevel = "info"
	// LevelWarn enables warn and error logs only
	LevelWarn LogLevel = "warn"
	// LevelError enables error logs only
	LevelError LogLevel = "error"
)

// LogFormat represents the available log output formats.
type LogFormat string

const (
	// FormatJSON outputs structured JSON logs (OTEL compliant)
	FormatJSON LogFormat = "json"
	// FormatText outputs human-readable text logs
	FormatText LogFormat = "text"
)

// Logger provides a structured logging interface with OTEL compliance.
type Logger interface {
	// Debug logs a debug-level message with optional key-value pairs
	Debug(ctx context.Context, msg string, keysAndValues ...any)
	// Info logs an info-level message with optional key-value pairs
	Info(ctx context.Context, msg string, keysAndValues ...any)
	// Warn logs a warn-level message with optional key-value pairs
	Warn(ctx context.Context, msg string, keysAndValues ...any)
	// Error logs an error-level message with optional key-value pairs
	Error(ctx context.Context, msg string, keysAndValues ...any)
	// With returns a logger with the given key-value pairs added to all log entries
	With(keysAndValues ...any) Logger
	// WithContext returns a logger that uses the given context
	WithContext(ctx context.Context) Logger
	// IsEnabled returns true if the logger would emit a log record at the given level
	IsEnabled(level LogLevel) bool
}

// Config holds the logging configuration.
type Config struct {
	// Level sets the minimum log level to output
	Level LogLevel
	// Format sets the output format (json or text)
	Format LogFormat
	// Output sets the output destination (defaults to os.Stderr)
	Output io.Writer
	// ServiceName is added to all log entries for service identification
	ServiceName string
	// ServiceVersion is added to all log entries for version tracking
	ServiceVersion string
}

// DefaultConfig returns a default logging configuration with silent mode.
func DefaultConfig() *Config {
	return &Config{
		Level:          LevelSilent,
		Format:         FormatJSON,
		Output:         os.Stderr,
		ServiceName:    "antimoji",
		ServiceVersion: "unknown",
	}
}

// otelLogger implements the Logger interface using OpenTelemetry.
type otelLogger struct {
	slogger *slog.Logger
	config  *Config
	ctx     context.Context
}

// NewLogger creates a new OTEL-compliant logger with the given configuration.
func NewLogger(config *Config) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// If silent mode, return a no-op logger
	if config.Level == LevelSilent {
		return newNoOpLogger(), nil
	}

	// Set up OTEL log provider
	_, err := setupOTELLogProvider(config)
	if err != nil {
		return nil, err
	}

	// Configure slog level
	var slogLevel slog.Level
	switch config.Level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Create slog logger with appropriate handler
	var slogger *slog.Logger
	if config.Format == FormatText {
		// Use text handler for human-readable output
		textHandler := slog.NewTextHandler(config.Output, &slog.HandlerOptions{
			Level: slogLevel,
		})
		slogger = slog.New(textHandler)
	} else {
		// Use JSON handler for structured output
		jsonHandler := slog.NewJSONHandler(config.Output, &slog.HandlerOptions{
			Level: slogLevel,
		})
		slogger = slog.New(jsonHandler)
	}

	// Add service metadata
	slogger = slogger.With(
		"service.name", config.ServiceName,
		"service.version", config.ServiceVersion,
	)

	return &otelLogger{
		slogger: slogger,
		config:  config,
		ctx:     context.Background(),
	}, nil
}

// setupOTELLogProvider configures the OpenTelemetry log provider.
func setupOTELLogProvider(config *Config) (*sdklog.LoggerProvider, error) {
	// For CLI applications, we'll use a simpler approach
	// Create a simple processor that doesn't batch to avoid deadlocks
	exporter, err := stdoutlog.New(
		stdoutlog.WithWriter(config.Output),
	)
	if err != nil {
		return nil, err
	}

	// Use simple processor instead of batch processor to avoid deadlocks in CLI
	processor := sdklog.NewSimpleProcessor(exporter)

	// Create logger provider with resource information
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(createResource(config)),
	)

	// Set as global logger provider
	global.SetLoggerProvider(provider)

	return provider, nil
}

// createResource creates an OTEL resource with service information.
func createResource(config *Config) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(config.ServiceName),
		semconv.ServiceVersion(config.ServiceVersion),
	)
}

// Debug logs a debug-level message with context information.
func (l *otelLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	if l.IsEnabled(LevelDebug) {
		// Enhance with context fields for better diagnostics
		enhancedKV := l.enhanceWithContext(ctx, keysAndValues...)
		l.slogger.DebugContext(ctx, msg, enhancedKV...)
	}
}

// Info logs an info-level message with context information.
func (l *otelLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	if l.IsEnabled(LevelInfo) {
		// Enhance with context fields for better diagnostics
		enhancedKV := l.enhanceWithContext(ctx, keysAndValues...)
		l.slogger.InfoContext(ctx, msg, enhancedKV...)
	}
}

// Warn logs a warn-level message with context information.
func (l *otelLogger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	if l.IsEnabled(LevelWarn) {
		// Enhance with context fields for better diagnostics
		enhancedKV := l.enhanceWithContext(ctx, keysAndValues...)
		l.slogger.WarnContext(ctx, msg, enhancedKV...)
	}
}

// Error logs an error-level message with context information.
func (l *otelLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	if l.IsEnabled(LevelError) {
		// Enhance with context fields for better diagnostics
		enhancedKV := l.enhanceWithContext(ctx, keysAndValues...)
		l.slogger.ErrorContext(ctx, msg, enhancedKV...)
	}
}

// enhanceWithContext adds context fields to the key-value pairs for better diagnostics.
func (l *otelLogger) enhanceWithContext(ctx context.Context, keysAndValues ...any) []any {
	// Extract context fields
	contextFields := ctxutil.ExtractContextFields(ctx)

	// Combine context fields with provided key-value pairs
	enhanced := make([]any, 0, len(contextFields)+len(keysAndValues))
	enhanced = append(enhanced, contextFields...)
	enhanced = append(enhanced, keysAndValues...)

	return enhanced
}

// With returns a logger with the given key-value pairs added to all log entries.
func (l *otelLogger) With(keysAndValues ...any) Logger {
	return &otelLogger{
		slogger: l.slogger.With(keysAndValues...),
		config:  l.config,
		ctx:     l.ctx,
	}
}

// WithContext returns a logger that uses the given context.
func (l *otelLogger) WithContext(ctx context.Context) Logger {
	return &otelLogger{
		slogger: l.slogger,
		config:  l.config,
		ctx:     ctx,
	}
}

// IsEnabled returns true if the logger would emit a log record at the given level.
func (l *otelLogger) IsEnabled(level LogLevel) bool {
	if l.config.Level == LevelSilent {
		return false
	}

	configLevel := l.levelToInt(l.config.Level)
	checkLevel := l.levelToInt(level)
	return checkLevel >= configLevel
}

// levelToInt converts LogLevel to int for comparison.
func (l *otelLogger) levelToInt(level LogLevel) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 999 // Silent mode
	}
}

// noOpLogger is a logger that does nothing (for silent mode).
type noOpLogger struct{}

// newNoOpLogger creates a new no-op logger.
func newNoOpLogger() Logger {
	return &noOpLogger{}
}

func (n *noOpLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {}
func (n *noOpLogger) Info(ctx context.Context, msg string, keysAndValues ...any)  {}
func (n *noOpLogger) Warn(ctx context.Context, msg string, keysAndValues ...any)  {}
func (n *noOpLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {}
func (n *noOpLogger) With(keysAndValues ...any) Logger                            { return n }
func (n *noOpLogger) WithContext(ctx context.Context) Logger                      { return n }
func (n *noOpLogger) IsEnabled(level LogLevel) bool                               { return false }
