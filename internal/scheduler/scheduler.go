package scheduler

import (
	"fmt"
	"sort"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

type Scheduler struct{}

func New() *Scheduler {
	return &Scheduler{}
}

// GeneratePlan creates a day plan for the given date
func (s *Scheduler) GeneratePlan(date string, tasks []models.Task, dayStart, dayEnd string) (models.DayPlan, error) {
	plan := models.DayPlan{
		Date:  date,
		Slots: []models.Slot{},
	}

	// Parse date
	planDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return plan, fmt.Errorf("invalid date format: %w", err)
	}

	// Parse day boundaries
	startTime, err := parseTime(dayStart)
	if err != nil {
		return plan, fmt.Errorf("invalid day start time: %w", err)
	}
	endTime, err := parseTime(dayEnd)
	if err != nil {
		return plan, fmt.Errorf("invalid day end time: %w", err)
	}

	// Filter active tasks
	var activeTasks []models.Task
	for _, task := range tasks {
		if task.Active {
			activeTasks = append(activeTasks, task)
		}
	}

	// Step 1: Place fixed appointments
	var fixedSlots []models.Slot
	var flexibleTasks []models.Task

	for _, task := range activeTasks {
		if task.Kind == models.TaskKindAppointment {
			// Appointments must have both fixed start and end times
			if task.FixedStart != "" && task.FixedEnd != "" {
				fixedSlots = append(fixedSlots, models.Slot{
					Start:  task.FixedStart,
					End:    task.FixedEnd,
					TaskID: task.ID,
					Status: models.SlotStatusPlanned,
				})
			} else {
				// Treat incomplete appointments as flexible tasks
				flexibleTasks = append(flexibleTasks, task)
			}
		} else if task.Kind == models.TaskKindFlexible {
			flexibleTasks = append(flexibleTasks, task)
		}
	}

	// Sort fixed slots by start time
	sort.Slice(fixedSlots, func(i, j int) bool {
		return fixedSlots[i].Start < fixedSlots[j].Start
	})

	// Step 2: Filter flexible tasks based on recurrence
	var candidateTasks []models.Task
	for _, task := range flexibleTasks {
		if shouldScheduleTask(task, planDate) {
			candidateTasks = append(candidateTasks, task)
		}
	}

	// Step 3: Sort flexible tasks by priority and lateness
	sort.Slice(candidateTasks, func(i, j int) bool {
		// Lower priority number = higher priority
		if candidateTasks[i].Priority != candidateTasks[j].Priority {
			return candidateTasks[i].Priority < candidateTasks[j].Priority
		}
		// Then by lateness
		return calculateLateness(candidateTasks[i], planDate) > calculateLateness(candidateTasks[j], planDate)
	})

	// Step 4: Find free blocks and schedule flexible tasks
	freeBlocks := findFreeBlocks(startTime, endTime, fixedSlots)
	
	scheduledSlots := make([]models.Slot, 0)
	usedTasks := make(map[string]bool)

	// Try to place each task in any available block
	for _, task := range candidateTasks {
		if usedTasks[task.ID] {
			continue
		}

		placed := false
		for blockIdx := 0; blockIdx < len(freeBlocks); blockIdx++ {
			block := freeBlocks[blockIdx]

			// Check if task fits in time constraints
			if !canScheduleInBlock(task, block) {
				continue
			}

			// Try to place task
			slot, ok := placeTaskInBlock(task, block)
			if ok {
				scheduledSlots = append(scheduledSlots, slot)
				usedTasks[task.ID] = true
				placed = true

				// Update blocks: remove current block and add up to 2 new blocks
				slotStart, _ := parseTime(slot.Start)
				slotEnd, _ := parseTime(slot.End)
				
				// Remove the current block
				freeBlocks = append(freeBlocks[:blockIdx], freeBlocks[blockIdx+1:]...)
				
				// Add block before the task if there's space
				if block.start < slotStart {
					freeBlocks = append(freeBlocks, timeBlock{start: block.start, end: slotStart})
				}
				
				// Add block after the task if there's space
				if slotEnd < block.end {
					freeBlocks = append(freeBlocks, timeBlock{start: slotEnd, end: block.end})
				}
				
				break // Move to next task
			}
		}

		if !placed {
			// Task couldn't be scheduled
			continue
		}
	}

	// Combine fixed and flexible slots, then sort
	plan.Slots = append(fixedSlots, scheduledSlots...)
	sort.Slice(plan.Slots, func(i, j int) bool {
		return plan.Slots[i].Start < plan.Slots[j].Start
	})

	return plan, nil
}

