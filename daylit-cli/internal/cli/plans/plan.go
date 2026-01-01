package plans

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/validation"
)

type PlanCmd struct {
	Date        string `arg:"" help:"Date to plan (YYYY-MM-DD or 'today')." default:"today"`
	NewRevision bool   `help:"Create a new revision instead of being blocked when an accepted plan exists." name:"new-revision"`
}

func (c *PlanCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Perform automatic backup on plan invocation (after successful load)
	ctx.PerformAutomaticBackup()

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
		if existingPlan.AcceptedAt != nil {
			// Plan is accepted - must create new revision
			if !c.NewRevision {
				fmt.Printf("An accepted plan already exists for %s (revision %d).\n", dateStr, existingPlan.Revision)
				fmt.Printf("To create a new revision, use: daylit plan %s --new-revision\n", dateStr)
				return nil
			}
			fmt.Printf("Creating new revision of plan for %s (will be revision %d)\n\n", dateStr, existingPlan.Revision+1)
		} else {
			// Plan exists but not accepted - can regenerate
			fmt.Printf("Warning: A plan already exists for %s (revision %d, not accepted). Generating a new plan will replace it.\n", dateStr, existingPlan.Revision)
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

	// Set revision to 0 so SavePlan will auto-assign it and perform immutability checks
	plan.Revision = 0

	// Validate both tasks and the generated plan
	validator := validation.New()
	taskValidationResult := validator.ValidateTasks(tasks)
	planValidationResult := validator.ValidatePlan(plan, tasks, settings.DayStart, settings.DayEnd)

	// Combine validation results
	allConflicts := append(taskValidationResult.Conflicts, planValidationResult.Conflicts...)
	validationResult := validation.ValidationResult{Conflicts: allConflicts}

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

		// Show validation warnings if any
		if validationResult.HasConflicts() {
			fmt.Println("\n⚠️  Validation warnings:")
			for _, conflict := range validationResult.Conflicts {
				fmt.Printf("  - %s\n", conflict.Description)
			}
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
		// Update all slots to accepted and set accepted_at timestamp
		for i := range plan.Slots {
			plan.Slots[i].Status = models.SlotStatusAccepted
		}
		now := time.Now().UTC().Format(time.RFC3339)
		plan.AcceptedAt = &now

		if err := ctx.Store.SavePlan(plan); err != nil {
			return err
		}

		// Get the saved plan to display the correct revision number
		savedPlan, err := ctx.Store.GetPlan(dateStr)
		if err != nil {
			// Fallback to displaying without revision number
			fmt.Println("Plan accepted and saved!")
		} else {
			fmt.Printf("Plan accepted and saved as revision %d!\n", savedPlan.Revision)
		}
	} else {
		fmt.Println("Plan discarded. You can modify tasks and regenerate.")
	}

	return nil
}
