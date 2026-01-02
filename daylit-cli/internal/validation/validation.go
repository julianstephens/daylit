package validation

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// ConflictType represents the type of validation conflict
type ConflictType string

const (
	ConflictOverlappingFixedTasks ConflictType = "overlapping_fixed_tasks"
	ConflictOverlappingSlots      ConflictType = "overlapping_slots"
	ConflictExceedsWakingWindow   ConflictType = "exceeds_waking_window"
	ConflictOvercommitted         ConflictType = "overcommitted"
	ConflictMissingTaskID         ConflictType = "missing_task_id"
	ConflictDuplicateTaskName     ConflictType = "duplicate_task_name"
	ConflictInvalidDateTime       ConflictType = "invalid_datetime"
)

// Conflict represents a detected conflict in tasks or plans
type Conflict struct {
	Type        ConflictType
	Description string
	Date        string   // YYYY-MM-DD format (if applicable)
	Items       []string // Task/slot names involved
	TimeRange   string   // Human-readable time range (if applicable)
	TaskIDs     []string // IDs of tasks involved (for auto-fixing)
}

// ValidationResult contains all detected conflicts
type ValidationResult struct {
	Conflicts []Conflict
}

// FixAction represents an action taken during auto-fix
type FixAction struct {
	Action         string   // Human-readable description of the action
	SourceConflict Conflict // The conflict that triggered this fix action
}

// HasConflicts returns true if there are any conflicts
func (vr *ValidationResult) HasConflicts() bool {
	return len(vr.Conflicts) > 0
}

// FormatReport returns a human-readable report of all conflicts
func (vr *ValidationResult) FormatReport() string {
	if !vr.HasConflicts() {
		return "No conflicts detected."
	}

	report := "Conflicts detected:\n"
	for _, conflict := range vr.Conflicts {
		report += fmt.Sprintf("- %s\n", conflict.Description)
	}
	return report
}

// Validator validates tasks and plans for conflicts
type Validator struct{}

// New creates a new Validator
func New() *Validator {
	return &Validator{}
}

// ValidateTasks checks tasks for conflicts
func (v *Validator) ValidateTasks(tasks []models.Task) ValidationResult {
	return v.ValidateTasksForDate(tasks, nil)
}

