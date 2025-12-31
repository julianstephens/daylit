package cli

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/internal/validation"
)

type ValidateCmd struct {
	Fix bool `help:"Automatically fix conflicts where possible (e.g., remove duplicate tasks)." default:"false"`
}

func (cmd *ValidateCmd) Run(ctx *Context) error {
	// Load storage
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load storage: %w", err)
	}
	defer ctx.Store.Close()

	// Get settings for day boundaries
	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Get all tasks
	tasks, err := ctx.Store.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Create validator
	validator := validation.New()

	// Validate tasks
	fmt.Println("Validating tasks...")
	taskResult := validator.ValidateTasks(tasks)

	// For plan validation, we'll only validate today's plan if it exists
	fmt.Println("Validating today's plan...")
	// Get today's date
	today := time.Now()
	dateStr := today.Format("2006-01-02")

	plan, err := ctx.Store.GetPlan(dateStr)
	var planResult validation.ValidationResult
	if err == nil && len(plan.Slots) > 0 {
		planResult = validator.ValidatePlan(plan, tasks, settings.DayStart, settings.DayEnd)
	} else {
		// No plan exists or error loading
		planResult = validation.ValidationResult{Conflicts: []validation.Conflict{}}
	}

	// Combine results
	allConflicts := append(taskResult.Conflicts, planResult.Conflicts...)
	combinedResult := validation.ValidationResult{Conflicts: allConflicts}

	// Apply auto-fix if requested
	if cmd.Fix {
		fmt.Println()
		fmt.Println("Auto-fixing conflicts...")

		// Auto-fix duplicate tasks
		actions := validation.AutoFixDuplicateTasks(combinedResult.Conflicts, tasks, ctx.Store.DeleteTask)

		if len(actions) > 0 {
			fmt.Println()
			fmt.Println("Actions taken:")
			for _, action := range actions {
				fmt.Printf("✓ %s\n", action.Action)
			}

			// Re-validate after fixes
			fmt.Println()
			fmt.Println("Re-validating after fixes...")
			tasks, err = ctx.Store.GetAllTasks()
			if err != nil {
				return fmt.Errorf("failed to reload tasks after fixes: %w", err)
			}

			taskResult = validator.ValidateTasks(tasks)
			plan, err = ctx.Store.GetPlan(dateStr)
			if err == nil && len(plan.Slots) > 0 {
				planResult = validator.ValidatePlan(plan, tasks, settings.DayStart, settings.DayEnd)
			} else {
				planResult = validation.ValidationResult{Conflicts: []validation.Conflict{}}
			}

			allConflicts = append(taskResult.Conflicts, planResult.Conflicts...)
			combinedResult = validation.ValidationResult{Conflicts: allConflicts}
		} else {
			fmt.Println("No fixable conflicts found.")
		}
	}

	// Print report
	fmt.Println()
	fmt.Println(combinedResult.FormatReport())

	if combinedResult.HasConflicts() {
		return nil // Don't return error, just show conflicts
	}

	return nil
}
