package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Logger is the global logger instance
	Logger *log.Logger
)

// Config holds logger configuration
type Config struct {
	Debug     bool
	ConfigDir string
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	// Determine log directory
	logDir := filepath.Join(cfg.ConfigDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, "daylit.log")

	// Create rotating file handler
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	// Set log level based on debug flag
	level := log.WarnLevel
	if cfg.Debug {
		level = log.DebugLevel
	}

	// Create logger with multi-writer (file + stderr when debug enabled)
	var writer io.Writer
	if cfg.Debug {
		// In debug mode, write to both stderr and file
		writer = io.MultiWriter(os.Stderr, fileWriter)
	} else {
		// In normal mode, only write to file (silent on stderr)
		writer = fileWriter
	}

	Logger = log.NewWithOptions(writer, log.Options{
		ReportCaller:    cfg.Debug,
		ReportTimestamp: true,
		Level:           level,
		Prefix:          "daylit",
	})

	return nil
}

// Debug logs a debug message
func Debug(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Debug(msg, keyvals...)
	}
}

// Info logs an info message
func Info(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Info(msg, keyvals...)
	}
}

// Warn logs a warning message
func Warn(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Warn(msg, keyvals...)
	}
}

// Error logs an error message
func Error(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Error(msg, keyvals...)
	}
}

// Fatal logs a fatal error and exits
func Fatal(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Fatal(msg, keyvals...)
	}
	os.Exit(1)
}
