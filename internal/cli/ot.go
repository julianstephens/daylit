package cli

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/julianstephens/daylit/internal/models"
	"github.com/julianstephens/daylit/internal/storage"
)

type OTCmd struct {
	Init     OTInitCmd     `cmd:"" help:"Initialize OT settings."`
	Settings OTSettingsCmd `cmd:"" help:"View or update OT settings."`
	Set      OTSetCmd      `cmd:"" help:"Set today's OT intention."`
	Show     OTShowCmd     `cmd:"" help:"Show OT for a day."`
	Nudge    OTNudgeCmd    `cmd:"" help:"Show today's OT or prompt to create."`
	Doctor   OTDoctorCmd   `cmd:"" help:"Check OT data integrity."`
	Delete   OTDeleteCmd   `cmd:"" help:"Delete OT entry (soft delete)."`
	Restore  OTRestoreCmd  `cmd:"" help:"Restore deleted OT entry."`
}

type OTInitCmd struct{}

func (c *OTInitCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Check if settings already exist
	_, err := ctx.Store.GetOTSettings()
	if err == nil {
		fmt.Println("OT settings already initialized.")
		return nil
	}

	// Initialize with defaults
	settings := models.OTSettings{
		ID:             1,
		PromptOnEmpty:  true,
		StrictMode:     true,
		DefaultLogDays: 14,
	}

	if err := ctx.Store.SaveOTSettings(settings); err != nil {
		return err
	}

	fmt.Println("OT settings initialized with defaults:")
	fmt.Printf("  prompt_on_empty: %t\n", settings.PromptOnEmpty)
	fmt.Printf("  strict_mode: %t\n", settings.StrictMode)
	fmt.Printf("  default_log_days: %d\n", settings.DefaultLogDays)
	return nil
}

type OTSettingsCmd struct {
	PromptOnEmpty  *bool `help:"Enable/disable prompt when OT is empty."`
	StrictMode     *bool `help:"Enable/disable strict mode (require title)."`
	DefaultLogDays *int  `help:"Set default number of days for log view."`
}

func (c *OTSettingsCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	settings, err := ctx.Store.GetOTSettings()
	if err != nil {
		return fmt.Errorf("OT settings not found. Run 'daylit ot init' first")
	}

	// Update settings if flags provided
	changed := false
	if c.PromptOnEmpty != nil {
		settings.PromptOnEmpty = *c.PromptOnEmpty
		changed = true
	}
	if c.StrictMode != nil {
		settings.StrictMode = *c.StrictMode
		changed = true
	}
	if c.DefaultLogDays != nil {
		if *c.DefaultLogDays < 1 {
			return fmt.Errorf("default_log_days must be at least 1")
		}
		settings.DefaultLogDays = *c.DefaultLogDays
		changed = true
	}

	if changed {
		if err := ctx.Store.SaveOTSettings(settings); err != nil {
			return err
		}
		fmt.Println("OT settings updated.")
	}

	// Display current settings
	fmt.Println("Current OT settings:")
	fmt.Printf("  prompt_on_empty: %t\n", settings.PromptOnEmpty)
	fmt.Printf("  strict_mode: %t\n", settings.StrictMode)
	fmt.Printf("  default_log_days: %d\n", settings.DefaultLogDays)

	return nil
}

type OTSetCmd struct {
	Day   string `help:"Date in YYYY-MM-DD format (default: today)." default:""`
	Title string `help:"OT title/intention." required:""`
	Note  string `help:"Optional note." default:""`
}

func (c *OTSetCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Check strict mode
	settings, err := ctx.Store.GetOTSettings()
	if err != nil {
		return fmt.Errorf("OT settings not found. Run 'daylit ot init' first")
	}

	if settings.StrictMode && c.Title == "" {
		return fmt.Errorf("strict mode requires a title")
	}

	// Determine the date
	day := c.Day
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else {
		// Validate date format
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", day)
		}
	}

	// Check if entry exists for this day
	existingEntry, err := ctx.Store.GetOTEntry(day)
	if err == nil {
		// Update existing entry
		existingEntry.Title = c.Title
		existingEntry.Note = c.Note
		existingEntry.UpdatedAt = time.Now()
		if err := ctx.Store.UpdateOTEntry(existingEntry); err != nil {
			return err
		}
		fmt.Printf("Updated OT for %s\n", day)
		return nil
	}

	// Create new entry
	entry := models.OTEntry{
		ID:        uuid.New().String(),
		Day:       day,
		Title:     c.Title,
		Note:      c.Note,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ctx.Store.AddOTEntry(entry); err != nil {
		return err
	}

	fmt.Printf("Set OT for %s\n", day)
	return nil
}

type OTShowCmd struct {
	Day     string `help:"Date in YYYY-MM-DD format (default: today)." default:""`
	Deleted bool   `help:"Include deleted entries in date range."`
	Days    int    `help:"Show last N days instead of single day." default:"0"`
}

func (c *OTShowCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	if c.Days > 0 {
		// Show last N days
		endDay := time.Now()
		if c.Day != "" {
			var err error
			endDay, err = time.Parse("2006-01-02", c.Day)
			if err != nil {
				return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", c.Day)
			}
		}
		startDay := endDay.AddDate(0, 0, -(c.Days - 1))

		entries, err := ctx.Store.GetOTEntries(
			startDay.Format("2006-01-02"),
			endDay.Format("2006-01-02"),
			c.Deleted,
		)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Printf("No OT entries found for last %d days.\n", c.Days)
			return nil
		}

		fmt.Printf("OT entries (last %d days):\n\n", c.Days)
		for _, entry := range entries {
			status := ""
			if entry.DeletedAt != nil {
				status = " [DELETED]"
			}
			fmt.Printf("%s:%s\n", entry.Day, status)
			fmt.Printf("  %s\n", entry.Title)
			if entry.Note != "" {
				fmt.Printf("  Note: %s\n", entry.Note)
			}
			fmt.Println()
		}
		return nil
	}

	// Show single day
	day := c.Day
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else {
		// Validate date format
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", day)
		}
	}

	entry, err := ctx.Store.GetOTEntry(day)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("No OT set for %s\n", day)
			return nil
		}
		return err
	}

	fmt.Printf("OT for %s:\n", day)
	fmt.Printf("  %s\n", entry.Title)
	if entry.Note != "" {
		fmt.Printf("  Note: %s\n", entry.Note)
	}

	return nil
}

