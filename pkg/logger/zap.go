package logger

import (
	"context"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger wraps zap.Logger to implement our Logger interface
type zapLogger struct {
	logger *zap.Logger
}

// NewZapLogger creates a new zap-based logger
func NewZapLogger(config Config) (Logger, error) {
	// Determine output writer
	var writer io.Writer
	if config.Output != nil {
		var ok bool
		writer, ok = config.Output.(io.Writer)
		if !ok {
			writer = os.Stderr
		}
	} else {
		writer = os.Stderr
	}

	// Set log level
	level := zapcore.InfoLevel
	switch config.Level {
	case DebugLevel:
		level = zapcore.DebugLevel
	case InfoLevel:
		level = zapcore.InfoLevel
	case WarnLevel:
		level = zapcore.WarnLevel
	case ErrorLevel:
		level = zapcore.ErrorLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Set encoder based on format
	var encoder zapcore.Encoder
	if config.Format == ConsoleFormat {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core with custom output
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		level,
	)

	// Build logger
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip wrapper functions
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &zapLogger{logger: logger}, nil
}

// Debug logs a debug message
func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.convertFields(fields)...)
}

// Info logs an info message
func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.convertFields(fields)...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.convertFields(fields)...)
}

// Error logs an error message
func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.convertFields(fields)...)
}

// With returns a new logger with additional fields
func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		logger: l.logger.With(l.convertFields(fields)...),
	}
}

// WithContext returns a new logger with context (placeholder for future tracing integration)
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	// Future: extract trace ID, span ID from context
	return l
}

// Sync flushes any buffered log entries
func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// convertFields converts our Field type to zap.Field
func (l *zapLogger) convertFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}

// New creates a new logger with the given configuration
func New(config Config) (Logger, error) {
	return NewZapLogger(config)
}

// MustNew creates a new logger and panics on error
func MustNew(config Config) Logger {
	logger, err := New(config)
	if err != nil {
		panic(err)
	}
	return logger
}

// NewDefault creates a logger with default configuration
func NewDefault() Logger {
	return MustNew(DefaultConfig())
}

// Nop returns a no-op logger for testing
func Nop() Logger {
	return &zapLogger{logger: zap.NewNop()}
}
