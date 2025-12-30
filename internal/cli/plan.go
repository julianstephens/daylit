package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

type PlanCmd struct {
	Date string `arg:"" help:"Date to plan (YYYY-MM-DD or 'today')." default:"today"`
}

func (c *PlanCmd) Run(ctx *Context) error {
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

	// Check if a plan already exists for this date
	existingPlan, err := ctx.Store.GetPlan(dateStr)
	if err == nil && len(existingPlan.Slots) > 0 {
		fmt.Printf("Warning: A plan already exists for %s. Generating a new plan will replace it.\n", dateStr)
		fmt.Print("Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		response = strings.TrimSpace(response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Plan generation cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Get settings
	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Get all tasks
	tasks, err := ctx.Store.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Generate plan
	plan, err := ctx.Scheduler.GeneratePlan(dateStr, tasks, settings.DayStart, settings.DayEnd)
	if err != nil {
		return err
	}

	// Display plan
	fmt.Printf("Proposed plan for %s:\n\n", dateStr)

	if len(plan.Slots) == 0 {
		fmt.Println("  No tasks scheduled for this day")
		fmt.Println("\nAccept this plan? [y/N]: ")
	} else {
		for _, slot := range plan.Slots {
			task, err := ctx.Store.GetTask(slot.TaskID)
			if err != nil {
				fmt.Printf("%s–%s  (unknown task)\n", slot.Start, slot.End)
				continue
			}
			fmt.Printf("%s–%s  %s\n", slot.Start, slot.End, task.Name)
		}
		fmt.Println("\nAccept this plan? [y/N]: ")
	}

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	response = strings.TrimSpace(response)

	if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		// Update all slots to accepted
		for i := range plan.Slots {
			plan.Slots[i].Status = models.SlotStatusAccepted
		}

		if err := ctx.Store.SavePlan(plan); err != nil {
			return err
		}

		fmt.Println("Plan accepted and saved!")
	} else {
		fmt.Println("Plan discarded. You can modify tasks and regenerate.")
	}

	return nil
}
