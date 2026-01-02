package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"time"

	pq "github.com/lib/pq"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/logger"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/migrations"
)

type Store struct {
	connStr string
	db      *sql.DB
}

var (
	ErrInvalidConnectionString = errors.New("invalid PostgreSQL connection string")
	ErrEmbeddedCredentials     = errors.New("connection string must not contain a password")
)

func New(connStr string) *Store {
	s := &Store{
		connStr: connStr,
	}
	s.ensureSearchPath()
	return s
}

func (s *Store) ensureSearchPath() {
	// Ensure search_path is set to daylit in the connection string
	if strings.HasPrefix(s.connStr, "postgres://") || strings.HasPrefix(s.connStr, "postgresql://") {
		u, err := url.Parse(s.connStr)
		if err != nil {
			logger.Warn("Failed to parse Postgres connection string", "connStr", s.connStr, "error", err)
			return
		}
		q := u.Query()
		// Only set search_path if it's not already present
		if q.Get("search_path") == "" {
			q.Set("search_path", constants.AppName)
			u.RawQuery = q.Encode()
			s.connStr = u.String()
		}
	} else {
		// Assume DSN format - only append if search_path is not already present
		if !hasSearchPathParam(s.connStr) {
			s.connStr = strings.TrimSpace(s.connStr) + " search_path=" + constants.AppName
		}
	}
}

// hasSearchPathParam returns true if the given DSN-style connection string
// contains a search_path parameter key (case-insensitive).
func hasSearchPathParam(connStr string) bool {
	// DSN format is typically space-separated key=value pairs.
	parts := strings.Fields(connStr)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "search_path") {
			return true
		}
	}
	return false
}

// hasSSLMode checks if the connection string contains an sslmode parameter key (case-insensitive).
// It supports both URL-style and DSN-style connection strings.
func hasSSLMode(connStr string) bool {
	// First, try to interpret the connection string as a URL (e.g. postgres://...?sslmode=disable).
	if u, err := url.Parse(connStr); err == nil && u.Scheme != "" {
		q := u.Query()
		for key := range q {
			if strings.EqualFold(key, "sslmode") {
				return true
			}
		}
	}

	// Fallback: treat the connection string as DSN-style space-separated key=value pairs.
	parts := strings.Fields(connStr)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "sslmode") {
			return true
		}
	}

	return false
}

// ValidateConnString checks if a connection string is a valid
// PostgreSQL connection string (URI or DSN) and ensures it does not
// contain a password.
//
// It returns true if the connection string is valid and contains no password.
// Otherwise, it returns false and an error describing the issue.
func ValidateConnString(connStr string) (bool, error) {
	if strings.TrimSpace(connStr) == "" {
		return false, fmt.Errorf("%w: connection string cannot be empty", ErrInvalidConnectionString)
	}

	_, err := pq.NewConnector(connStr)
	if err != nil {
		return false, fmt.Errorf("%w: invalid connection string format: %v", ErrInvalidConnectionString, err)
	}

	if strings.HasPrefix(connStr, "postgres://") || strings.HasPrefix(connStr, "postgresql://") {
		parsedURL, err := url.Parse(connStr)
		if err != nil {
			return false, fmt.Errorf("%w: failed to parse connection URL: %v", ErrInvalidConnectionString, err)
		}

		if _, isSet := parsedURL.User.Password(); isSet {
			return false, ErrEmbeddedCredentials
		}

		if parsedURL.Host == "" && parsedURL.User == nil && (parsedURL.Path == "" || parsedURL.Path == "/") {
			return false, fmt.Errorf("%w: connection URL is incomplete", ErrInvalidConnectionString)
		}
	} else {
		pairs := strings.Fields(connStr)
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 && strings.ToLower(strings.TrimSpace(parts[0])) == "password" {
				return false, ErrEmbeddedCredentials
			}
		}
	}

	return true, nil
}

func (s *Store) Init() error {
	// Open database connection
	db, err := sql.Open("postgres", s.connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool parameters to avoid connection exhaustion
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create schema if it doesn't exist (before assigning to s.db to maintain consistency)
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + constants.AppName); err != nil {
		db.Close()
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Assign to s.db only after schema creation succeeds
	s.db = db

	// Test connection
	if err := s.db.Ping(); err != nil {
		if strings.Contains(err.Error(), "SSL is not enabled on the server") && !hasSSLMode(s.connStr) {
			return fmt.Errorf("failed to connect to database: %w (hint: try adding ?sslmode=disable to your connection string)", err)
		}
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize default settings if not present
	if _, err := s.GetSettings(); err != nil {
		defaultSettings := storage.Settings{
			DayStart:                   constants.DefaultDayStart,
			DayEnd:                     constants.DefaultDayEnd,
			DefaultBlockMin:            constants.DefaultBlockMin,
			NotificationsEnabled:       constants.DefaultNotificationsEnabled,
			NotifyBlockStart:           constants.DefaultNotifyBlockStart,
			NotifyBlockEnd:             constants.DefaultNotifyBlockEnd,
			BlockStartOffsetMin:        constants.DefaultBlockStartOffsetMin,
			BlockEndOffsetMin:          constants.DefaultBlockEndOffsetMin,
			NotificationGracePeriodMin: constants.DefaultNotificationGracePeriodMin,
		}
		if err := s.SaveSettings(defaultSettings); err != nil {
			return fmt.Errorf("failed to save default settings: %w", err)
		}
	}

	return nil
}

func (s *Store) Load() error {
	if s.db != nil {
		return nil
	}

	db, err := sql.Open("postgres", s.connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Configure connection pool parameters to avoid connection exhaustion
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := s.db.Ping(); err != nil {
		if strings.Contains(err.Error(), "SSL is not enabled on the server") && !hasSSLMode(s.connStr) {
			return fmt.Errorf("failed to connect to database: %w (hint: try adding ?sslmode=disable to your connection string)", err)
		}
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Validate schema version using embedded migrations
	if err := s.validateSchemaVersion(); err != nil {
		return err
	}

	return nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) runMigrations() error {
	// Get the embedded PostgreSQL migrations sub-filesystem
	subFS, err := fs.Sub(migrations.FS, "postgres")
	if err != nil {
		return fmt.Errorf("failed to access postgres migrations: %w", err)
	}

	runner := migration.NewRunner(s.db, subFS)
	_, err = runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})
	return err
}

func (s *Store) validateSchemaVersion() error {
	subFS, err := fs.Sub(migrations.FS, "postgres")
	if err != nil {
		return fmt.Errorf("failed to access postgres migrations: %w", err)
	}

	runner := migration.NewRunner(s.db, subFS)
	return runner.ValidateVersion()
}

func (s *Store) GetConfigPath() string {
	// Return a non-sensitive identifier instead of the full connection string
	return "postgresql"
}
