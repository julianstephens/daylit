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
}

func (c *SettingsCmd) Run(ctx *cli.Context) error {
	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if c.List {
		fmt.Println("Current Settings:")
		fmt.Printf("  Day Start:             %s\n", settings.DayStart)
		fmt.Printf("  Day End:               %s\n", settings.DayEnd)
		fmt.Printf("  Default Block Min:     %d\n", settings.DefaultBlockMin)
		fmt.Println("\nNotification Settings:")
		fmt.Printf("  Notifications Enabled: %v\n", settings.NotificationsEnabled)
		fmt.Printf("  Notify Block Start:    %v\n", settings.NotifyBlockStart)
		fmt.Printf("  Notify Block End:      %v\n", settings.NotifyBlockEnd)
		fmt.Printf("  Block Start Offset:    %d min\n", settings.BlockStartOffsetMin)
		fmt.Printf("  Block End Offset:      %d min\n", settings.BlockEndOffsetMin)
		return nil
	}

	updated := false
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

	if updated {
		if err := ctx.Store.SaveSettings(settings); err != nil {
			return fmt.Errorf("failed to save settings: %w", err)
		}
		fmt.Println("Settings updated successfully.")
	} else {
		fmt.Println("No changes specified. Use --list to view settings or flags to update them.")
	}

	return nil
}
