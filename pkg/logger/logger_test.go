package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZapLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "default config",
			config: Config{
				Level:  InfoLevel,
				Format: JSONFormat,
			},
			wantErr: false,
		},
		{
			name: "debug level json format",
			config: Config{
				Level:  DebugLevel,
				Format: JSONFormat,
			},
			wantErr: false,
		},
		{
			name: "console format",
			config: Config{
				Level:  InfoLevel,
				Format: ConsoleFormat,
			},
			wantErr: false,
		},
		{
			name: "error level",
			config: Config{
				Level:  ErrorLevel,
				Format: JSONFormat,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewZapLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	logger := Nop() // Use nop logger for testing
	require.NotNil(t, logger)

	// Test all log methods don't panic
	t.Run("debug", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Debug("test message", String("key", "value"))
		})
	})

	t.Run("info", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Info("test message", String("key", "value"))
		})
	})

	t.Run("warn", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Warn("test message", String("key", "value"))
		})
	})

	t.Run("error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Error("test message", String("key", "value"))
		})
	})

	t.Run("with", func(t *testing.T) {
		newLogger := logger.With(String("field", "value"))
		assert.NotNil(t, newLogger)
	})

	t.Run("sync", func(t *testing.T) {
		err := logger.Sync()
		assert.NoError(t, err)
	})
}

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		wantKey  string
		wantType string
	}{
		{
			name:     "string field",
			field:    String("key", "value"),
			wantKey:  "key",
			wantType: "string",
		},
		{
			name:     "int field",
			field:    Int("count", 42),
			wantKey:  "count",
			wantType: "int",
		},
		{
			name:     "int64 field",
			field:    Int64("bigcount", 9223372036854775807),
			wantKey:  "bigcount",
			wantType: "int64",
		},
		{
			name:     "float64 field",
			field:    Float64("ratio", 3.14),
			wantKey:  "ratio",
			wantType: "float64",
		},
		{
			name:     "bool field",
			field:    Bool("enabled", true),
			wantKey:  "enabled",
			wantType: "bool",
		},
		{
			name:     "duration field",
			field:    Duration("elapsed_ms", 1234),
			wantKey:  "elapsed_ms",
			wantType: "int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantKey, tt.field.Key)
			assert.NotNil(t, tt.field.Value)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, InfoLevel, config.Level)
	assert.Equal(t, JSONFormat, config.Format)
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault()
	assert.NotNil(t, logger)
}

func TestMustNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger := MustNew(DefaultConfig())
			assert.NotNil(t, logger)
		})
	})
}
