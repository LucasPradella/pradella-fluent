// Package review implements the spaced-repetition interval rules (FR-020).
// Pure functions only.
package review

import "time"

// State is the scheduling state of one review-queue item.
type State struct {
	IntervalDays int // last scheduled interval (1, 3, 7, 21)
	StreakAt7d   int // consecutive passes at >=7-day spacing
	FailureCount int // total failures for this item (>=1 once queued)
}

// Outcome is the result of applying a review result to an item's state.
type Outcome struct {
	State   State
	DueIn   time.Duration // when the item should resurface
	Removed bool          // item leaves the queue (mastered)
}

// ladder is the pass progression of intervals in days.
var ladder = [...]int{1, 3, 7, 21}

// nextInterval returns the ladder step after the current interval.
func nextInterval(current int) int {
	for _, step := range ladder {
		if step > current {
			return step
		}
	}
	return ladder[len(ladder)-1]
}

// Initial returns the state and due time for an item entering the queue
// after its first failure.
func Initial() Outcome {
	return Outcome{
		State: State{IntervalDays: 1, StreakAt7d: 0, FailureCount: 1},
		DueIn: 24 * time.Hour,
	}
}

// Apply advances the schedule after a review attempt.
// Pass: move up the 1d→3d→7d→21d ladder; two consecutive passes at >=7d
// spacing remove the item. Fail: reset to 1 day, and repeated failures
// shorten the wait (24h divided by the failure count, floor 6h).
func Apply(s State, passed bool) Outcome {
	if passed {
		if s.IntervalDays >= 7 {
			s.StreakAt7d++
		}
		if s.StreakAt7d >= 2 {
			return Outcome{State: s, Removed: true}
		}
		s.IntervalDays = nextInterval(s.IntervalDays)
		return Outcome{State: s, DueIn: time.Duration(s.IntervalDays) * 24 * time.Hour}
	}

	s.FailureCount++
	s.StreakAt7d = 0
	s.IntervalDays = 1
	wait := 24 * time.Hour / time.Duration(s.FailureCount)
	if wait < 6*time.Hour {
		wait = 6 * time.Hour
	}
	return Outcome{State: s, DueIn: wait}
}