// ValidateTasksForDate checks tasks for conflicts, optionally scoped to a specific date.
// If planDate is nil, all tasks are validated.
// If planDate is provided, only tasks that would be scheduled on that date are validated for conflicts.
func (v *Validator) ValidateTasksForDate(tasks []models.Task, planDate *time.Time) ValidationResult {
	result := ValidationResult{Conflicts: []Conflict{}}

	// Check for duplicate task names
	nameCount := make(map[string][]string)
	for _, task := range tasks {
		if task.DeletedAt != nil {
			continue // Skip deleted tasks
		}
		// Skip empty names to avoid false positives
		if task.Name == "" {
			continue
		}
		nameCount[task.Name] = append(nameCount[task.Name], task.ID)
	}

	for name, ids := range nameCount {
		if len(ids) > 1 {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:        ConflictDuplicateTaskName,
				Description: fmt.Sprintf("Duplicate task name: \"%s\" (IDs: %v)", name, ids),
				Items:       []string{name},
				TaskIDs:     ids,
			})
		}
	}

	// Check for invalid time values in tasks
	for _, task := range tasks {
		if task.DeletedAt != nil {
			continue
		}

		if task.EarliestStart != "" {
			if !isValidTimeFormat(task.EarliestStart) {
				result.Conflicts = append(result.Conflicts, Conflict{
					Type:        ConflictInvalidDateTime,
					Description: fmt.Sprintf("Task \"%s\" has invalid earliest_start time: %s", task.Name, task.EarliestStart),
					Items:       []string{task.Name},
				})
			}
		}

		if task.LatestEnd != "" {
			if !isValidTimeFormat(task.LatestEnd) {
				result.Conflicts = append(result.Conflicts, Conflict{
					Type:        ConflictInvalidDateTime,
					Description: fmt.Sprintf("Task \"%s\" has invalid latest_end time: %s", task.Name, task.LatestEnd),
					Items:       []string{task.Name},
				})
			}
		}

		if task.FixedStart != "" {
			if !isValidTimeFormat(task.FixedStart) {
				result.Conflicts = append(result.Conflicts, Conflict{
					Type:        ConflictInvalidDateTime,
					Description: fmt.Sprintf("Task \"%s\" has invalid fixed_start time: %s", task.Name, task.FixedStart),
					Items:       []string{task.Name},
				})
			}
		}

		if task.FixedEnd != "" {
			if !isValidTimeFormat(task.FixedEnd) {
				result.Conflicts = append(result.Conflicts, Conflict{
					Type:        ConflictInvalidDateTime,
					Description: fmt.Sprintf("Task \"%s\" has invalid fixed_end time: %s", task.Name, task.FixedEnd),
					Items:       []string{task.Name},
				})
			}
		}

		// Check for negative duration in fixed appointments
		if task.FixedStart != "" && task.FixedEnd != "" {
			startMin, err1 := parseTimeToMinutes(task.FixedStart)
			endMin, err2 := parseTimeToMinutes(task.FixedEnd)
			if err1 == nil && err2 == nil && endMin < startMin {
				result.Conflicts = append(result.Conflicts, Conflict{
					Type:        ConflictInvalidDateTime,
					Description: fmt.Sprintf("Task \"%s\" has end time (%s) before start time (%s)", task.Name, task.FixedEnd, task.FixedStart),
					Items:       []string{task.Name},
				})
			}
		}
	}

	// Check for overlapping fixed appointments (across all tasks)
	// Note: Only active appointments are checked
	var fixedTasks []models.Task
	for _, task := range tasks {
		if task.DeletedAt != nil {
			continue
		}
		// Skip inactive tasks as they won't be scheduled
		if !task.Active {
			continue
		}
		if task.Kind == models.TaskKindAppointment && task.FixedStart != "" && task.FixedEnd != "" {
			// If a plan date is provided, only check tasks that would be scheduled on that date
			if planDate != nil && !taskScheduledOnDate(task, *planDate) {
				continue
			}
			fixedTasks = append(fixedTasks, task)
		}
	}

	// Sort by start time for overlap detection
	sort.Slice(fixedTasks, func(i, j int) bool {
		return fixedTasks[i].FixedStart < fixedTasks[j].FixedStart
	})

	// Check for overlaps in fixed appointments
	// Note: This checks time-of-day overlap only. Tasks with different recurrence patterns
	// (e.g., Monday-only vs Tuesday-only) may be flagged even though they never occur on the same day.
	// O(n²) complexity - acceptable for typical use cases with few appointments.
	for i := 0; i < len(fixedTasks); i++ {
		for j := i + 1; j < len(fixedTasks); j++ {
			t1 := fixedTasks[i]
			t2 := fixedTasks[j]

			// Check if times overlap (ignoring dates since these are time-of-day based)
			if timesOverlap(t1.FixedStart, t1.FixedEnd, t2.FixedStart, t2.FixedEnd) {
				// Check if recurrence patterns overlap
				if recurrenceOverlaps(t1.Recurrence, t2.Recurrence) {
					result.Conflicts = append(result.Conflicts, Conflict{
						Type: ConflictOverlappingFixedTasks,
						Description: fmt.Sprintf("Appointments overlap: \"%s\" (%s-%s) and \"%s\" (%s-%s)",
							t1.Name, t1.FixedStart, t1.FixedEnd, t2.Name, t2.FixedStart, t2.FixedEnd),
						Items:     []string{t1.Name, t2.Name},
						TimeRange: fmt.Sprintf("%s-%s", t1.FixedStart, t1.FixedEnd),
					})
				}
			}
		}
	}

	return result
}

