package cli

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

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
				if task.AvgActualDurationMin <= 0 {
					// Initialize average if it was unset or invalid
					task.AvgActualDurationMin = float64(slotDuration)
				} else {
					task.AvgActualDurationMin = task.AvgActualDurationMin*constants.FeedbackExistingWeight + float64(slotDuration)*constants.FeedbackNewWeight
				}
			}
			task.LastDone = dateStr
		case models.FeedbackTooMuch:
			// Reduce duration slightly
			task.DurationMin = int(float64(task.DurationMin) * constants.FeedbackTooMuchReductionFactor)
			if task.DurationMin < constants.MinTaskDurationMin {
				task.DurationMin = constants.MinTaskDurationMin
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
