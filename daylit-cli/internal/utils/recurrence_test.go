package utils

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func TestShouldScheduleTask_MonthlyDate(t *testing.T) {
	task := models.Task{
		ID:   "monthly-15",
		Name: "Monthly Task on 15th",
		Recurrence: models.Recurrence{
			Type:     constants.RecurrenceMonthlyDate,
			MonthDay: 15,
		},
	}

	// Test on the 15th - should be scheduled
	date15, _ := time.Parse(constants.DateFormat, "2026-01-15")
	if !ShouldScheduleTask(task, date15) {
		t.Error("Expected task to be scheduled on the 15th")
	}

	// Test on other days - should not be scheduled
	date14, _ := time.Parse(constants.DateFormat, "2026-01-14")
	if ShouldScheduleTask(task, date14) {
		t.Error("Expected task not to be scheduled on the 14th")
	}

	date16, _ := time.Parse(constants.DateFormat, "2026-01-16")
	if ShouldScheduleTask(task, date16) {
		t.Error("Expected task not to be scheduled on the 16th")
	}

	// Test on 15th of different month
	date15Feb, _ := time.Parse(constants.DateFormat, "2026-02-15")
	if !ShouldScheduleTask(task, date15Feb) {
		t.Error("Expected task to be scheduled on February 15th")
	}
}

func TestShouldScheduleTask_MonthlyDay_LastFriday(t *testing.T) {
	task := models.Task{
		ID:   "last-friday",
		Name: "Last Friday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   -1, // Last occurrence
			DayOfWeekInMonth: time.Friday,
		},
	}

	// January 2026: Friday the 30th is the last Friday
	lastFridayJan, _ := time.Parse(constants.DateFormat, "2026-01-30")
	if !ShouldScheduleTask(task, lastFridayJan) {
		t.Error("Expected task to be scheduled on last Friday of January (30th)")
	}

	// January 23rd is a Friday but not the last Friday
	notLastFriday, _ := time.Parse(constants.DateFormat, "2026-01-23")
	if ShouldScheduleTask(task, notLastFriday) {
		t.Error("Expected task not to be scheduled on non-last Friday")
	}

	// February 2026: Friday the 27th is the last Friday
	lastFridayFeb, _ := time.Parse(constants.DateFormat, "2026-02-27")
	if !ShouldScheduleTask(task, lastFridayFeb) {
		t.Error("Expected task to be scheduled on last Friday of February (27th)")
	}

	// Not a Friday
	notFriday, _ := time.Parse(constants.DateFormat, "2026-01-28")
	if ShouldScheduleTask(task, notFriday) {
		t.Error("Expected task not to be scheduled on non-Friday")
	}
}

func TestShouldScheduleTask_MonthlyDay_FirstMonday(t *testing.T) {
	task := models.Task{
		ID:   "first-monday",
		Name: "First Monday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   1, // First occurrence
			DayOfWeekInMonth: time.Monday,
		},
	}

	// January 2026: Monday the 5th is the first Monday
	firstMondayJan, _ := time.Parse(constants.DateFormat, "2026-01-05")
	if !ShouldScheduleTask(task, firstMondayJan) {
		t.Error("Expected task to be scheduled on first Monday of January (5th)")
	}

	// Monday the 12th is the second Monday
	secondMonday, _ := time.Parse(constants.DateFormat, "2026-01-12")
	if ShouldScheduleTask(task, secondMonday) {
		t.Error("Expected task not to be scheduled on second Monday")
	}

	// February 2026: Monday the 2nd is the first Monday
	firstMondayFeb, _ := time.Parse(constants.DateFormat, "2026-02-02")
	if !ShouldScheduleTask(task, firstMondayFeb) {
		t.Error("Expected task to be scheduled on first Monday of February (2nd)")
	}
}

func TestShouldScheduleTask_MonthlyDay_ThirdWednesday(t *testing.T) {
	task := models.Task{
		ID:   "third-wednesday",
		Name: "Third Wednesday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   3, // Third occurrence
			DayOfWeekInMonth: time.Wednesday,
		},
	}

	// January 2026: Wednesday the 21st is the third Wednesday
	thirdWedJan, _ := time.Parse(constants.DateFormat, "2026-01-21")
	if !ShouldScheduleTask(task, thirdWedJan) {
		t.Error("Expected task to be scheduled on third Wednesday of January (21st)")
	}

	// Wednesday the 14th is the second Wednesday
	secondWed, _ := time.Parse(constants.DateFormat, "2026-01-14")
	if ShouldScheduleTask(task, secondWed) {
		t.Error("Expected task not to be scheduled on second Wednesday")
	}

	// Wednesday the 7th is the first Wednesday
	firstWed, _ := time.Parse(constants.DateFormat, "2026-01-07")
	if ShouldScheduleTask(task, firstWed) {
		t.Error("Expected task not to be scheduled on first Wednesday")
	}
}

