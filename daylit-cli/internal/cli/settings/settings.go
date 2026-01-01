package settings

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type SettingsCmd struct {
	List bool `help:"List current settings."`

	NotificationsEnabled *bool `help:"Enable or disable notifications."`
	NotifyBlockStart     *bool `help:"Notify on block start."`
	NotifyBlockEnd       *bool `help:"Notify on block end."`
	BlockStartOffsetMin  *int  `help:"Minutes before block start to notify."`
	BlockEndOffsetMin    *int  `help:"Minutes before block end to notify."`

	OTPromptOnEmpty  *bool `help:"OT: Prompt when no entry exists for today."`
	OTStrictMode     *bool `help:"OT: Strict mode - only one entry per day."`
	OTDefaultLogDays *int  `help:"OT: Default number of days to show in log view."`
}

func (c *SettingsCmd) Run(ctx *cli.Context) error {
	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	otSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		return fmt.Errorf("failed to get OT settings: %w", err)
	}

	if c.List {
		fmt.Println("Current Settings:")
		fmt.Printf("  Day Start:             %s\n", settings.DayStart)
		fmt.Printf("  Day End:               %s\n", settings.DayEnd)
		fmt.Printf("  Default Block Min:     %d\n", settings.DefaultBlockMin)
		fmt.Println("\nOnce Today (OT) Settings:")
		fmt.Printf("  Prompt On Empty:       %v\n", otSettings.PromptOnEmpty)
		fmt.Printf("  Strict Mode:           %v\n", otSettings.StrictMode)
		fmt.Printf("  Default Log Days:      %d\n", otSettings.DefaultLogDays)
		fmt.Println("\nNotification Settings:")
		fmt.Printf("  Notifications Enabled: %v\n", settings.NotificationsEnabled)
		fmt.Printf("  Notify Block Start:    %v\n", settings.NotifyBlockStart)
		fmt.Printf("  Notify Block End:      %v\n", settings.NotifyBlockEnd)
		fmt.Printf("  Block Start Offset:    %d min\n", settings.BlockStartOffsetMin)
		fmt.Printf("  Block End Offset:      %d min\n", settings.BlockEndOffsetMin)
		return nil
	}

	updated := false
	otUpdated := false

	if c.NotificationsEnabled != nil {
		settings.NotificationsEnabled = *c.NotificationsEnabled
		updated = true
	}
	if c.NotifyBlockStart != nil {
		settings.NotifyBlockStart = *c.NotifyBlockStart
		updated = true
	}
	if c.NotifyBlockEnd != nil {
		settings.NotifyBlockEnd = *c.NotifyBlockEnd
		updated = true
	}
	if c.BlockStartOffsetMin != nil {
		settings.BlockStartOffsetMin = *c.BlockStartOffsetMin
		updated = true
	}
	if c.BlockEndOffsetMin != nil {
		settings.BlockEndOffsetMin = *c.BlockEndOffsetMin
		updated = true
	}

	if c.OTPromptOnEmpty != nil {
		otSettings.PromptOnEmpty = *c.OTPromptOnEmpty
		otUpdated = true
	}
	if c.OTStrictMode != nil {
		otSettings.StrictMode = *c.OTStrictMode
		otUpdated = true
	}
	if c.OTDefaultLogDays != nil {
		if *c.OTDefaultLogDays < 1 {
			return fmt.Errorf("OTDefaultLogDays must be at least 1")
		}
		otSettings.DefaultLogDays = *c.OTDefaultLogDays
		otUpdated = true
	}

	if updated {
		if err := ctx.Store.SaveSettings(settings); err != nil {
			return fmt.Errorf("failed to save settings: %w", err)
		}
	}

	if otUpdated {
		if err := ctx.Store.SaveOTSettings(otSettings); err != nil {
			return fmt.Errorf("failed to save OT settings: %w", err)
		}
	}

	if updated || otUpdated {
		fmt.Println("Settings updated successfully.")
	} else {
		fmt.Println("No changes specified. Use --list to view settings or flags to update them.")
	}

	return nil
}
