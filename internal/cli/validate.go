package cli

import (
	"fmt"

	"github.com/julianstephens/daylit/internal/validation"
)

type ValidateCmd struct{}

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
	today := "today"
	dateStr, err := parseDateArgument(today)
	if err != nil {
		return fmt.Errorf("failed to get today's date: %w", err)
	}

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

	// Print report
	fmt.Println()
	fmt.Println(combinedResult.FormatReport())

	if combinedResult.HasConflicts() {
		return nil // Don't return error, just show conflicts
	}

	return nil
}