type OTNudgeCmd struct{}

func (c *OTNudgeCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")
	entry, err := ctx.Store.GetOTEntry(today)
	if err == nil {
		// OT exists for today
		fmt.Printf("Today's OT:\n  %s\n", entry.Title)
		if entry.Note != "" {
			fmt.Printf("  Note: %s\n", entry.Note)
		}
		return nil
	}

	// No OT for today
	settings, err := ctx.Store.GetOTSettings()
	if err != nil {
		fmt.Println("No OT set for today.")
		fmt.Println("Run 'daylit ot init' to initialize OT settings.")
		return nil
	}

	fmt.Println("No OT set for today.")
	if settings.PromptOnEmpty {
		fmt.Println("Set your Once-Today intention with: daylit ot set --title \"...\"")
	}

	return nil
}

type OTDoctorCmd struct{}

func (c *OTDoctorCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	fmt.Println("Running OT diagnostics...")
	fmt.Println()

	hasError := false

	// Check 1: OT settings exist
	_, err := ctx.Store.GetOTSettings()
	if err != nil {
		fmt.Println("❌ OT settings: MISSING")
		fmt.Println("   Run 'daylit ot init' to initialize")
		hasError = true
	} else {
		fmt.Println("✓ OT settings: OK")
	}

	// Check 2: Check for invalid dates
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		fmt.Println("⊘ Date validation: SKIPPED (not SQLite)")
	} else {
		db := sqliteStore.GetDB()
		var invalidCount int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM ot_entries
			WHERE day NOT LIKE '____-__-__'
		`).Scan(&invalidCount)
		if err != nil {
			fmt.Printf("❌ Date validation: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else if invalidCount > 0 {
			fmt.Printf("❌ Date validation: FAIL\n")
			fmt.Printf("   Found %d entries with invalid date format\n", invalidCount)
			hasError = true
		} else {
			fmt.Println("✓ Date validation: OK")
		}
	}

	// Check 3: Check for duplicate days
	if sqliteStore, ok := ctx.Store.(*storage.SQLiteStore); ok {
		db := sqliteStore.GetDB()
		var duplicateCount int
		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM (
				SELECT day, COUNT(*) as cnt
				FROM ot_entries
				WHERE deleted_at IS NULL
				GROUP BY day
				HAVING cnt > 1
			)
		`).Scan(&duplicateCount)
		if err != nil {
			fmt.Printf("❌ Duplicate days: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else if duplicateCount > 0 {
			fmt.Printf("❌ Duplicate days: FAIL\n")
			fmt.Printf("   Found %d days with multiple active entries\n", duplicateCount)
			hasError = true
		} else {
			fmt.Println("✓ Duplicate days: OK")
		}
	}

	// Check 4: Check for corrupted timestamps
	if sqliteStore, ok := ctx.Store.(*storage.SQLiteStore); ok {
		db := sqliteStore.GetDB()
		var corruptedCount int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM ot_entries
			WHERE created_at = '' OR updated_at = ''
		`).Scan(&corruptedCount)
		if err != nil {
			fmt.Printf("❌ Timestamp validation: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else if corruptedCount > 0 {
			fmt.Printf("❌ Timestamp validation: FAIL\n")
			fmt.Printf("   Found %d entries with corrupted timestamps\n", corruptedCount)
			hasError = true
		} else {
			fmt.Println("✓ Timestamp validation: OK")
		}
	}

	fmt.Println()
	if hasError {
		fmt.Println("OT diagnostics completed with errors.")
		return fmt.Errorf("one or more OT health checks failed")
	}

	fmt.Println("All OT diagnostics passed!")
	return nil
}

type OTDeleteCmd struct {
	Day string `help:"Date in YYYY-MM-DD format (default: today)." default:""`
}

func (c *OTDeleteCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	day := c.Day
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else {
		// Validate date format
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", day)
		}
	}

	if err := ctx.Store.DeleteOTEntry(day); err != nil {
		return err
	}

	fmt.Printf("Deleted OT entry for %s\n", day)
	fmt.Println("(This is a soft delete. Use 'daylit ot restore' to undo)")
	return nil
}

type OTRestoreCmd struct {
	Day string `help:"Date in YYYY-MM-DD format (default: today)." default:""`
}

func (c *OTRestoreCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	day := c.Day
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else {
		// Validate date format
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", day)
		}
	}

	if err := ctx.Store.RestoreOTEntry(day); err != nil {
		return err
	}

	fmt.Printf("Restored OT entry for %s\n", day)
	return nil
}

// Ensure storage is SQLite for OT commands
func ensureSQLiteStoreOT(ctx *Context) error {
	if _, ok := ctx.Store.(*storage.SQLiteStore); !ok {
		return fmt.Errorf("OT is only supported with SQLite storage (not JSON)")
	}
	return nil
}

// Add validation to all OT commands
func (c *OTInitCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTSettingsCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTSetCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTShowCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTNudgeCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTDoctorCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTDeleteCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}

func (c *OTRestoreCmd) Validate(ctx *Context) error {
	return ensureSQLiteStoreOT(ctx)
}
