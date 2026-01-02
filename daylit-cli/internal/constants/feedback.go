package constants

// OptimizationType represents the type of optimization suggested
type OptimizationType string

const (
	// Feedback adjustment constants:
	// - FeedbackExistingWeight and FeedbackNewWeight are exponential moving average (EMA)
	//   weights for the existing average and the new actual duration. They must sum to 1.0.
	// - FeedbackTooMuchReductionFactor is an independent multiplicative scaling factor
	//   applied to reduce a task's duration when feedback indicates it is too much.
	FeedbackExistingWeight         = 0.8 // EMA weight for existing average duration
	FeedbackNewWeight              = 0.2 // EMA weight for new actual duration
	FeedbackTooMuchReductionFactor = 0.9 // Scaling factor applied when reducing task duration
	MinTaskDurationMin             = 10  // Minimum task duration in minutes

	// Feedback Rating constants
	FeedbackOnTrack     = "on_track"
	FeedbackTooMuch     = "too_much"
	FeedbackUnnecessary = "unnecessary"

	// Optimization Types
	OptimizationReduceDuration   OptimizationType = "reduce_duration"
	OptimizationIncreaseDuration OptimizationType = "increase_duration"
	OptimizationSplitTask        OptimizationType = "split_task"
	OptimizationRemoveTask       OptimizationType = "remove_task"
	OptimizationReduceFrequency  OptimizationType = "reduce_frequency"
)

func init() {
	// Runtime validation: ensure EMA weights sum to 1.0
	if FeedbackExistingWeight+FeedbackNewWeight != 1.0 {
		panic("FeedbackExistingWeight and FeedbackNewWeight must sum to 1.0")
	}
}
