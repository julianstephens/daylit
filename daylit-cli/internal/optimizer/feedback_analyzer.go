package optimizer

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

// OptimizationType represents the type of optimization suggested
type OptimizationType string

const (
	OptimizationReduceDuration   OptimizationType = "reduce_duration"
	OptimizationIncreaseDuration OptimizationType = "increase_duration"
	OptimizationSplitTask        OptimizationType = "split_task"
	OptimizationRemoveTask       OptimizationType = "remove_task"
	OptimizationReduceFrequency  OptimizationType = "reduce_frequency"
)

// Optimization represents a suggested optimization for a task
type Optimization struct {
	TaskID         string           `json:"task_id"`
	TaskName       string           `json:"task_name"`
	Type           OptimizationType `json:"type"`
	Reason         string           `json:"reason"`
	CurrentValue   interface{}      `json:"current_value,omitempty"`
	SuggestedValue interface{}      `json:"suggested_value,omitempty"`
}

// FeedbackAnalyzer analyzes task feedback and suggests optimizations
type FeedbackAnalyzer struct {
	store storage.Provider
}

// NewFeedbackAnalyzer creates a new FeedbackAnalyzer
func NewFeedbackAnalyzer(store storage.Provider) *FeedbackAnalyzer {
	return &FeedbackAnalyzer{store: store}
}

// AnalyzeTask analyzes feedback for a single task and returns optimization suggestions
func (fa *FeedbackAnalyzer) AnalyzeTask(task models.Task, feedbackLimit int) ([]Optimization, error) {
	// Get feedback history for this task
	feedbackHistory, err := fa.store.GetTaskFeedbackHistory(task.ID, feedbackLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback history: %w", err)
	}

	// No feedback means no optimizations
	if len(feedbackHistory) == 0 {
		return nil, nil
	}

	var optimizations []Optimization

	// Count feedback types
	tooMuchCount := 0
	unnecessaryCount := 0
	onTrackCount := 0

	for _, entry := range feedbackHistory {
		switch entry.Rating {
		case models.FeedbackTooMuch:
			tooMuchCount++
		case models.FeedbackUnnecessary:
			unnecessaryCount++
		case models.FeedbackOnTrack:
			onTrackCount++
		}
	}

	totalFeedback := len(feedbackHistory)
	tooMuchPercent := float64(tooMuchCount) / float64(totalFeedback) * 100
	unnecessaryPercent := float64(unnecessaryCount) / float64(totalFeedback) * 100

	// Optimization logic
	// If >50% of feedback is "too_much", suggest reducing duration or splitting
	if tooMuchPercent > 50 {
		// If the task is already short (<= 30 min), suggest splitting
		if task.DurationMin <= 30 {
			optimizations = append(optimizations, Optimization{
				TaskID:   task.ID,
				TaskName: task.Name,
				Type:     OptimizationSplitTask,
				Reason:   fmt.Sprintf("%.0f%% of recent feedback indicates task is overwhelming (too_much)", tooMuchPercent),
				CurrentValue: map[string]interface{}{
					"duration_min": task.DurationMin,
				},
			})
		} else {
			// Suggest reducing duration by 25%
			newDuration := int(float64(task.DurationMin) * 0.75)
			if newDuration < 10 {
				newDuration = 10
			}
			optimizations = append(optimizations, Optimization{
				TaskID:   task.ID,
				TaskName: task.Name,
				Type:     OptimizationReduceDuration,
				Reason:   fmt.Sprintf("%.0f%% of recent feedback indicates task takes too long (too_much)", tooMuchPercent),
				CurrentValue: map[string]interface{}{
					"duration_min": task.DurationMin,
				},
				SuggestedValue: map[string]interface{}{
					"duration_min": newDuration,
				},
			})
		}
	}

	// If >= 3 instances of "unnecessary" feedback or >40% unnecessary, suggest removal or frequency reduction
	if unnecessaryCount >= 3 || unnecessaryPercent > 40 {
		// If it's a recurring task, suggest reducing frequency
		switch task.Recurrence.Type {
		case models.RecurrenceNDays:
			newInterval := task.Recurrence.IntervalDays + 2
			optimizations = append(optimizations, Optimization{
				TaskID:   task.ID,
				TaskName: task.Name,
				Type:     OptimizationReduceFrequency,
				Reason:   fmt.Sprintf("%.0f%% of recent feedback indicates task is unnecessary", unnecessaryPercent),
				CurrentValue: map[string]interface{}{
					"interval_days": task.Recurrence.IntervalDays,
				},
				SuggestedValue: map[string]interface{}{
					"interval_days": newInterval,
				},
			})
		case models.RecurrenceDaily:
			optimizations = append(optimizations, Optimization{
				TaskID:   task.ID,
				TaskName: task.Name,
				Type:     OptimizationReduceFrequency,
				Reason:   fmt.Sprintf("%.0f%% of recent feedback indicates task is unnecessary", unnecessaryPercent),
				CurrentValue: map[string]interface{}{
					"recurrence": "daily",
				},
				SuggestedValue: map[string]interface{}{
					"recurrence":    "n_days",
					"interval_days": 2,
				},
			})
		default:
			// For other types, suggest removal
			optimizations = append(optimizations, Optimization{
				TaskID:   task.ID,
				TaskName: task.Name,
				Type:     OptimizationRemoveTask,
				Reason:   fmt.Sprintf("%.0f%% of recent feedback indicates task is unnecessary", unnecessaryPercent),
			})
		}
	}

	return optimizations, nil
}

// AnalyzeAllTasks analyzes feedback for all active tasks
func (fa *FeedbackAnalyzer) AnalyzeAllTasks(feedbackLimit int) ([]Optimization, error) {
	tasks, err := fa.store.GetAllTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	var allOptimizations []Optimization
	for _, task := range tasks {
		if !task.Active {
			continue
		}

		opts, err := fa.AnalyzeTask(task, feedbackLimit)
		if err != nil {
			// Log error but continue with other tasks
			continue
		}
		allOptimizations = append(allOptimizations, opts...)
	}

	return allOptimizations, nil
}
