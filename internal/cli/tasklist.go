package cli

import (
	"fmt"

	"github.com/julianstephens/daylit/internal/models"
)

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