// ValidatePlan checks a plan for conflicts
func (v *Validator) ValidatePlan(plan models.DayPlan, tasks []models.Task, dayStart, dayEnd string) ValidationResult {
	result := ValidationResult{Conflicts: []Conflict{}}

	// Build task map for quick lookup
	taskMap := make(map[string]models.Task)
	for _, task := range tasks {
		if task.DeletedAt == nil {
			taskMap[task.ID] = task
		}
	}

	// Check for invalid date
	planDate, err := time.Parse(constants.DateFormat, plan.Date)
	if err != nil {
		result.Conflicts = append(result.Conflicts, Conflict{
			Type:        ConflictInvalidDateTime,
			Description: fmt.Sprintf("Invalid plan date: %s", plan.Date),
			Date:        plan.Date,
		})
		return result // Can't continue validation without valid date
	}

	// Parse day boundaries
	dayStartMinutes, err := parseTimeToMinutes(dayStart)
	if err != nil {
		result.Conflicts = append(result.Conflicts, Conflict{
			Type:        ConflictInvalidDateTime,
			Description: fmt.Sprintf("Invalid day start time: %s", dayStart),
		})
	}

	dayEndMinutes, err := parseTimeToMinutes(dayEnd)
	if err != nil {
		result.Conflicts = append(result.Conflicts, Conflict{
			Type:        ConflictInvalidDateTime,
			Description: fmt.Sprintf("Invalid day end time: %s", dayEnd),
		})
	}

	wakingWindowMinutes := dayEndMinutes - dayStartMinutes
	if wakingWindowMinutes <= 0 {
		result.Conflicts = append(result.Conflicts, Conflict{
			Type:        ConflictInvalidDateTime,
			Description: fmt.Sprintf("Invalid waking window: day_start (%s) must be before day_end (%s)", dayStart, dayEnd),
		})
		return result // Can't continue validation
	}

	// Check each slot
	totalPlannedMinutes := 0
	for _, slot := range plan.Slots {
		if slot.DeletedAt != nil {
			continue
		}

		// Check for invalid time format in slots
		if !isValidTimeFormat(slot.Start) {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:        ConflictInvalidDateTime,
				Description: fmt.Sprintf("%s: Invalid slot start time: %s", formatDate(planDate), slot.Start),
				Date:        plan.Date,
			})
		}
		if !isValidTimeFormat(slot.End) {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:        ConflictInvalidDateTime,
				Description: fmt.Sprintf("%s: Invalid slot end time: %s", formatDate(planDate), slot.End),
				Date:        plan.Date,
			})
		}

		// Check for missing task ID
		_, exists := taskMap[slot.TaskID]
		if !exists {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:        ConflictMissingTaskID,
				Description: fmt.Sprintf("%s: Slot references missing task ID: %s", formatDate(planDate), slot.TaskID),
				Date:        plan.Date,
			})
		}

		// Calculate slot duration
		slotStart, err := parseTimeToMinutes(slot.Start)
		if err != nil {
			continue // Already reported as invalid time
		}
		slotEnd, err := parseTimeToMinutes(slot.End)
		if err != nil {
			continue // Already reported as invalid time
		}

		// Validate that the slot end time is not before the start time
		if slotEnd < slotStart {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:        ConflictInvalidDateTime,
				Description: fmt.Sprintf("%s: Slot end time '%s' is before start time '%s'", formatDate(planDate), slot.End, slot.Start),
				Date:        plan.Date,
			})
			continue
		}

		slotDuration := slotEnd - slotStart
		totalPlannedMinutes += slotDuration
	}

	// Check for overlapping slots
	// O(n²) complexity - acceptable for typical daily plans with moderate number of slots
	nonDeletedSlots := make([]models.Slot, 0)
	for _, slot := range plan.Slots {
		if slot.DeletedAt == nil {
			nonDeletedSlots = append(nonDeletedSlots, slot)
		}
	}

	sort.Slice(nonDeletedSlots, func(i, j int) bool {
		return nonDeletedSlots[i].Start < nonDeletedSlots[j].Start
	})

	for i := 0; i < len(nonDeletedSlots); i++ {
		for j := i + 1; j < len(nonDeletedSlots); j++ {
			slot1 := nonDeletedSlots[i]
			slot2 := nonDeletedSlots[j]

			if timesOverlap(slot1.Start, slot1.End, slot2.Start, slot2.End) {
				task1Name := "Unknown"
				task2Name := "Unknown"
				if t, ok := taskMap[slot1.TaskID]; ok {
					task1Name = t.Name
				}
				if t, ok := taskMap[slot2.TaskID]; ok {
					task2Name = t.Name
				}

				result.Conflicts = append(result.Conflicts, Conflict{
					Type: ConflictOverlappingSlots,
					Description: fmt.Sprintf("%s: %s-%s \"%s\" overlaps \"%s\"",
						formatDate(planDate), slot1.Start, slot1.End, task1Name, task2Name),
					Date:      plan.Date,
					Items:     []string{task1Name, task2Name},
					TimeRange: fmt.Sprintf("%s-%s", slot1.Start, slot1.End),
				})
			}
		}
	}

	// Check if plan exceeds waking window
	if totalPlannedMinutes > wakingWindowMinutes {
		hoursScheduled := float64(totalPlannedMinutes) / 60.0
		hoursAvailable := float64(wakingWindowMinutes) / 60.0
		result.Conflicts = append(result.Conflicts, Conflict{
			Type: ConflictExceedsWakingWindow,
			Description: fmt.Sprintf("%s: %.1fh scheduled exceeds %.1fh waking window",
				formatDate(planDate), hoursScheduled, hoursAvailable),
			Date: plan.Date,
		})
	}

	// Check if plan is overcommitted (more than 80% of waking window as a warning)
	overcommitThreshold := int(float64(wakingWindowMinutes) * 0.8)
	if totalPlannedMinutes > overcommitThreshold && totalPlannedMinutes <= wakingWindowMinutes {
		hoursScheduled := float64(totalPlannedMinutes) / 60.0
		hoursAvailable := float64(wakingWindowMinutes) / 60.0
		result.Conflicts = append(result.Conflicts, Conflict{
			Type: ConflictOvercommitted,
			Description: fmt.Sprintf("%s: %.1fh scheduled in %.1fh waking window (>80%% capacity)",
				formatDate(planDate), hoursScheduled, hoursAvailable),
			Date: plan.Date,
		})
	}

	return result
}

