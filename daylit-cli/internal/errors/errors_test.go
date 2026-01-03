package errors

import (
	"errors"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "simple error",
			err:      errors.New("something went wrong"),
			expected: "Error: something went wrong",
		},
		{
			name:     "wrapped error",
			err:      errors.New("failed to connect: connection refused"),
			expected: "Error: failed to connect: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.err)
			if result != tt.expected {
				t.Errorf("Format(%v) = %q, want %q", tt.err, result, tt.expected)
			}
		})
	}
}

func TestFormatf(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple message",
			format:   "something went wrong",
			args:     nil,
			expected: "Error: something went wrong",
		},
		{
			name:     "formatted message with string",
			format:   "failed to load %s",
			args:     []interface{}{"database"},
			expected: "Error: failed to load database",
		},
		{
			name:     "formatted message with multiple args",
			format:   "connection to %s:%d failed",
			args:     []interface{}{"localhost", 5432},
			expected: "Error: connection to localhost:5432 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Formatf(tt.format, tt.args...)
			if result != tt.expected {
				t.Errorf("Formatf(%q, %v) = %q, want %q", tt.format, tt.args, result, tt.expected)
			}
		})
	}
}
