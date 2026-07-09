// Package progress implements streak and heatmap rules over the immutable
// activity log (FR-018, FR-019). Pure functions, user-timezone aware.
package progress

import "time"

// Streaks summarizes consecutive-day activity.
type Streaks struct {
	Current int
	Longest int
}

// dayKey buckets an instant into a calendar day in the user's timezone.
func dayKey(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}

// ComputeStreaks derives current and longest streaks from activity
// timestamps. Days are bucketed in the user's IANA timezone (never UTC).
// The current streak survives until the end of the day after the last
// activity: activity yesterday but none yet today still counts.
func ComputeStreaks(activity []time.Time, loc *time.Location, now time.Time) Streaks {
	if len(activity) == 0 {
		return Streaks{}
	}

	days := make(map[string]bool, len(activity))
	for _, t := range activity {
		days[dayKey(t, loc)] = true
	}

	today := now.In(loc)
	midnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

	// Current: walk back from today (or yesterday, if today is inactive).
	current := 0
	cursor := midnight
	if !days[cursor.Format("2006-01-02")] {
		cursor = cursor.AddDate(0, 0, -1)
	}
	for days[cursor.Format("2006-01-02")] {
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}

	// Longest: scan every active day for run starts.
	longest := 0
	for k := range days {
		day, err := time.ParseInLocation("2006-01-02", k, loc)
		if err != nil {
			continue
		}
		prev := day.AddDate(0, 0, -1).Format("2006-01-02")
		if days[prev] {
			continue // not a run start
		}
		run := 0
		for d := day; days[d.Format("2006-01-02")]; d = d.AddDate(0, 0, 1) {
			run++
		}
		if run > longest {
			longest = run
		}
	}

	return Streaks{Current: current, Longest: longest}
}