// Helper functions

func isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse(constants.TimeFormat, timeStr)
	return err == nil
}

func parseTimeToMinutes(timeStr string) (int, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

// timesOverlap checks if two time ranges overlap
// Assumes all times are in HH:MM format
func timesOverlap(start1, end1, start2, end2 string) bool {
	s1, err := parseTimeToMinutes(start1)
	if err != nil {
		return false
	}
	e1, err := parseTimeToMinutes(end1)
	if err != nil {
		return false
	}
	s2, err := parseTimeToMinutes(start2)
	if err != nil {
		return false
	}
	e2, err := parseTimeToMinutes(end2)
	if err != nil {
		return false
	}

	// Two ranges overlap if: start1 < end2 AND start2 < end1
	return s1 < e2 && s2 < e1
}

func formatDate(t time.Time) string {
	// Format as "Mon" for day of week abbreviation
	return t.Format("Mon")
}

// AutoFixDuplicateTasks fixes duplicate task conflicts by keeping a single task and soft-deleting the others
// Returns a slice of FixActions describing what was fixed
func AutoFixDuplicateTasks(conflicts []Conflict, tasks []models.Task, deleteFunc func(id string) error) []FixAction {
	actions := []FixAction{}

	// Build a map of tasks by ID for quick lookup
	taskMap := make(map[string]models.Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// Process each duplicate conflict
	for _, conflict := range conflicts {
		if conflict.Type != ConflictDuplicateTaskName {
			continue
		}

		if len(conflict.TaskIDs) <= 1 {
			continue // No duplicates to fix
		}

		// Identify a task to keep and mark others for deletion
		// Note: Since tasks don't have creation timestamps, we use ID ordering
		// as a heuristic. This keeps behavior consistent and deterministic.
		// In practice, any duplicate could be kept with similar results.
		var tasksToCheck []models.Task
		for _, id := range conflict.TaskIDs {
			if task, ok := taskMap[id]; ok && task.DeletedAt == nil {
				tasksToCheck = append(tasksToCheck, task)
			}
		}

		if len(tasksToCheck) <= 1 {
			continue // Nothing to fix
		}

		// Sort by ID for deterministic behavior
		// Note: This uses lexicographic ordering of ID strings (e.g., UUIDs).
		// While this doesn't reflect creation order, it ensures consistent behavior
		// across runs for the same set of duplicates.
		sort.Slice(tasksToCheck, func(i, j int) bool {
			return tasksToCheck[i].ID < tasksToCheck[j].ID
		})

		// Keep the first task (by ID ordering), delete the rest
		keepTask := tasksToCheck[0]
		var deletedIDs []string
		var failedIDs []string

		for i := 1; i < len(tasksToCheck); i++ {
			taskToDelete := tasksToCheck[i]
			if err := deleteFunc(taskToDelete.ID); err == nil {
				deletedIDs = append(deletedIDs, taskToDelete.ID)
			} else {
				// Track failed deletions but continue processing
				failedIDs = append(failedIDs, taskToDelete.ID)
			}
		}

		if len(deletedIDs) > 0 {
			actionMsg := fmt.Sprintf("Removed %d duplicate task(s) with name \"%s\" (kept ID: %s, removed: %v)", len(deletedIDs), keepTask.Name, keepTask.ID, deletedIDs)
			if len(failedIDs) > 0 {
				actionMsg += fmt.Sprintf(" (failed to remove: %v)", failedIDs)
			}
			actions = append(actions, FixAction{
				Action:         actionMsg,
				SourceConflict: conflict,
			})
		} else if len(failedIDs) > 0 {
			// All deletions failed
			actions = append(actions, FixAction{
				Action:         fmt.Sprintf("Failed to remove duplicates for \"%s\": %v", keepTask.Name, failedIDs),
				SourceConflict: conflict,
			})
		}
	}

	return actions
}

// recurrenceOverlaps checks if two recurrence patterns can occur on the same day
func recurrenceOverlaps(r1, r2 models.Recurrence) bool {
	// If either is daily, they overlap (unless the other is weekly with empty mask, which shouldn't happen)
	if r1.Type == models.RecurrenceDaily || r2.Type == models.RecurrenceDaily {
		return true
	}

	// If both are weekly, check for common weekdays
	if r1.Type == models.RecurrenceWeekly && r2.Type == models.RecurrenceWeekly {
		// If either mask is empty, assume overlap (conservative)
		if len(r1.WeekdayMask) == 0 || len(r2.WeekdayMask) == 0 {
			return true
		}

		for _, d1 := range r1.WeekdayMask {
			for _, d2 := range r2.WeekdayMask {
				if d1 == d2 {
					return true
				}
			}
		}
		return false
	}

	// For other combinations (e.g. NDays, AdHoc), assume overlap to be safe
	return true
}

// taskScheduledOnDate checks if a task should be scheduled on the given date based on its recurrence pattern.
// This mirrors the logic in scheduler.shouldScheduleTask to ensure consistency.
func taskScheduledOnDate(task models.Task, date time.Time) bool {
	switch task.Recurrence.Type {
	case models.RecurrenceDaily:
		return true
	case models.RecurrenceWeekly:
		if len(task.Recurrence.WeekdayMask) == 0 {
			return false
		}
		for _, wd := range task.Recurrence.WeekdayMask {
			if date.Weekday() == wd {
				return true
			}
		}
		return false
	case models.RecurrenceNDays:
		if task.LastDone == "" {
			return true
		}
		lastDone, err := time.Parse(constants.DateFormat, task.LastDone)
		if err != nil {
			return false
		}
		// Use date-based arithmetic to avoid DST issues with explicit rounding
		daysSince := int(math.Round(date.Sub(lastDone).Hours() / 24))
		return daysSince >= task.Recurrence.IntervalDays
	case models.RecurrenceAdHoc:
		return false // Ad-hoc tasks are not automatically scheduled
	default:
		return false
	}
}
