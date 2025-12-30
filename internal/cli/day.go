package cli

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

type DayCmd struct {
	Date string `arg:"" help:"Date to show (YYYY-MM-DD or 'today')." default:"today"`
}

func (c *DayCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Parse date
	var planDate time.Time
	if c.Date == "today" {
		planDate = time.Now()
	} else {
		var err error
		planDate, err = time.Parse("2006-01-02", c.Date)
		if err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD or 'today': %w", err)
		}
	}

	dateStr := planDate.Format("2006-01-02")

	plan, err := ctx.Store.GetPlan(dateStr)
	if err != nil {
		return fmt.Errorf("no plan found for %s", dateStr)
	}

	fmt.Printf("Plan for %s:\n\n", dateStr)

	if len(plan.Slots) == 0 {
		fmt.Println("  No slots scheduled")
		return nil
	}

	for _, slot := range plan.Slots {
		task, err := ctx.Store.GetTask(slot.TaskID)
		taskName := "unknown task"
		if err == nil {
			taskName = task.Name
		}

		statusStr := ""
		switch slot.Status {
		case models.SlotStatusPlanned:
			statusStr = "[planned]"
		case models.SlotStatusAccepted:
			statusStr = "[accepted]"
		case models.SlotStatusDone:
			if slot.Feedback != nil {
				statusStr = fmt.Sprintf("[done, %s]", slot.Feedback.Rating)
			} else {
				statusStr = "[done]"
			}
		case models.SlotStatusSkipped:
			statusStr = "[skipped]"
		}

		fmt.Printf("%s–%s  %-30s  %s\n", slot.Start, slot.End, taskName, statusStr)

		if slot.Feedback != nil && slot.Feedback.Note != "" {
			fmt.Printf("            Note: %s\n", slot.Feedback.Note)
		}
	}

	return nil
}
