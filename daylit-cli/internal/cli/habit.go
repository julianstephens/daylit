package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

type HabitCmd struct {
	Add     HabitAddCmd     `cmd:"" help:"Add a new habit."`
	List    HabitListCmd    `cmd:"" help:"List habits."`
	Mark    HabitMarkCmd    `cmd:"" help:"Mark a habit as done for a day."`
	Today   HabitTodayCmd   `cmd:"" help:"Show today's habit status."`
	Log     HabitLogCmd     `cmd:"" help:"Show habit log (ASCII history)."`
	Archive HabitArchiveCmd `cmd:"" help:"Archive a habit."`
	Delete  HabitDeleteCmd  `cmd:"" help:"Delete a habit (soft delete)."`
	Restore HabitRestoreCmd `cmd:"" help:"Restore a deleted habit."`
}

type HabitAddCmd struct {
	Name string `arg:"" help:"Habit name."`
}

func (c *HabitAddCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Check if habit with same name already exists
	_, err := ctx.Store.GetHabitByName(c.Name)
	if err == nil {
		return fmt.Errorf("habit with name %q already exists", c.Name)
	}

	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      c.Name,
		CreatedAt: time.Now(),
	}

	if err := ctx.Store.AddHabit(habit); err != nil {
		return err
	}

	fmt.Printf("Added habit: %s\n", c.Name)
	return nil
}

type HabitListCmd struct {
	Archived bool `help:"Include archived habits."`
	Deleted  bool `help:"Include deleted habits."`
}

func (c *HabitListCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	habits, err := ctx.Store.GetAllHabits(c.Archived, c.Deleted)
	if err != nil {
		return err
	}

	if len(habits) == 0 {
		fmt.Println("No habits found.")
		return nil
	}

	for _, habit := range habits {
		status := ""
		if habit.DeletedAt != nil {
			status = " [DELETED]"
		} else if habit.ArchivedAt != nil {
			status = " [ARCHIVED]"
		}
		fmt.Printf("%s%s\n", habit.Name, status)
	}

	return nil
}

type HabitMarkCmd struct {
	Name string `arg:"" help:"Habit name."`
	Date string `help:"Date in YYYY-MM-DD format (default: today)." default:""`
	Note string `help:"Optional note for this entry." default:""`
}

func (c *HabitMarkCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Get the habit
	habit, err := ctx.Store.GetHabitByName(c.Name)
	if err != nil {
		return fmt.Errorf("habit %q not found", c.Name)
	}

	// Determine the date
	day := c.Date
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else {
		// Validate date format
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", day)
		}
	}

	// Check if entry already exists
	existingEntry, err := ctx.Store.GetHabitEntry(habit.ID, day)
	if err == nil {
		// Entry exists, delete it (toggle off)
		if err := ctx.Store.DeleteHabitEntry(existingEntry.ID); err != nil {
			return err
		}
		fmt.Printf("Unmarked habit %q for %s\n", c.Name, day)
		return nil
	}

	// Entry doesn't exist, create it (toggle on)
	entry := models.HabitEntry{
		ID:        uuid.New().String(),
		HabitID:   habit.ID,
		Day:       day,
		Note:      c.Note,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ctx.Store.AddHabitEntry(entry); err != nil {
		return err
	}

	fmt.Printf("Marked habit %q for %s\n", c.Name, day)
	return nil
}

type HabitTodayCmd struct{}

func (c *HabitTodayCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	habits, err := ctx.Store.GetAllHabits(false, false)
	if err != nil {
		return err
	}

	if len(habits) == 0 {
		fmt.Println("No habits found.")
		return nil
	}

	today := time.Now().Format("2006-01-02")
	entries, err := ctx.Store.GetHabitEntriesForDay(today)
	if err != nil {
		return err
	}

	// Create a map of habit IDs that have entries today
	entryMap := make(map[string]bool)
	for _, entry := range entries {
		entryMap[entry.HabitID] = true
	}

	fmt.Printf("Habits for %s:\n\n", today)
	recorded := 0
	for _, habit := range habits {
		if habit.ArchivedAt != nil {
			continue
		}
		status := "[ ]"
		if entryMap[habit.ID] {
			status = "[x]"
			recorded++
		}
		fmt.Printf("%s %s\n", status, habit.Name)
	}

	activeCount := 0
	for _, habit := range habits {
		if habit.ArchivedAt == nil {
			activeCount++
		}
	}

	fmt.Printf("\nRecorded: %d/%d\n", recorded, activeCount)
	return nil
}

type HabitLogCmd struct {
	Days  int    `help:"Number of days to show." default:"14"`
	Habit string `help:"Show log for specific habit only."`
}

