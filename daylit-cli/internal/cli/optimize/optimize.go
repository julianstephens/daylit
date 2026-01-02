package optimize

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/optimizer"
)

type OptimizeCmd struct {
	DryRun         bool `help:"Show optimization suggestions without applying them (report mode)." default:"false"`
	FeedbackLimit  int  `help:"Number of recent feedback entries to analyze per task." default:"10"`
	Interactive    bool `help:"Interactively review and apply optimizations." default:"false"`
	AutoApply      bool `help:"Automatically apply all optimizations without confirmation." default:"false"`
}

func (c *OptimizeCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Create feedback analyzer
	analyzer := optimizer.NewFeedbackAnalyzer(ctx.Store)

	// Analyze all tasks
	fmt.Println("Analyzing task feedback history...")
	optimizations, err := analyzer.AnalyzeAllTasks(c.FeedbackLimit)
	if err != nil {
		return fmt.Errorf("failed to analyze tasks: %w", err)
	}

	if len(optimizations) == 0 {
		fmt.Println("✅ No optimizations needed. All tasks are performing well based on feedback!")
		return nil
	}

	// Display optimizations
	fmt.Printf("\n📊 Found %d optimization suggestion(s):\n\n", len(optimizations))
	for i, opt := range optimizations {
		displayOptimization(i+1, opt)
	}

	// Dry run mode - just show suggestions
	if c.DryRun {
		fmt.Println("\n💡 This was a dry run. Use --interactive to apply optimizations.")
		return nil
	}

	// Auto-apply mode
	if c.AutoApply {
		fmt.Println("\n🚀 Applying all optimizations...")
		applied := 0
		for _, opt := range optimizations {
			if err := applyOptimization(ctx, opt); err != nil {
				fmt.Printf("  ❌ Failed to apply optimization for %s: %v\n", opt.TaskName, err)
			} else {
				applied++
				fmt.Printf("  ✅ Applied optimization for %s\n", opt.TaskName)
			}
		}
		fmt.Printf("\n✨ Successfully applied %d/%d optimizations.\n", applied, len(optimizations))
		return nil
	}

	// Interactive mode
	if c.Interactive {
		return c.runInteractive(ctx, optimizations)
	}

	// Default: suggest using interactive or auto-apply
	fmt.Println("\n💡 To apply these optimizations:")
	fmt.Println("  - Use --interactive to review and select which to apply")
	fmt.Println("  - Use --auto-apply to apply all automatically")
	fmt.Println("  - Use --dry-run to just see the suggestions (current mode)")
	
	return nil
}

func (c *OptimizeCmd) runInteractive(ctx *cli.Context, optimizations []optimizer.Optimization) error {
	fmt.Println("\n🎯 Interactive optimization mode")
	fmt.Println("Review each suggestion and choose whether to apply it.\n")

	applied := 0
	skipped := 0

	for i, opt := range optimizations {
		fmt.Printf("\n[%d/%d] ", i+1, len(optimizations))
		displayOptimization(0, opt)

		var choice string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Apply this optimization?").
					Options(
						huh.NewOption("Apply", "apply"),
						huh.NewOption("Skip", "skip"),
						huh.NewOption("Skip remaining", "skip_all"),
					).
					Value(&choice),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("interactive form error: %w", err)
		}

		switch choice {
		case "apply":
			if err := applyOptimization(ctx, opt); err != nil {
				fmt.Printf("  ❌ Failed to apply: %v\n", err)
			} else {
				fmt.Printf("  ✅ Applied successfully\n")
				applied++
			}
		case "skip":
			fmt.Println("  ⏭️  Skipped")
			skipped++
		case "skip_all":
			fmt.Println("  ⏭️  Skipping all remaining optimizations")
			skipped += len(optimizations) - i
			goto done
		}
	}

done:
	fmt.Printf("\n✨ Completed: %d applied, %d skipped\n", applied, skipped)
	return nil
}

func displayOptimization(num int, opt optimizer.Optimization) {
	prefix := ""
	if num > 0 {
		prefix = fmt.Sprintf("%d. ", num)
	}

	var typeIcon string
	switch opt.Type {
	case optimizer.OptimizationReduceDuration:
		typeIcon = "⏱️  Reduce Duration"
	case optimizer.OptimizationIncreaseDuration:
		typeIcon = "⏱️  Increase Duration"
	case optimizer.OptimizationSplitTask:
		typeIcon = "✂️  Split Task"
	case optimizer.OptimizationRemoveTask:
		typeIcon = "🗑️  Remove Task"
	case optimizer.OptimizationReduceFrequency:
		typeIcon = "📉 Reduce Frequency"
	default:
		typeIcon = "🔧 Optimize"
	}

	fmt.Printf("%s%s\n", prefix, typeIcon)
	fmt.Printf("   Task: %s\n", opt.TaskName)
	fmt.Printf("   Reason: %s\n", opt.Reason)

	if opt.CurrentValue != nil {
		fmt.Printf("   Current: %v\n", formatValue(opt.CurrentValue))
	}
	if opt.SuggestedValue != nil {
		fmt.Printf("   Suggested: %v\n", formatValue(opt.SuggestedValue))
	}
	fmt.Println()
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		var parts []string
		for key, val := range v {
			parts = append(parts, fmt.Sprintf("%s=%v", key, val))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", value)
	}
}

func applyOptimization(ctx *cli.Context, opt optimizer.Optimization) error {
	// Get the task
	task, err := ctx.Store.GetTask(opt.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Apply the optimization based on type
	switch opt.Type {
	case optimizer.OptimizationReduceDuration:
		if suggestedMap, ok := opt.SuggestedValue.(map[string]interface{}); ok {
			if newDuration, ok := suggestedMap["duration_min"].(int); ok {
				task.DurationMin = newDuration
			}
		}

	case optimizer.OptimizationIncreaseDuration:
		if suggestedMap, ok := opt.SuggestedValue.(map[string]interface{}); ok {
			if newDuration, ok := suggestedMap["duration_min"].(int); ok {
				task.DurationMin = newDuration
			}
		}

	case optimizer.OptimizationReduceFrequency:
		if suggestedMap, ok := opt.SuggestedValue.(map[string]interface{}); ok {
			// Handle change from daily to n_days
			if recurrence, ok := suggestedMap["recurrence"].(string); ok && recurrence == "n_days" {
				task.Recurrence.Type = models.RecurrenceNDays
				if intervalDays, ok := suggestedMap["interval_days"].(int); ok {
					task.Recurrence.IntervalDays = intervalDays
				}
			}
			// Handle increase in interval_days
			if intervalDays, ok := suggestedMap["interval_days"].(int); ok {
				task.Recurrence.IntervalDays = intervalDays
			}
		}

	case optimizer.OptimizationRemoveTask:
		// Mark task as inactive instead of deleting
		task.Active = false

	case optimizer.OptimizationSplitTask:
		// For split task, we just print a message since it requires manual intervention
		fmt.Println("   ℹ️  Task splitting requires manual action:")
		fmt.Printf("      1. Create new smaller tasks to replace '%s'\n", task.Name)
		fmt.Printf("      2. Deactivate or delete the original task\n")
		return nil
	}

	// Update the task
	if err := ctx.Store.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func init() {
	// Ensure huh uses stdout to avoid issues in some terminal environments
	huh.NewForm().WithTheme(huh.ThemeBase())
}
