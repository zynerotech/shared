package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   error
	}{
		{
			name:   "default config",
			config: Config{},
			want:   nil,
		},
		{
			name: "debug level",
			config: Config{
				Level:  "debug",
				Format: "json",
				Output: "stdout",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != (tt.want != nil) {
				t.Errorf("New() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		logger: zerolog.New(&buf),
	}

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	newLogger := l.WithFields(fields)
	newLogger.Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "key1") {
		t.Error("Field key1 not found in output")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test that global functions don't panic
	Debug().Msg("global debug")
	Info().Msg("global info")
	Warn().Msg("global warn")
	Error().Msg("global error")
}

func TestSetLevel(t *testing.T) {
	// Test valid levels
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	for _, level := range validLevels {
		if err := SetLevel(level); err != nil {
			t.Errorf("SetLevel(%s) returned error: %v", level, err)
		}
	}

	// Test invalid level
	if err := SetLevel("invalid"); err == nil {
		t.Error("SetLevel('invalid') should return error")
	}
}

func TestSanitizeConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected Config
	}{
		{
			name:  "empty config",
			input: Config{},
			expected: Config{
				Level:      "info",
				Format:     "json",
				Output:     "stdout",
				TimeFormat: time.RFC3339,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitize(&tt.input)
			if result.Level != tt.expected.Level {
				t.Errorf("Expected Level=%s, got %s", tt.expected.Level, result.Level)
			}
		})
	}
}

func TestInit(t *testing.T) {
	cfg := Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	if err := Init(cfg); err != nil {
		t.Errorf("Init() returned error: %v", err)
	}

	// Test that global logger is set
	globalLogger := GetGlobal()
	if globalLogger == nil {
		t.Error("Global logger not set after Init()")
	}
}
