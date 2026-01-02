package system

import (
	"fmt"
	"io/fs"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/migrations"
)

type MigrateCmd struct{}

func (c *MigrateCmd) Run(ctx *cli.Context) error {
	defer ctx.Store.Close()

	// Get database connection for SQLite stores
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return fmt.Errorf("migrate command only supports SQLite storage")
	}

	// Get the embedded SQLite migrations sub-filesystem
	subFS, err := fs.Sub(migrations.FS, "sqlite")
	if err != nil {
		return fmt.Errorf("failed to access sqlite migrations: %w", err)
	}

	// Get database connection
	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Create migration runner
	runner := migration.NewRunner(db, subFS)

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
