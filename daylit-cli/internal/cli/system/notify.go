package system

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/notifier"
)

type NotifyCmd struct {
	DryRun bool `help:"Print notifications to stdout instead of sending them."`
}

func (c *NotifyCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if !settings.NotificationsEnabled {
		if c.DryRun {
			fmt.Println("Notifications are disabled in settings.")
		}
		return nil
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	currentMinutes := now.Hour()*60 + now.Minute()

	// Get the latest plan for today
	plan, err := ctx.Store.GetLatestPlanRevision(dateStr)
	if err != nil {
		// No plan for today, nothing to notify
		if c.DryRun {
			fmt.Println("No plan found for today.")
		}
		return nil
	}

	n := notifier.New()

	for _, slot := range plan.Slots {
		// Only notify for accepted or done slots (though usually we notify before they are done)
		// If a slot is "suggested" it might not be confirmed yet, but maybe we should still notify?
		// Let's stick to Accepted/Done for now as "active" parts of the plan.
		if slot.Status != models.SlotStatusAccepted && slot.Status != models.SlotStatusDone {
			continue
		}

		startMinutes, err := cli.ParseTimeToMinutes(slot.Start)
		if err != nil {
			continue
		}
		endMinutes, err := cli.ParseTimeToMinutes(slot.End)
		if err != nil {
			continue
		}

		taskName := "Unknown Task"
		if task, err := ctx.Store.GetTask(slot.TaskID); err == nil {
			taskName = task.Name
		}

		// Check Start Notification
		if settings.NotifyBlockStart {
			triggerTime := startMinutes - settings.BlockStartOffsetMin
			if currentMinutes == triggerTime {
				var msg string
				if settings.BlockStartOffsetMin == 0 {
					msg = fmt.Sprintf("Starting now: %s (%s)", taskName, slot.Start)
				} else {
					msg = fmt.Sprintf("Upcoming: %s starts in %d min (%s)", taskName, settings.BlockStartOffsetMin, slot.Start)
				}

				if c.DryRun {
					fmt.Println("[DryRun] " + msg)
				} else {
					if err := n.Notify(msg); err != nil {
						// Log error but continue checking other slots
						fmt.Printf("Failed to send notification: %v\n", err)
					}
				}
			}
		}

		// Check End Notification
		if settings.NotifyBlockEnd {
			triggerTime := endMinutes - settings.BlockEndOffsetMin
			if currentMinutes == triggerTime {
				var msg string
				if settings.BlockEndOffsetMin == 0 {
					msg = fmt.Sprintf("Ending now: %s (%s)", taskName, slot.End)
				} else {
					msg = fmt.Sprintf("Ending soon: %s ends in %d min (%s)", taskName, settings.BlockEndOffsetMin, slot.End)
				}

				if c.DryRun {
					fmt.Println("[DryRun] " + msg)
				} else {
					if err := n.Notify(msg); err != nil {
						fmt.Printf("Failed to send notification: %v\n", err)
					}
				}
			}
		}
	}

	return nil
}
