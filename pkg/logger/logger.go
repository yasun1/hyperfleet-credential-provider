package logger

import (
	"context"
)

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message with optional fields
	Debug(msg string, fields ...Field)

	// Info logs an info message with optional fields
	Info(msg string, fields ...Field)

	// Warn logs a warning message with optional fields
	Warn(msg string, fields ...Field)

	// Error logs an error message with optional fields
	Error(msg string, fields ...Field)

	// With returns a new logger with additional fields
	With(fields ...Field) Logger

	// WithContext returns a new logger with context
	WithContext(ctx context.Context) Logger

	// Sync flushes any buffered log entries
	Sync() error
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value type
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field (in milliseconds)
func Duration(key string, ms int64) Field {
	return Field{Key: key, Value: ms}
}

// Level represents the log level
type Level string

const (
	// DebugLevel enables debug logging
	DebugLevel Level = "debug"
	// InfoLevel enables info logging
	InfoLevel Level = "info"
	// WarnLevel enables warn logging
	WarnLevel Level = "warn"
	// ErrorLevel enables error logging
	ErrorLevel Level = "error"
)

// Format represents the log output format
type Format string

const (
	// JSONFormat outputs logs in JSON format
	JSONFormat Format = "json"
	// ConsoleFormat outputs logs in human-readable console format
	ConsoleFormat Format = "console"
)

// Config holds logger configuration
type Config struct {
	Level  Level
	Format Format
	Output interface{} // io.Writer, defaults to os.Stderr
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: nil, // Will default to os.Stderr in implementation
	}
}
