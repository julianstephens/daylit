package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

// CalculateSlotDuration calculates the duration of a slot in minutes
func CalculateSlotDuration(slot models.Slot) int {
	start, err := time.Parse(constants.TimeFormat, slot.Start)
	if err != nil {
		return 0
	}
	end, err := time.Parse(constants.TimeFormat, slot.End)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

// NewEditForm creates a new form for editing tasks
func NewEditForm(fm *state.TaskFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Value(&fm.Name),
			huh.NewInput().
				Title("Duration (min)").
				Value(&fm.Duration).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("duration must be a positive number of minutes")
					}
					return nil
				}),
			huh.NewSelect[constants.RecurrenceType]().
				Title("Recurrence").
				Options(
					huh.NewOption("Ad-hoc", constants.RecurrenceAdHoc),
					huh.NewOption("Daily", constants.RecurrenceDaily),
					huh.NewOption("Weekly", constants.RecurrenceWeekly),
					huh.NewOption("Every N Days", constants.RecurrenceNDays),
				).
				Value(&fm.Recurrence),
			huh.NewInput().
				Title("Interval (days)").
				Description("For 'Every N Days' recurrence").
				Value(&fm.Interval).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("interval must be a positive number of days")
					}
					return nil
				}),
			huh.NewInput().
				Title("Priority (1-5)").
				Value(&fm.Priority).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i < 1 || i > 5 {
						return fmt.Errorf("priority must be 1-5")
					}
					return nil
				}),
			huh.NewConfirm().
				Title("Active").
				Value(&fm.Active),
		),
	).WithTheme(huh.ThemeDracula())
}

// NewHabitForm creates a new form for adding habits
func NewHabitForm(fm *state.HabitFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Habit Name").
				Value(&fm.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("habit name cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeDracula())
}

// NewAlertForm creates a new form for adding alerts
func NewAlertForm(fm *state.AlertFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Message").
				Value(&fm.Message).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("message cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Time (HH:MM)").
				Value(&fm.Time).
				Validate(func(s string) error {
					_, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					return nil
				}),
			huh.NewInput().
				Title("Date (YYYY-MM-DD)").
				Description("Leave empty for recurring alert").
				Value(&fm.Date).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					_, err := time.Parse("2006-01-02", s)
					if err != nil {
						return fmt.Errorf("invalid date format, use YYYY-MM-DD")
					}
					return nil
				}),
			huh.NewSelect[constants.RecurrenceType]().
				Title("Recurrence").
				Description("Only for recurring alerts (no date)").
				Options(
					huh.NewOption("Daily", constants.RecurrenceDaily),
					huh.NewOption("Weekly", constants.RecurrenceWeekly),
					huh.NewOption("Every N Days", constants.RecurrenceNDays),
				).
				Value(&fm.Recurrence).
				Validate(func(r constants.RecurrenceType) error {
					// When Date is empty (recurring alert), a recurrence type must be selected
					if strings.TrimSpace(fm.Date) == "" && r == "" {
						return fmt.Errorf("recurrence is required when date is empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Interval (days)").
				Description("For 'Every N Days' recurrence").
				Value(&fm.Interval).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("interval must be a positive number of days")
					}
					return nil
				}),
			huh.NewInput().
				Title("Weekdays").
				Description("For weekly: comma-separated (mon,wed,fri)").
				Value(&fm.Weekdays),
		),
	).WithTheme(huh.ThemeDracula())
}

// NewSettingsForm creates a new form for editing settings
func NewSettingsForm(fm *state.SettingsFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Day Start (HH:MM)").
				Value(&fm.DayStart).
				Validate(func(s string) error {
					_, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					return nil
				}),
			huh.NewInput().
				Title("Day End (HH:MM)").
				Value(&fm.DayEnd).
				Validate(func(s string) error {
					endTime, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					// Cross-field validation: ensure Day End is after Day Start
					startTime, err := time.Parse(constants.TimeFormat, fm.DayStart)
					if err == nil && !endTime.After(startTime) {
						return fmt.Errorf("day end must be after day start")
					}
					return nil
				}),
			huh.NewInput().
				Title("Default Block (minutes)").
				Value(&fm.DefaultBlockMin).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),
			huh.NewInput().
				Title("Timezone (IANA name or 'Local')").
				Description("Examples: Local, UTC, America/New_York, Europe/London, Asia/Tokyo").
				Value(&fm.Timezone).
				Validate(func(s string) error {
					if !utils.ValidateTimezone(s) {
						return fmt.Errorf("invalid timezone name")
					}
					return nil
				}),
			huh.NewConfirm().
				Title("Prompt On Empty").
				Value(&fm.PromptOnEmpty),
			huh.NewConfirm().
				Title("Strict Mode").
				Value(&fm.StrictMode),
			huh.NewInput().
				Title("Default Log Days").
				Value(&fm.DefaultLogDays).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i < 0 {
						return fmt.Errorf("must be a non-negative number")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Notifications").
				Value(&fm.NotificationsEnabled),
			huh.NewConfirm().
				Title("Notify on Block Start").
				Value(&fm.NotifyBlockStart),
			huh.NewInput().
				Title("Start Offset (minutes)").
				Value(&fm.BlockStartOffsetMin).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					return err
				}),
			huh.NewConfirm().
				Title("Notify on Block End").
				Value(&fm.NotifyBlockEnd),
			huh.NewInput().
				Title("End Offset (minutes)").
				Value(&fm.BlockEndOffsetMin).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					return err
				}),
		),
	).WithTheme(huh.ThemeDracula())
}

// NewOTForm creates a new form for editing One Thing
func NewOTForm(fm *state.OTFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("One Thing Title").
				Value(&fm.Title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("title cannot be empty")
					}
					return nil
				}),
			huh.NewText().
				Title("Note (optional)").
				Value(&fm.Note),
		),
	).WithTheme(huh.ThemeDracula())
}