func TestShouldScheduleTask_Yearly(t *testing.T) {
	task := models.Task{
		ID:   "yearly-jan-1",
		Name: "New Year's Day",
		Recurrence: models.Recurrence{
			Type:     constants.RecurrenceYearly,
			Month:    1, // January
			MonthDay: 1, // 1st
		},
	}

	// Test on January 1st - should be scheduled
	jan1_2026, _ := time.Parse(constants.DateFormat, "2026-01-01")
	if !ShouldScheduleTask(task, jan1_2026) {
		t.Error("Expected task to be scheduled on January 1st, 2026")
	}

	jan1_2027, _ := time.Parse(constants.DateFormat, "2027-01-01")
	if !ShouldScheduleTask(task, jan1_2027) {
		t.Error("Expected task to be scheduled on January 1st, 2027")
	}

	// Test on different dates - should not be scheduled
	jan2, _ := time.Parse(constants.DateFormat, "2026-01-02")
	if ShouldScheduleTask(task, jan2) {
		t.Error("Expected task not to be scheduled on January 2nd")
	}

	dec1, _ := time.Parse(constants.DateFormat, "2026-12-01")
	if ShouldScheduleTask(task, dec1) {
		t.Error("Expected task not to be scheduled on December 1st")
	}

	// Test July 4th yearly task
	taskJuly4 := models.Task{
		ID:   "yearly-jul-4",
		Name: "Independence Day",
		Recurrence: models.Recurrence{
			Type:     constants.RecurrenceYearly,
			Month:    7, // July
			MonthDay: 4,
		},
	}

	july4, _ := time.Parse(constants.DateFormat, "2026-07-04")
	if !ShouldScheduleTask(taskJuly4, july4) {
		t.Error("Expected task to be scheduled on July 4th")
	}

	july5, _ := time.Parse(constants.DateFormat, "2026-07-05")
	if ShouldScheduleTask(taskJuly4, july5) {
		t.Error("Expected task not to be scheduled on July 5th")
	}
}

func TestShouldScheduleTask_Weekdays(t *testing.T) {
	task := models.Task{
		ID:   "weekdays",
		Name: "Weekday Task",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceWeekdays,
		},
	}

	// Test Monday - Friday (should be scheduled)
	monday, _ := time.Parse(constants.DateFormat, "2026-01-05") // Monday
	if !ShouldScheduleTask(task, monday) {
		t.Error("Expected task to be scheduled on Monday")
	}

	tuesday, _ := time.Parse(constants.DateFormat, "2026-01-06")
	if !ShouldScheduleTask(task, tuesday) {
		t.Error("Expected task to be scheduled on Tuesday")
	}

	wednesday, _ := time.Parse(constants.DateFormat, "2026-01-07")
	if !ShouldScheduleTask(task, wednesday) {
		t.Error("Expected task to be scheduled on Wednesday")
	}

	thursday, _ := time.Parse(constants.DateFormat, "2026-01-08")
	if !ShouldScheduleTask(task, thursday) {
		t.Error("Expected task to be scheduled on Thursday")
	}

	friday, _ := time.Parse(constants.DateFormat, "2026-01-09")
	if !ShouldScheduleTask(task, friday) {
		t.Error("Expected task to be scheduled on Friday")
	}

	// Test Saturday and Sunday (should not be scheduled)
	saturday, _ := time.Parse(constants.DateFormat, "2026-01-10")
	if ShouldScheduleTask(task, saturday) {
		t.Error("Expected task not to be scheduled on Saturday")
	}

	sunday, _ := time.Parse(constants.DateFormat, "2026-01-11")
	if ShouldScheduleTask(task, sunday) {
		t.Error("Expected task not to be scheduled on Sunday")
	}
}

