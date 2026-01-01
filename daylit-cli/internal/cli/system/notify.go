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
	const maxRetries = 3
	const retryDelay = 100 * time.Millisecond

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = c.runWithRetry(ctx)
		if err == nil {
			return nil
		}
		// Check if it's a database lock error
		if attempt < maxRetries-1 && isDatabaseBusyError(err) {
			time.Sleep(retryDelay * time.Duration(attempt+1))
			continue
		}
		break
	}
	return err
}

func isDatabaseBusyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for SQLite busy/locked errors
	if len(errStr) >= 16 && errStr[:16] == "database is lock" {
		return true // "database is locked"
	}
	if len(errStr) >= 13 && errStr[:13] == "database busy" {
		return true
	}
	if len(errStr) >= 20 && errStr[:20] == "database table is lo" {
		return true // "database table is locked"
	}
	return false
}

func (c *NotifyCmd) runWithRetry(ctx *cli.Context) error {
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
		// Only notify for accepted or done slots
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
			if err := c.checkAndSendStartNotification(
				ctx, &slot, taskName, startMinutes, currentMinutes, now,
				settings.BlockStartOffsetMin, settings.NotificationGracePeriodMin,
				plan.Date, plan.Revision, n,
			); err != nil {
				return err
			}
		}

		// Check End Notification
		if settings.NotifyBlockEnd {
			if err := c.checkAndSendEndNotification(
				ctx, &slot, taskName, endMinutes, currentMinutes, now,
				settings.BlockEndOffsetMin, settings.NotificationGracePeriodMin,
				plan.Date, plan.Revision, n,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *NotifyCmd) checkAndSendStartNotification(
	ctx *cli.Context,
	slot *models.Slot,
	taskName string,
	startMinutes, currentMinutes int,
	now time.Time,
	offsetMin, gracePeriodMin int,
	planDate string,
	planRevision int,
	n *notifier.Notifier,
) error {
	triggerTime := startMinutes - offsetMin

	// Check if we've already notified
	if slot.LastNotifiedStart != nil {
		// Already notified, skip
		return nil
	}

	// Check if current time is past the trigger time
	if currentMinutes < triggerTime {
		// Not time yet
		return nil
	}

	// Calculate how late we are
	minutesLate := currentMinutes - triggerTime

	// If we're too late (beyond grace period), skip
	if minutesLate > gracePeriodMin {
		return nil
	}

	// Build notification message
	var msg string
	if minutesLate == 0 {
		// On time
		if offsetMin == 0 {
			msg = fmt.Sprintf("Starting now: %s (%s)", taskName, slot.Start)
		} else {
			msg = fmt.Sprintf("Upcoming: %s starts in %d min (%s)", taskName, offsetMin, slot.Start)
		}
	} else {
		// Late notification
		if offsetMin == 0 {
			msg = fmt.Sprintf("Started %d min ago: %s (%s)", minutesLate, taskName, slot.Start)
		} else {
			actualMinutesAgo := minutesLate - offsetMin
			if actualMinutesAgo > 0 {
				msg = fmt.Sprintf("Started %d min ago: %s (%s)", actualMinutesAgo, taskName, slot.Start)
			} else {
				// Still in the "upcoming" window
				minutesUntilStart := -actualMinutesAgo
				msg = fmt.Sprintf("Upcoming: %s starts in %d min (%s)", taskName, minutesUntilStart, slot.Start)
			}
		}
	}

	// Send notification
	if c.DryRun {
		fmt.Println("[DryRun] " + msg)
	} else {
		if err := n.Notify(msg); err != nil {
			// Log error but continue
			fmt.Printf("Failed to send notification: %v\n", err)
		}
	}

	// Update notification timestamp
	timestamp := now.Format(time.RFC3339)
	if err := ctx.Store.UpdateSlotNotificationTimestamp(planDate, planRevision, slot.Start, slot.TaskID, "start", timestamp); err != nil {
		return fmt.Errorf("failed to update notification timestamp: %w", err)
	}

	return nil
}

func (c *NotifyCmd) checkAndSendEndNotification(
	ctx *cli.Context,
	slot *models.Slot,
	taskName string,
	endMinutes, currentMinutes int,
	now time.Time,
	offsetMin, gracePeriodMin int,
	planDate string,
	planRevision int,
	n *notifier.Notifier,
) error {
	triggerTime := endMinutes - offsetMin

	// Check if we've already notified
	if slot.LastNotifiedEnd != nil {
		// Already notified, skip
		return nil
	}

	// Check if current time is past the trigger time
	if currentMinutes < triggerTime {
		// Not time yet
		return nil
	}

	// Calculate how late we are
	minutesLate := currentMinutes - triggerTime

	// If we're too late (beyond grace period), skip
	if minutesLate > gracePeriodMin {
		return nil
	}

	// Build notification message
	var msg string
	if minutesLate == 0 {
		// On time
		if offsetMin == 0 {
			msg = fmt.Sprintf("Ending now: %s (%s)", taskName, slot.End)
		} else {
			msg = fmt.Sprintf("Ending soon: %s ends in %d min (%s)", taskName, offsetMin, slot.End)
		}
	} else {
		// Late notification
		if offsetMin == 0 {
			msg = fmt.Sprintf("Ended %d min ago: %s (%s)", minutesLate, taskName, slot.End)
		} else {
			actualMinutesAgo := minutesLate - offsetMin
			if actualMinutesAgo > 0 {
				msg = fmt.Sprintf("Ended %d min ago: %s (%s)", actualMinutesAgo, taskName, slot.End)
			} else {
				// Still in the "ending soon" window
				minutesUntilEnd := -actualMinutesAgo
				msg = fmt.Sprintf("Ending soon: %s ends in %d min (%s)", taskName, minutesUntilEnd, slot.End)
			}
		}
	}

	// Send notification
	if c.DryRun {
		fmt.Println("[DryRun] " + msg)
	} else {
		if err := n.Notify(msg); err != nil {
			// Log error but continue
			fmt.Printf("Failed to send notification: %v\n", err)
		}
	}

	// Update notification timestamp
	timestamp := now.Format(time.RFC3339)
	if err := ctx.Store.UpdateSlotNotificationTimestamp(planDate, planRevision, slot.Start, slot.TaskID, "end", timestamp); err != nil {
		return fmt.Errorf("failed to update notification timestamp: %w", err)
	}

	return nil
}