func (c *HabitLogCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	habits, err := ctx.Store.GetAllHabits(false, false)
	if err != nil {
		return err
	}

	if len(habits) == 0 {
		fmt.Println("No habits found.")
		return nil
	}

	// Filter by habit name if specified
	var selectedHabits []models.Habit
	if c.Habit != "" {
		for _, h := range habits {
			if h.Name == c.Habit {
				selectedHabits = []models.Habit{h}
				break
			}
		}
		if len(selectedHabits) == 0 {
			return fmt.Errorf("habit %q not found", c.Habit)
		}
	} else {
		// Only show active habits
		for _, h := range habits {
			if h.ArchivedAt == nil {
				selectedHabits = append(selectedHabits, h)
			}
		}
	}

	// Calculate date range
	endDay := time.Now()
	startDay := endDay.AddDate(0, 0, -(c.Days - 1))

	// Get entries for each habit
	fmt.Printf("Habit log (last %d days):\n\n", c.Days)

	// Print header with dates
	fmt.Print("Habit               ")
	maxNameLen := 20
	for i := 0; i < c.Days; i++ {
		day := startDay.AddDate(0, 0, i)
		fmt.Printf(" %5s", day.Format("01/02"))
	}
	fmt.Println()

	// Print separator
	fmt.Print(strings.Repeat("-", maxNameLen))
	for i := 0; i < c.Days; i++ {
		fmt.Print("------")
	}
	fmt.Println()

	// Print each habit's log
	for _, habit := range selectedHabits {
		// Truncate or pad habit name
		name := habit.Name
		if len(name) > maxNameLen {
			// Ensure we keep at least 1 character of the name visible
			if maxNameLen >= 5 {
				name = name[:maxNameLen-3] + "..."
			} else if maxNameLen > 0 {
				// For very small widths, just truncate without ellipsis
				name = name[:maxNameLen]
			}
		} else {
			name = name + strings.Repeat(" ", maxNameLen-len(name))
		}
		fmt.Print(name)

		// Get entries for this habit
		entries, err := ctx.Store.GetHabitEntriesForHabit(
			habit.ID,
			startDay.Format("2006-01-02"),
			endDay.Format("2006-01-02"),
		)
		if err != nil {
			return err
		}

		// Create a map of days with entries
		entryMap := make(map[string]bool)
		for _, entry := range entries {
			entryMap[entry.Day] = true
		}

		// Print markers for each day
		for i := 0; i < c.Days; i++ {
			day := startDay.AddDate(0, 0, i)
			dayStr := day.Format("2006-01-02")
			if entryMap[dayStr] {
				fmt.Print("  x   ")
			} else {
				fmt.Print("  .   ")
			}
		}
		fmt.Println()
	}

	return nil
}

type HabitArchiveCmd struct {
	Name      string `arg:"" help:"Habit name to archive."`
	Unarchive bool   `help:"Unarchive the habit instead."`
}

func (c *HabitArchiveCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	habit, err := ctx.Store.GetHabitByName(c.Name)
	if err != nil {
		return fmt.Errorf("habit %q not found", c.Name)
	}

	if c.Unarchive {
		if err := ctx.Store.UnarchiveHabit(habit.ID); err != nil {
			return err
		}
		fmt.Printf("Unarchived habit: %s\n", c.Name)
	} else {
		if err := ctx.Store.ArchiveHabit(habit.ID); err != nil {
			return err
		}
		fmt.Printf("Archived habit: %s\n", c.Name)
	}

	return nil
}

type HabitDeleteCmd struct {
	Name string `arg:"" help:"Habit name to delete."`
}

func (c *HabitDeleteCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	habit, err := ctx.Store.GetHabitByName(c.Name)
	if err != nil {
		return fmt.Errorf("habit %q not found", c.Name)
	}

	if err := ctx.Store.DeleteHabit(habit.ID); err != nil {
		return err
	}

	fmt.Printf("Deleted habit: %s\n", c.Name)
	fmt.Println("(This is a soft delete. Use 'daylit habit restore' to undo)")
	return nil
}

type HabitRestoreCmd struct {
	Name string `arg:"" help:"Habit name to restore."`
}

func (c *HabitRestoreCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Get habit including deleted ones
	habits, err := ctx.Store.GetAllHabits(true, true)
	if err != nil {
		return err
	}

	var habit *models.Habit
	for _, h := range habits {
		if h.Name == c.Name && h.DeletedAt != nil {
			habit = &h
			break
		}
	}

	if habit == nil {
		return fmt.Errorf("deleted habit %q not found", c.Name)
	}

	if err := ctx.Store.RestoreHabit(habit.ID); err != nil {
		return err
	}

	fmt.Printf("Restored habit: %s\n", c.Name)
	return nil
}

// Helper function to check if storage is SQLite
func isSQLiteStore(store storage.Provider) bool {
	_, ok := store.(*storage.SQLiteStore)
	return ok
}

// Ensure storage is SQLite for habit commands
func ensureSQLiteStore(ctx *Context) error {
	if !isSQLiteStore(ctx.Store) {
		return fmt.Errorf("habits are only supported with SQLite storage (not JSON)")
	}
	return nil
}

// Add validation to all habit commands
func (c *HabitAddCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitListCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitMarkCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitTodayCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitLogCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitArchiveCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitDeleteCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}

func (c *HabitRestoreCmd) Validate(ctx *Context) error {
	return ensureSQLiteStore(ctx)
}