func TestIsNthWeekdayOfMonth(t *testing.T) {
	// Test first Monday of January 2026 (5th)
	firstMon, _ := time.Parse(constants.DateFormat, "2026-01-05")
	if !isNthWeekdayOfMonth(firstMon, time.Monday, 1) {
		t.Error("Expected Jan 5 to be the first Monday")
	}
	if isNthWeekdayOfMonth(firstMon, time.Monday, 2) {
		t.Error("Expected Jan 5 not to be the second Monday")
	}

	// Test second Monday of January 2026 (12th)
	secondMon, _ := time.Parse(constants.DateFormat, "2026-01-12")
	if !isNthWeekdayOfMonth(secondMon, time.Monday, 2) {
		t.Error("Expected Jan 12 to be the second Monday")
	}

	// Test last Friday of January 2026 (30th)
	lastFri, _ := time.Parse(constants.DateFormat, "2026-01-30")
	if !isNthWeekdayOfMonth(lastFri, time.Friday, -1) {
		t.Error("Expected Jan 30 to be the last Friday")
	}

	// Test that Jan 23 is not the last Friday
	notLastFri, _ := time.Parse(constants.DateFormat, "2026-01-23")
	if isNthWeekdayOfMonth(notLastFri, time.Friday, -1) {
		t.Error("Expected Jan 23 not to be the last Friday")
	}

	// Test wrong weekday
	if isNthWeekdayOfMonth(firstMon, time.Tuesday, 1) {
		t.Error("Expected Jan 5 not to be a Tuesday")
	}
}

func TestShouldScheduleTask_BackwardCompatibility(t *testing.T) {
	// Ensure existing recurrence types still work
	dailyTask := models.Task{
		Recurrence: models.Recurrence{Type: constants.RecurrenceDaily},
	}
	date, _ := time.Parse(constants.DateFormat, "2026-01-15")
	if !ShouldScheduleTask(dailyTask, date) {
		t.Error("Daily task should be scheduled every day")
	}

	weeklyTask := models.Task{
		Recurrence: models.Recurrence{
			Type:        constants.RecurrenceWeekly,
			WeekdayMask: []time.Weekday{time.Monday, time.Wednesday},
		},
	}
	monday, _ := time.Parse(constants.DateFormat, "2026-01-05")
	if !ShouldScheduleTask(weeklyTask, monday) {
		t.Error("Weekly task should be scheduled on Monday")
	}
	tuesday, _ := time.Parse(constants.DateFormat, "2026-01-06")
	if ShouldScheduleTask(weeklyTask, tuesday) {
		t.Error("Weekly task should not be scheduled on Tuesday")
	}

	nDaysTask := models.Task{
		LastDone: "2026-01-10",
		Recurrence: models.Recurrence{
			Type:         constants.RecurrenceNDays,
			IntervalDays: 5,
		},
	}
	date15, _ := time.Parse(constants.DateFormat, "2026-01-15")
	if !ShouldScheduleTask(nDaysTask, date15) {
		t.Error("N-days task should be scheduled 5 days after last done")
	}

	adHocTask := models.Task{
		Recurrence: models.Recurrence{Type: constants.RecurrenceAdHoc},
	}
	if ShouldScheduleTask(adHocTask, date) {
		t.Error("Ad-hoc task should never be automatically scheduled")
	}
}

func TestShouldScheduleTask_MonthlyDate_Day31EdgeCases(t *testing.T) {
	task := models.Task{
		ID:   "monthly-31",
		Name: "Monthly Task on 31st",
		Recurrence: models.Recurrence{
			Type:     constants.RecurrenceMonthlyDate,
			MonthDay: 31,
		},
	}

	// Test on January 31st - should be scheduled (31 days)
	jan31, _ := time.Parse(constants.DateFormat, "2026-01-31")
	if !ShouldScheduleTask(task, jan31) {
		t.Error("Expected task to be scheduled on January 31st")
	}

	// Test on February 28th - should NOT be scheduled (no Feb 31)
	feb28, _ := time.Parse(constants.DateFormat, "2026-02-28")
	if ShouldScheduleTask(task, feb28) {
		t.Error("Expected task not to be scheduled on February 28th (no Feb 31)")
	}

	// Test on March 31st - should be scheduled (31 days)
	mar31, _ := time.Parse(constants.DateFormat, "2026-03-31")
	if !ShouldScheduleTask(task, mar31) {
		t.Error("Expected task to be scheduled on March 31st")
	}

	// Test on April 30th - should NOT be scheduled (only 30 days in April)
	apr30, _ := time.Parse(constants.DateFormat, "2026-04-30")
	if ShouldScheduleTask(task, apr30) {
		t.Error("Expected task not to be scheduled on April 30th (no Apr 31)")
	}

	// Test on May 31st - should be scheduled (31 days)
	may31, _ := time.Parse(constants.DateFormat, "2026-05-31")
	if !ShouldScheduleTask(task, may31) {
		t.Error("Expected task to be scheduled on May 31st")
	}
}

