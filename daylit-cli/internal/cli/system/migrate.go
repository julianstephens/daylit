package system

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

type MigrateCmd struct{}

func (c *MigrateCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}
	defer ctx.Store.Close()

	// Get database connection for SQLite stores
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return fmt.Errorf("migrate command only supports SQLite storage")
	}

	// Get the migrations path
	migrationsPath := sqliteStore.GetMigrationsPath()

	// Get database connection
	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Create migration runner
	runner := migration.NewRunner(db, migrationsPath, migration.DriverSQLite)

	// Apply migrations
	count, err := runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if count == 0 {
		fmt.Println("No migrations to apply. Database is up to date.")
	} else {
		fmt.Printf("\nSuccessfully applied %d migration(s).\n", count)
	}

	return nil
}