type timeBlock struct {
	start int // minutes from midnight
	end   int // minutes from midnight
}

func parseTime(timeStr string) (int, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

func formatTime(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%02d:%02d", hours, mins)
}

func shouldScheduleTask(task models.Task, date time.Time) bool {
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
		lastDone, err := time.Parse("2006-01-02", task.LastDone)
		if err != nil {
			return false
		}
		daysSince := int(date.Sub(lastDone).Hours() / 24)
		return daysSince >= task.Recurrence.IntervalDays
	case models.RecurrenceAdHoc:
		return false // Only schedule if explicitly marked (not implemented in v0.1)
	default:
		return false
	}
}

func calculateLateness(task models.Task, date time.Time) float64 {
	if task.LastDone == "" {
		return 1.0
	}

	lastDone, err := time.Parse("2006-01-02", task.LastDone)
	if err != nil {
		return 0.0
	}

	daysSince := date.Sub(lastDone).Hours() / 24
	
	interval := float64(task.Recurrence.IntervalDays)
	if interval == 0 {
		interval = 1
	}

	return daysSince / interval
}

func findFreeBlocks(dayStart, dayEnd int, fixedSlots []models.Slot) []timeBlock {
	var blocks []timeBlock

	currentStart := dayStart

	for _, slot := range fixedSlots {
		slotStart, err := parseTime(slot.Start)
		if err != nil {
			continue
		}
		slotEnd, err := parseTime(slot.End)
		if err != nil {
			continue
		}

		// If there's a gap before this slot
		if currentStart < slotStart {
			blocks = append(blocks, timeBlock{start: currentStart, end: slotStart})
		}

		currentStart = slotEnd
	}

	// Add final block if there's time remaining
	if currentStart < dayEnd {
		blocks = append(blocks, timeBlock{start: currentStart, end: dayEnd})
	}

	return blocks
}

func canScheduleInBlock(task models.Task, block timeBlock) bool {
	// Check if task fits in the block duration
	if task.DurationMin > block.end-block.start {
		return false
	}

	// Check earliest/latest constraints
	if task.EarliestStart != "" {
		earliest, err := parseTime(task.EarliestStart)
		if err == nil && block.end <= earliest {
			return false
		}
	}

	if task.LatestEnd != "" {
		latest, err := parseTime(task.LatestEnd)
		if err == nil && block.start >= latest {
			return false
		}
	}

	return true
}

func placeTaskInBlock(task models.Task, block timeBlock) (models.Slot, bool) {
	// Determine actual start time within constraints
	startTime := block.start

	if task.EarliestStart != "" {
		earliest, err := parseTime(task.EarliestStart)
		if err == nil && earliest > startTime {
			startTime = earliest
		}
	}

	// Calculate end time
	endTime := startTime + task.DurationMin

	// Check if it fits within latest end constraint
	if task.LatestEnd != "" {
		latest, err := parseTime(task.LatestEnd)
		if err == nil && endTime > latest {
			return models.Slot{}, false
		}
	}

	// Check if it fits within the block
	if endTime > block.end {
		return models.Slot{}, false
	}

	return models.Slot{
		Start:  formatTime(startTime),
		End:    formatTime(endTime),
		TaskID: task.ID,
		Status: models.SlotStatusPlanned,
	}, true
}
