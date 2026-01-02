package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	// Create a temporary directory for logs
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")

	// Test normal mode (non-debug)
	err := Init(Config{
		Debug:     false,
		ConfigDir: configDir,
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Verify log directory was created
	logDir := filepath.Join(configDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("Log directory was not created: %s", logDir)
	}

	// Verify logger is not nil
	if Logger == nil {
		t.Error("Logger is nil after initialization")
	}

	// Test that we can log without errors
	Debug("Test debug message")
	Info("Test info message")
	Warn("Test warning message")
	Error("Test error message")
}

func TestInitDebugMode(t *testing.T) {
	// Create a temporary directory for logs
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")

	// Test debug mode
	err := Init(Config{
		Debug:     true,
		ConfigDir: configDir,
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger in debug mode: %v", err)
	}

	// Verify logger is not nil
	if Logger == nil {
		t.Error("Logger is nil after initialization")
	}

	// Test that we can log without errors
	Debug("Test debug message in debug mode")
	Info("Test info message in debug mode")
}

func TestLogFunctionsWithoutInit(t *testing.T) {
	// Reset logger to nil
	Logger = nil

	// These should not panic when Logger is nil
	Debug("Test debug message")
	Info("Test info message")
	Warn("Test warning message")
	Error("Test error message")
}

func TestInitWithInvalidDirectory(t *testing.T) {
	// Try to initialize with a path that can't be created
	// This is platform-dependent, so we'll just test with a reasonable path
	err := Init(Config{
		Debug:     false,
		ConfigDir: "/nonexistent/path/that/should/not/exist",
	})
	if err == nil {
		t.Skip("Unable to test invalid directory - path was created or already exists")
	}
}
