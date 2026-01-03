package errors

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
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

// TestFatal tests the Fatal function using exec helper process
func TestFatal(t *testing.T) {
	if os.Getenv("GO_TEST_FATAL") == "1" {
		// This is the subprocess - call Fatal
		Fatal(errors.New("test error"))
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "GO_TEST_FATAL=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// Check that exit code is 1
		if e.ExitCode() != 1 {
			t.Errorf("Fatal() exit code = %d, want 1", e.ExitCode())
		}
		// Check that stderr contains the error message
		stderrStr := stderr.String()
		if !strings.Contains(stderrStr, "Error: test error") {
			t.Errorf("Fatal() stderr = %q, want to contain %q", stderrStr, "Error: test error")
		}
	} else {
		t.Errorf("Fatal() did not exit with error: %v", err)
	}
}

// TestFatal_NilError tests that Fatal does nothing when passed a nil error
func TestFatal_NilError(t *testing.T) {
	if os.Getenv("GO_TEST_FATAL_NIL") == "1" {
		// This is the subprocess - call Fatal with nil
		Fatal(nil)
		// If we get here, the function returned normally (which is correct)
		os.Exit(0)
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_NilError")
	cmd.Env = append(os.Environ(), "GO_TEST_FATAL_NIL=1")

	err := cmd.Run()
	if err != nil {
		t.Errorf("Fatal(nil) should not exit, but got error: %v", err)
	}
}

// TestFatalf tests the Fatalf function using exec helper process
func TestFatalf(t *testing.T) {
	if os.Getenv("GO_TEST_FATALF") == "1" {
		// This is the subprocess - call Fatalf
		Fatalf("connection to %s:%d failed", "localhost", 5432)
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf")
	cmd.Env = append(os.Environ(), "GO_TEST_FATALF=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// Check that exit code is 1
		if e.ExitCode() != 1 {
			t.Errorf("Fatalf() exit code = %d, want 1", e.ExitCode())
		}
		// Check that stderr contains the formatted error message
		stderrStr := stderr.String()
		if !strings.Contains(stderrStr, "Error: connection to localhost:5432 failed") {
			t.Errorf("Fatalf() stderr = %q, want to contain %q", stderrStr, "Error: connection to localhost:5432 failed")
		}
	} else {
		t.Errorf("Fatalf() did not exit with error: %v", err)
	}
}
