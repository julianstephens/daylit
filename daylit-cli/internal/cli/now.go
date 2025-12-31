package cli

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

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
