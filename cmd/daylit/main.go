package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/internal/models"
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
)

const (
	// Feedback adjustment constants:
	// - feedbackExistingWeight and feedbackNewWeight are exponential moving average (EMA)
	//   weights for the existing average and the new actual duration. They must sum to 1.0.
	// - feedbackTooMuchReductionFactor is an independent multiplicative scaling factor
	//   applied to reduce a task's duration when feedback indicates it is too much.
	feedbackExistingWeight         = 0.8  // EMA weight for existing average duration
	feedbackNewWeight              = 0.2  // EMA weight for new actual duration
	feedbackTooMuchReductionFactor = 0.9  // Scaling factor applied when reducing task duration
	minTaskDurationMin             = 10   // Minimum task duration in minutes
)

var CLI struct {
	Version kong.VersionFlag
	Config  string `help:"Config file path." type:"path" default:"~/.config/daylit/state.json"`

	Init     InitCmd     `cmd:"" help:"Initialize daylit storage."`
	Task     TaskCmd     `cmd:"" help:"Manage tasks."`
	Plan     PlanCmd     `cmd:"" help:"Generate day plans."`
	Now      NowCmd      `cmd:"" help:"Show current task."`
	Feedback FeedbackCmd `cmd:"" help:"Provide feedback on a slot."`
	Day      DayCmd      `cmd:"" help:"Show plan for a day."`
}

type InitCmd struct{}

func (c *InitCmd) Run(ctx *Context) error {
	if err := ctx.Store.Init(); err != nil {
		return err
	}
	fmt.Printf("Initialized daylit storage at: %s\n", ctx.Store.GetConfigPath())
	return nil
}

type TaskCmd struct {
	Add  TaskAddCmd  `cmd:"" help:"Add a new task."`
	List TaskListCmd `cmd:"" help:"List all tasks."`
}

type TaskAddCmd struct {
	Name       string `arg:"" help:"Task name."`
	Duration   int    `help:"Duration in minutes." required:""`
	Recurrence string `help:"Recurrence type (daily|weekly|n_days|ad_hoc)." default:"ad_hoc"`
	Interval   int    `help:"Interval for n_days recurrence." default:"1"`
	Weekdays   string `help:"Comma-separated weekdays for weekly recurrence."`
	Earliest   string `help:"Earliest start time (HH:MM)."`
	Latest     string `help:"Latest end time (HH:MM)."`
	FixedStart string `help:"Fixed start time for appointments (HH:MM)."`
	FixedEnd   string `help:"Fixed end time for appointments (HH:MM)."`
	Priority   int    `help:"Priority (1-5, lower is higher priority)." default:"3"`
}

func (c *TaskAddCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Determine task kind
	taskKind := models.TaskKindFlexible
	if c.FixedStart != "" && c.FixedEnd != "" {
		taskKind = models.TaskKindAppointment
	}

	// Parse recurrence
	var recType models.RecurrenceType
	switch c.Recurrence {
	case "daily":
		recType = models.RecurrenceDaily
	case "weekly":
		recType = models.RecurrenceWeekly
	case "n_days":
		recType = models.RecurrenceNDays
	case "ad_hoc":
		recType = models.RecurrenceAdHoc
	default:
		return fmt.Errorf("invalid recurrence type: %s", c.Recurrence)
	}

	rec := models.Recurrence{
		Type:         recType,
		IntervalDays: c.Interval,
	}

	// Parse weekdays for weekly recurrence
	if recType == models.RecurrenceWeekly && c.Weekdays != "" {
		wds, err := parseWeekdays(c.Weekdays)
		if err != nil {
			return err
		}
		rec.WeekdayMask = wds
	}

	// Create task
	task := models.Task{
		ID:                   uuid.New().String(),
		Name:                 c.Name,
		Kind:                 taskKind,
		DurationMin:          c.Duration,
		EarliestStart:        c.Earliest,
		LatestEnd:            c.Latest,
		FixedStart:           c.FixedStart,
		FixedEnd:             c.FixedEnd,
		Recurrence:           rec,
		Priority:             c.Priority,
		Active:               true,
		SuccessStreak:        0,
		AvgActualDurationMin: float64(c.Duration),
	}

	if err := ctx.Store.AddTask(task); err != nil {
		return err
	}

	fmt.Printf("Added task: %s (ID: %s)\n", c.Name, task.ID)
	return nil
}

