package progress_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain/progress"
)

var saoPaulo = mustLoad("America/Sao_Paulo")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func at(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, saoPaulo)
}

func TestConsecutiveDaysIncrement(t *testing.T) {
	now := at(2026, 7, 3, 20, 0)
	s := progress.ComputeStreaks([]time.Time{
		at(2026, 7, 1, 10, 0),
		at(2026, 7, 2, 11, 0),
		at(2026, 7, 3, 9, 0),
	}, saoPaulo, now)
	assert.Equal(t, 3, s.Current)
	assert.Equal(t, 3, s.Longest)
}

func TestGapResetsCurrentButKeepsLongest(t *testing.T) {
	now := at(2026, 7, 10, 12, 0)
	s := progress.ComputeStreaks([]time.Time{
		at(2026, 7, 1, 10, 0),
		at(2026, 7, 2, 10, 0),
		at(2026, 7, 3, 10, 0),
		// gap 4..8
		at(2026, 7, 9, 10, 0),
		at(2026, 7, 10, 10, 0),
	}, saoPaulo, now)
	assert.Equal(t, 2, s.Current)
	assert.Equal(t, 3, s.Longest, "longest streak must be preserved")
}

func TestYesterdayActivityKeepsStreakAlive(t *testing.T) {
	// No activity yet today: streak still counts up to yesterday.
	now := at(2026, 7, 4, 8, 0)
	s := progress.ComputeStreaks([]time.Time{
		at(2026, 7, 2, 10, 0),
		at(2026, 7, 3, 10, 0),
	}, saoPaulo, now)
	assert.Equal(t, 2, s.Current)
}

func TestTwoDayGapBreaksStreak(t *testing.T) {
	now := at(2026, 7, 5, 8, 0)
	s := progress.ComputeStreaks([]time.Time{
		at(2026, 7, 2, 10, 0),
		at(2026, 7, 3, 10, 0),
	}, saoPaulo, now)
	assert.Equal(t, 0, s.Current)
	assert.Equal(t, 2, s.Longest)
}

func TestMidnightBoundaryUsesUserTimezone(t *testing.T) {
	// 23:59 and 00:01 across midnight in São Paulo are different local
	// days — consecutive, so streak = 2. In UTC both would land on the
	// same date (02:59Z and 03:01Z), collapsing the streak to 1.
	require.NotNil(t, saoPaulo)
	nights := []time.Time{
		at(2026, 7, 1, 23, 59),
		at(2026, 7, 2, 0, 1),
	}
	s := progress.ComputeStreaks(nights, saoPaulo, at(2026, 7, 2, 9, 0))
	assert.Equal(t, 2, s.Current, "day bucketing must follow the user timezone")

	utc := progress.ComputeStreaks(nights, time.UTC, at(2026, 7, 2, 9, 0).In(time.UTC))
	assert.Equal(t, 1, utc.Current, "sanity: UTC would collapse the two days")
}

func TestEmptyActivity(t *testing.T) {
	s := progress.ComputeStreaks(nil, saoPaulo, at(2026, 7, 4, 8, 0))
	assert.Zero(t, s.Current)
	assert.Zero(t, s.Longest)
}

func TestBuildHeatmapShapeAndLevels(t *testing.T) {
	now := at(2026, 7, 4, 12, 0)
	counts := map[string]int{
		"2026-07-04": 1,  // level 1
		"2026-07-03": 4,  // level 2
		"2026-07-02": 7,  // level 3
		"2026-07-01": 12, // level 4
	}
	hm := progress.BuildHeatmap(counts, saoPaulo, now)
	assert.Len(t, hm, 90)
	assert.Equal(t, "2026-07-04", hm[89].Date)
	assert.Equal(t, 1, hm[89].Level)
	assert.Equal(t, 2, hm[88].Level)
	assert.Equal(t, 3, hm[87].Level)
	assert.Equal(t, 4, hm[86].Level)
	assert.Equal(t, 0, hm[0].Level)
	assert.Equal(t, "2026-04-06", hm[0].Date) // 89 days before today
}