func TestShouldScheduleTask_MonthlyDay_Sunday(t *testing.T) {
	task := models.Task{
		ID:   "first-sunday",
		Name: "First Sunday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   1,
			DayOfWeekInMonth: time.Sunday, // Sunday = 0
		},
	}

	// January 2026: Sunday the 4th is the first Sunday
	firstSundayJan, _ := time.Parse(constants.DateFormat, "2026-01-04")
	if !ShouldScheduleTask(task, firstSundayJan) {
		t.Error("Expected task to be scheduled on first Sunday of January (4th)")
	}

	// Sunday the 11th is the second Sunday
	secondSunday, _ := time.Parse(constants.DateFormat, "2026-01-11")
	if ShouldScheduleTask(task, secondSunday) {
		t.Error("Expected task not to be scheduled on second Sunday")
	}

	// Test last Sunday
	taskLastSunday := models.Task{
		ID:   "last-sunday",
		Name: "Last Sunday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   -1,
			DayOfWeekInMonth: time.Sunday,
		},
	}

	// January 2026: Sunday the 25th is the last Sunday
	lastSundayJan, _ := time.Parse(constants.DateFormat, "2026-01-25")
	if !ShouldScheduleTask(taskLastSunday, lastSundayJan) {
		t.Error("Expected task to be scheduled on last Sunday of January (25th)")
	}

	// Sunday the 18th is NOT the last Sunday
	notLastSunday, _ := time.Parse(constants.DateFormat, "2026-01-18")
	if ShouldScheduleTask(taskLastSunday, notLastSunday) {
		t.Error("Expected task not to be scheduled on non-last Sunday")
	}
}

func TestShouldScheduleTask_MonthlyDay_FifthOccurrence(t *testing.T) {
	task := models.Task{
		ID:   "fifth-friday",
		Name: "Fifth Friday of Month",
		Recurrence: models.Recurrence{
			Type:             constants.RecurrenceMonthlyDay,
			WeekOccurrence:   5,
			DayOfWeekInMonth: time.Friday,
		},
	}

	// January 2026: Friday the 30th is the 5th Friday
	fifthFriday, _ := time.Parse(constants.DateFormat, "2026-01-30")
	if !ShouldScheduleTask(task, fifthFriday) {
		t.Error("Expected task to be scheduled on 5th Friday of January (30th)")
	}

	// Fourth Friday should not match
	fourthFriday, _ := time.Parse(constants.DateFormat, "2026-01-23")
	if ShouldScheduleTask(task, fourthFriday) {
		t.Error("Expected task not to be scheduled on 4th Friday")
	}

	// February 2026 has no 5th Friday (only 4 Fridays)
	// Friday the 27th is the 4th Friday
	feb27, _ := time.Parse(constants.DateFormat, "2026-02-27")
	if ShouldScheduleTask(task, feb27) {
		t.Error("Expected task not to be scheduled on 4th Friday of February (no 5th Friday)")
	}
}

func TestIsNthWeekdayOfMonth_EdgeCases(t *testing.T) {
	// Test 5th occurrence exists
	// January 2026: 5th Friday is the 30th
	fifthFriday, _ := time.Parse(constants.DateFormat, "2026-01-30")
	if !isNthWeekdayOfMonth(fifthFriday, time.Friday, 5) {
		t.Error("Expected Jan 30 to be the 5th Friday")
	}

	// Test month without 5th occurrence
	// February 2026: 4th Friday is the 27th, no 5th Friday
	fourthFridayFeb, _ := time.Parse(constants.DateFormat, "2026-02-27")
	if isNthWeekdayOfMonth(fourthFridayFeb, time.Friday, 5) {
		t.Error("Expected Feb 27 not to be the 5th Friday (only 4 Fridays in Feb 2026)")
	}

	// Test Sunday (0) as first occurrence
	firstSunday, _ := time.Parse(constants.DateFormat, "2026-01-04")
	if !isNthWeekdayOfMonth(firstSunday, time.Sunday, 1) {
		t.Error("Expected Jan 4 to be the first Sunday")
	}

	// Test Sunday (0) as last occurrence
	lastSunday, _ := time.Parse(constants.DateFormat, "2026-01-25")
	if !isNthWeekdayOfMonth(lastSunday, time.Sunday, -1) {
		t.Error("Expected Jan 25 to be the last Sunday")
	}
}