type TaskListCmd struct {
	ActiveOnly bool `help:"Show only active tasks."`
}

func (c *TaskListCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	tasks := ctx.Store.GetAllTasks()
	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	fmt.Println("Tasks:")
	for _, task := range tasks {
		if c.ActiveOnly && !task.Active {
			continue
		}

		status := "active"
		if !task.Active {
			status = "inactive"
		}

		recStr := formatRecurrence(task.Recurrence)
		fmt.Printf("  [%s] %s - %dm (%s, priority %d)\n",
			status, task.Name, task.DurationMin, recStr, task.Priority)

		if task.Kind == models.TaskKindAppointment {
			fmt.Printf("      Fixed: %s - %s\n", task.FixedStart, task.FixedEnd)
		} else if task.EarliestStart != "" || task.LatestEnd != "" {
			fmt.Printf("      Window: %s - %s\n", task.EarliestStart, task.LatestEnd)
		}
	}

	return nil
}

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

	// Get settings
	settings := ctx.Store.GetSettings()

	// Get all tasks
	tasks := ctx.Store.GetAllTasks()

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

type NowCmd struct{}

func (c *NowCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	currentMinutes := now.Hour()*60 + now.Minute()

	plan, err := ctx.Store.GetPlan(dateStr)
	if err != nil {
		fmt.Println("No active plan for today.")
		return nil
	}

	// Find current slot
	var currentSlot *models.Slot
	for i := range plan.Slots {
		if plan.Slots[i].Status == models.SlotStatusAccepted || plan.Slots[i].Status == models.SlotStatusDone {
			startMinutes, err := parseTimeToMinutes(plan.Slots[i].Start)
			if err != nil {
				continue
			}
			endMinutes, err := parseTimeToMinutes(plan.Slots[i].End)
			if err != nil {
				continue
			}
			if startMinutes <= currentMinutes && currentMinutes < endMinutes {
				currentSlot = &plan.Slots[i]
				break
			}
		}
	}

	if currentSlot == nil {
		fmt.Printf("Now (%02d:%02d): Free time\n", now.Hour(), now.Minute())
		return nil
	}

	task, err := ctx.Store.GetTask(currentSlot.TaskID)
	if err != nil {
		return err
	}

	fmt.Printf("Now (%02d:%02d): You planned to be doing:\n\n", now.Hour(), now.Minute())
	fmt.Printf("%s–%s  %s\n", currentSlot.Start, currentSlot.End, task.Name)

	return nil
}

type FeedbackCmd struct {
	Rating string `help:"Rating (on_track|too_much|unnecessary)." required:""`
	Note   string `help:"Optional note."`
}

