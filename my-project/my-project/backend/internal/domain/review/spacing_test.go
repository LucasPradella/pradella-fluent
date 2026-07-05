package review_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pradella/fluentdev/backend/internal/domain/review"
)

func TestInitialEntry(t *testing.T) {
	out := review.Initial()
	assert.Equal(t, 1, out.State.IntervalDays)
	assert.Equal(t, 1, out.State.FailureCount)
	assert.Equal(t, 24*time.Hour, out.DueIn)
	assert.False(t, out.Removed)
}

func TestPassLadder1_3_7_21(t *testing.T) {
	s := review.State{IntervalDays: 1, FailureCount: 1}

	out := review.Apply(s, true)
	assert.Equal(t, 3, out.State.IntervalDays)
	assert.Equal(t, 3*24*time.Hour, out.DueIn)

	out = review.Apply(out.State, true)
	assert.Equal(t, 7, out.State.IntervalDays)

	out = review.Apply(out.State, true)
	assert.Equal(t, 21, out.State.IntervalDays)
	assert.Equal(t, 1, out.State.StreakAt7d, "pass at 7d counts toward exit")
	assert.False(t, out.Removed)
}

func TestExitAfterTwoPassesAtSevenDaysOrMore(t *testing.T) {
	// Passed at 7d (streak 1) and now passes at 21d → mastered, removed.
	s := review.State{IntervalDays: 21, StreakAt7d: 1, FailureCount: 1}
	out := review.Apply(s, true)
	assert.True(t, out.Removed)
}

func TestFailResetsToOneDay(t *testing.T) {
	s := review.State{IntervalDays: 21, StreakAt7d: 1, FailureCount: 1}
	out := review.Apply(s, false)
	assert.False(t, out.Removed)
	assert.Equal(t, 1, out.State.IntervalDays)
	assert.Equal(t, 0, out.State.StreakAt7d, "7d streak resets on failure")
	assert.Equal(t, 2, out.State.FailureCount)
	assert.Equal(t, 12*time.Hour, out.DueIn, "second failure comes back sooner")
}

func TestRepeatedFailuresShortenTheWait(t *testing.T) {
	s := review.State{IntervalDays: 1, FailureCount: 2}
	out := review.Apply(s, false) // 3rd failure → 24h/3 = 8h
	assert.Equal(t, 8*time.Hour, out.DueIn)

	s = review.State{IntervalDays: 1, FailureCount: 5}
	out = review.Apply(s, false) // 6th failure → 24h/6 = 4h, floored at 6h
	assert.Equal(t, 6*time.Hour, out.DueIn)
}
