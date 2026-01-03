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

// Fatal logs an error and exits the program with exit code 1
func Fatal(err error) {
	if err != nil {
		logger.Error("Command execution failed", "error", err)
		fmt.Fprintf(os.Stderr, "%s\n", Format(err))
		os.Exit(1)
	}
}

// Fatalf logs and formats an error message, then exits the program with exit code 1
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Error("Command execution failed", "error", msg)
	fmt.Fprintf(os.Stderr, "%s\n", Formatf(format, args...))
	os.Exit(1)
}