func (c *FeedbackCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Validate rating
	var rating models.FeedbackRating
	switch c.Rating {
	case "on_track":
		rating = models.FeedbackOnTrack
	case "too_much":
		rating = models.FeedbackTooMuch
	case "unnecessary":
		rating = models.FeedbackUnnecessary
	default:
		return fmt.Errorf("invalid rating: %s (use on_track, too_much, or unnecessary)", c.Rating)
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	currentMinutes := now.Hour()*60 + now.Minute()

	plan, err := ctx.Store.GetPlan(dateStr)
	if err != nil {
		return fmt.Errorf("no plan found for today")
	}

	// Find the most recent past slot without feedback
	var targetSlotIdx = -1

	for i := len(plan.Slots) - 1; i >= 0; i-- {
		slot := &plan.Slots[i]
		if (slot.Status == models.SlotStatusAccepted || slot.Status == models.SlotStatusDone) &&
			slot.Feedback == nil {
			endMinutes, err := parseTimeToMinutes(slot.End)
			if err != nil {
				// Skip slots with invalid end time format
				continue
			}
			if endMinutes <= currentMinutes {
				targetSlotIdx = i
				break
			}
		}
	}

	if targetSlotIdx == -1 {
		return fmt.Errorf("no past slot found without feedback")
	}

	// Add feedback
	plan.Slots[targetSlotIdx].Feedback = &models.Feedback{
		Rating: rating,
		Note:   c.Note,
	}
	plan.Slots[targetSlotIdx].Status = models.SlotStatusDone

	// Update task statistics
	task, err := ctx.Store.GetTask(plan.Slots[targetSlotIdx].TaskID)
	if err == nil {
		switch rating {
		case models.FeedbackOnTrack:
			// Keep duration as is, nudge slightly toward actual
			slotDuration := calculateSlotDuration(plan.Slots[targetSlotIdx])
			if slotDuration > 0 {
				task.AvgActualDurationMin = task.AvgActualDurationMin*feedbackExistingWeight + float64(slotDuration)*feedbackNewWeight
			}
			task.LastDone = dateStr
		case models.FeedbackTooMuch:
			// Reduce duration slightly
			task.DurationMin = int(float64(task.DurationMin) * feedbackTooMuchReductionFactor)
			if task.DurationMin < minTaskDurationMin {
				task.DurationMin = minTaskDurationMin
			}
			task.LastDone = dateStr
		case models.FeedbackUnnecessary:
			// Increase interval or reduce priority
			if task.Recurrence.Type == models.RecurrenceNDays {
				task.Recurrence.IntervalDays++
			}
		}
		if err := ctx.Store.UpdateTask(task); err != nil {
			return fmt.Errorf("update task with feedback: %w", err)
		}
	}

	if err := ctx.Store.SavePlan(plan); err != nil {
		return err
	}

	taskName := "Unknown task"
	if err == nil {
		taskName = task.Name
	}

	fmt.Printf("Feedback recorded for: %s–%s  %s\n",
		plan.Slots[targetSlotIdx].Start, plan.Slots[targetSlotIdx].End, taskName)

	return nil
}

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

type Context struct {
	Store     *storage.Storage
	Scheduler *scheduler.Scheduler
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("daylit"),
		kong.Description("Daily structure scheduler / time-blocking companion"),
		kong.UsageOnError(),
		kong.Vars{"version": "v0.1.0"},
	)

	// Expand home directory in config path
	configPath := CLI.Config
	if strings.HasPrefix(configPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(home, configPath[2:])
		}
	}

	// Initialize context
	store, err := storage.New(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	appCtx := &Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	// Run the command
	err = ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseWeekdays(s string) ([]time.Weekday, error) {
	parts := strings.Split(s, ",")
	var weekdays []time.Weekday

	dayMap := map[string]time.Weekday{
		"sun":       time.Sunday,
		"sunday":    time.Sunday,
		"mon":       time.Monday,
		"monday":    time.Monday,
		"tue":       time.Tuesday,
		"tuesday":   time.Tuesday,
		"wed":       time.Wednesday,
		"wednesday": time.Wednesday,
		"thu":       time.Thursday,
		"thursday":  time.Thursday,
		"fri":       time.Friday,
		"friday":    time.Friday,
		"sat":       time.Saturday,
		"saturday":  time.Saturday,
	}

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if wd, ok := dayMap[part]; ok {
			weekdays = append(weekdays, wd)
		} else {
			// Try parsing as number (0=Sunday, 6=Saturday)
			num, err := strconv.Atoi(part)
			if err == nil && num >= 0 && num <= 6 {
				weekdays = append(weekdays, time.Weekday(num))
			} else {
				return nil, fmt.Errorf("invalid weekday: %s", part)
			}
		}
	}

	return weekdays, nil
}

func formatRecurrence(rec models.Recurrence) string {
	switch rec.Type {
	case models.RecurrenceDaily:
		return "daily"
	case models.RecurrenceWeekly:
		if len(rec.WeekdayMask) > 0 {
			var days []string
			for _, wd := range rec.WeekdayMask {
				days = append(days, wd.String()[:3])
			}
			return fmt.Sprintf("weekly on %s", strings.Join(days, ","))
		}
		return "weekly"
	case models.RecurrenceNDays:
		return fmt.Sprintf("every %d days", rec.IntervalDays)
	case models.RecurrenceAdHoc:
		return "ad-hoc"
	default:
		return "unknown"
	}
}

func parseTimeToMinutes(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %q", timeStr)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour in %q: %w", timeStr, err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute in %q: %w", timeStr, err)
	}
	return hour*60 + minute, nil
}

func calculateSlotDuration(slot models.Slot) int {
	start, err := time.Parse("15:04", slot.Start)
	if err != nil {
		return 0
	}
	end, err := time.Parse("15:04", slot.End)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}
