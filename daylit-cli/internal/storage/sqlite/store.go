package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/migrations"
)

type Store struct {
	path string
	db   *sql.DB
}

func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

func (s *Store) Init() error {
	// Create config directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize default settings if not present or incomplete
	settings, err := s.GetSettings()
	if err != nil || settings.DayStart == "" {
		defaultSettings := models.Settings{
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

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return fmt.Errorf("storage not initialized, run 'daylit init' first")
	}

	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

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

// tableExists checks if a table exists in the SQLite database.
// Returns true if the table exists, false otherwise. Returns an error if the check itself fails.
// The check is case-insensitive to match SQLite's behavior.
func (s *Store) tableExists(tableName string) (bool, error) {
	var count int
	row := s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name COLLATE NOCASE = ?", tableName)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) runMigrations() error {
	// Get the embedded SQLite migrations sub-filesystem
	subFS, err := fs.Sub(migrations.FS, "sqlite")
	if err != nil {
		return fmt.Errorf("failed to access sqlite migrations: %w", err)
	}

	// Create migration runner
	runner := migration.NewRunner(s.db, subFS)

	// Apply all pending migrations
	_, err = runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})
	return err
}

func (s *Store) validateSchemaVersion() error {
	subFS, err := fs.Sub(migrations.FS, "sqlite")
	if err != nil {
		return fmt.Errorf("failed to access sqlite migrations: %w", err)
	}

	runner := migration.NewRunner(s.db, subFS)
	return runner.ValidateVersion()
}

func (s *Store) GetConfigPath() string {
	return s.path
}

// GetDB returns the underlying database connection.
// Returns nil if the database has not been initialized or loaded.
// Callers should use Load() before calling this method.
func (s *Store) GetDB() *sql.DB {
	return s.db
}
