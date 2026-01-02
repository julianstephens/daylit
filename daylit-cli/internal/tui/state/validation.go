package state

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/validation"
)

// UpdateValidationStatus runs validation and updates the warning message
func (m *Model) UpdateValidationStatus() {
	// Get all tasks
	tasks, err := m.Store.GetAllTasks()
	if err != nil {
		// Store errors prevent validation - show generic message
		m.ValidationWarning = "⚠ Validation unavailable"
		m.ValidationConflicts = nil
		return
	}

	// Get settings
	settings, err := m.Store.GetSettings()
	if err != nil {
		// Store errors prevent validation - show generic message
		m.ValidationWarning = "⚠ Validation unavailable"
		m.ValidationConflicts = nil
		return
	}

	// Get today's plan
	today := time.Now().Format(constants.DateFormat)
	todayDate := time.Now()
	plan, err := m.Store.GetPlan(today)

	validator := validation.New()

	// Validate tasks first - scoped to today's date
	taskResult := validator.ValidateTasksForDate(tasks, &todayDate)

	// Validate plan if it exists
	var planResult validation.ValidationResult
	if err == nil && len(plan.Slots) > 0 {
		planResult = validator.ValidatePlan(plan, tasks, settings.DayStart, settings.DayEnd)
	}

	// Combine conflicts
	allConflicts := append(taskResult.Conflicts, planResult.Conflicts...)
	m.ValidationConflicts = allConflicts

	if len(allConflicts) > 0 {
		// Show count of conflicts
		m.ValidationWarning = fmt.Sprintf("⚠ %d validation warning(s)", len(allConflicts))
	} else {
		m.ValidationWarning = ""
	}
}
