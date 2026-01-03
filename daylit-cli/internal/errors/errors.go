package errors

import (
	"fmt"
	"os"

	"github.com/julianstephens/daylit/daylit-cli/internal/logger"
)

// Format formats an error message with a consistent "Error: " prefix
func Format(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Error: %v", err)
}

// Formatf formats an error message with a consistent "Error: " prefix using a format string
func Formatf(format string, args ...interface{}) string {
	return fmt.Sprintf("Error: "+format, args...)
}

// Fatal logs an error and exits the program with exit code 1.
// Note: This function depends on the logger being initialized via logger.Init().
// If the logger is not initialized, the error will still be written to stderr,
// but file logging will be skipped. This is acceptable behavior for fatal errors
// as the primary goal is to inform the user via stderr output.
func Fatal(err error) {
	if err != nil {
		// Log to file if logger is initialized (logger.Error handles nil check internally)
		logger.Error("Command execution failed", "error", err)
		// Always write to stderr regardless of logger state
		fmt.Fprintf(os.Stderr, "%s\n", Format(err))
		os.Exit(1)
	}
}

// Fatalf logs and formats an error message, then exits the program with exit code 1.
// Note: This function depends on the logger being initialized via logger.Init().
// If the logger is not initialized, the error will still be written to stderr,
// but file logging will be skipped. This is acceptable behavior for fatal errors
// as the primary goal is to inform the user via stderr output.
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// Log to file if logger is initialized (logger.Error handles nil check internally)
	logger.Error("Command execution failed", "error", msg)
	// Always write to stderr regardless of logger state
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}
