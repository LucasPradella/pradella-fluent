package progress

import "time"

// HeatmapDay is one bucket of the 90-day activity heatmap (FR-019).
type HeatmapDay struct {
	Date         string // YYYY-MM-DD in the user's timezone
	Interactions int
	Level        int // saturation bucket 0–4
}

// saturationLevel maps an interaction count to a display bucket 0–4.
func saturationLevel(n int) int {
	switch {
	case n <= 0:
		return 0
	case n <= 2:
		return 1
	case n <= 5:
		return 2
	case n <= 9:
		return 3
	default:
		return 4
	}
}

// BuildHeatmap expands per-day counts into exactly 90 consecutive buckets
// ending today (user timezone), zero-filling inactive days.
func BuildHeatmap(counts map[string]int, loc *time.Location, now time.Time) []HeatmapDay {
	today := now.In(loc)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -89)

	out := make([]HeatmapDay, 0, 90)
	for i := 0; i < 90; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		n := counts[date]
		out = append(out, HeatmapDay{Date: date, Interactions: n, Level: saturationLevel(n)})
	}
	return out
}
