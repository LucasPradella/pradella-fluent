package lessons

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/progress"
	"github.com/pradella/fluentdev/backend/internal/domain/review"
	"github.com/pradella/fluentdev/backend/internal/domain/speech"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// AttemptResult is the scored outcome returned to the client.
type AttemptResult struct {
	Correct         bool     `json:"correct"`
	AccuracyScore   float64  `json:"accuracyScore"`
	ToleratedTypos  []string `json:"toleratedTypos,omitempty"`
	ExpectedAnswer  string   `json:"expectedAnswer,omitempty"`
	LessonCompleted bool     `json:"lessonCompleted"`
	XPAwarded       int      `json:"xpAwarded"`
	Duplicate       bool     `json:"-"` // replayed outbox row → 200 instead of 201
}

// attemptDetail is what we persist in progress_logs.detail — enough to
// replay the exact response for duplicate submissions.
type attemptDetail struct {
	Kind            string   `json:"kind"` // writing | listening_choice | listening_order
	Correct         bool     `json:"correct"`
	ToleratedTypos  []string `json:"toleratedTypos,omitempty"`
	ExpectedAnswer  string   `json:"expectedAnswer,omitempty"`
	LessonCompleted bool     `json:"lessonCompleted"`
	XPAwarded       int      `json:"xpAwarded,omitempty"`
	XPLessonID      string   `json:"xpLessonId,omitempty"`
}

// SubmitAttempt validates and scores a written or listening answer,
// then atomically logs progress, recomputes streaks and feeds the review
// queue. Idempotent by attemptID (offline outbox replay).
func (s *Service) SubmitAttempt(ctx context.Context, userID, exerciseID, attemptID uuid.UUID, answer string, completedAt time.Time, isReview bool) (AttemptResult, error) {
	// Duplicate replay short-circuits before any validation. The log must
	// belong to the acting user — replaying someone else's attempt id is a
	// cross-user access attempt (OWASP A01).
	if prior, err := s.progress.GetLog(ctx, attemptID); err == nil {
		if prior.UserID != userID {
			return AttemptResult{}, domain.ErrForbidden
		}
		return replayResult(prior)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return AttemptResult{}, err
	}
	ex, lesson, module, err := s.content.GetExerciseContext(ctx, exerciseID)
	if err != nil {
		return AttemptResult{}, err
	}
	if module.LockedFor(u.Level) {
		return AttemptResult{}, fmt.Errorf("%w: track requires level %s", domain.ErrForbidden, module.Difficulty)
	}
	if ex.Type == content.Speaking {
		return AttemptResult{}, fmt.Errorf("%w: speaking exercises use the speech endpoint", domain.ErrInvalid)
	}

	result := scoreAnswer(ex, answer)
	if !result.Correct {
		result.ExpectedAnswer = ex.Target // US2 scenario 3
	}

	now := time.Now()
	if completedAt.IsZero() || completedAt.After(now) {
		completedAt = now // clamp client-reported offline timestamps
	}

	accuracy := 0.0
	if result.Correct {
		accuracy = 1.0
	}

	// Lesson completion + one-time XP award.
	if result.Correct {
		unpassed, err := s.progress.CountUnpassedInLesson(ctx, lesson.ID, userID)
		if err != nil {
			return AttemptResult{}, err
		}
		passedBefore, err := s.progress.IsExercisePassed(ctx, userID, exerciseID)
		if err != nil {
			return AttemptResult{}, err
		}
		remaining := unpassed
		if !passedBefore {
			remaining--
		}
		if remaining <= 0 {
			result.LessonCompleted = true
			awarded, err := s.progress.HasLessonXPAward(ctx, userID, lesson.ID)
			if err != nil {
				return AttemptResult{}, err
			}
			if !awarded {
				result.XPAwarded = lesson.XP
			}
		}
	}

	detail := attemptDetail{
		Kind:            string(ex.Type),
		Correct:         result.Correct,
		ToleratedTypos:  result.ToleratedTypos,
		ExpectedAnswer:  result.ExpectedAnswer,
		LessonCompleted: result.LessonCompleted,
		XPAwarded:       result.XPAwarded,
	}
	if result.XPAwarded > 0 {
		detail.XPLessonID = lesson.ID.String()
	}
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return AttemptResult{}, err
	}

	params, err := s.buildRecord(ctx, u, LogEntry{
		ID:          attemptID,
		UserID:      userID,
		ExerciseID:  exerciseID,
		CompletedAt: completedAt,
		Accuracy:    accuracy,
		IsReview:    isReview,
		Detail:      detailJSON,
	}, result.Correct, isReview)
	if err != nil {
		return AttemptResult{}, err
	}

	inserted, err := s.progress.RecordAttempt(ctx, params)
	if err != nil {
		return AttemptResult{}, err
	}
	if !inserted { // lost a race with a concurrent replay
		if prior, err := s.progress.GetLog(ctx, attemptID); err == nil {
			return replayResult(prior)
		}
	}
	return result, nil
}

// scoreAnswer applies the per-type validation rules (FR-009, FR-010).
func scoreAnswer(ex content.Exercise, answer string) AttemptResult {
	switch ex.Type {
	case content.Translate, content.FillBlank:
		r := speech.ValidateWriting(ex.Target, answer)
		return AttemptResult{Correct: r.Correct, ToleratedTypos: r.ToleratedTypos, AccuracyScore: boolScore(r.Correct)}
	case content.ListeningChoice:
		return AttemptResult{Correct: answer == ex.Target, AccuracyScore: boolScore(answer == ex.Target)}
	case content.ListeningOrder:
		want := speech.Normalize(ex.Target)
		got := speech.Normalize(answer)
		correct := len(want) == len(got)
		if correct {
			for i := range want {
				if want[i] != got[i] {
					correct = false
					break
				}
			}
		}
		return AttemptResult{Correct: correct, AccuracyScore: boolScore(correct)}
	default:
		return AttemptResult{}
	}
}

func boolScore(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// buildRecord recomputes streaks with the new activity and derives the
// review-queue change (fail → enqueue/reset; review pass → advance/remove).
func (s *Service) buildRecord(ctx context.Context, u user.User, log LogEntry, correct, isReview bool) (RecordParams, error) {
	activity, err := s.progress.ActivityTimestamps(ctx, u.ID)
	if err != nil {
		return RecordParams{}, err
	}
	activity = append(activity, log.CompletedAt)
	streaks := progress.ComputeStreaks(activity, u.Location(), time.Now())

	params := RecordParams{
		Log:           log,
		CurrentStreak: streaks.Current,
		LongestStreak: max(streaks.Longest, u.LongestStreak),
	}

	change, err := s.reviewChange(ctx, u.ID, log.ExerciseID, log.CompletedAt, correct, isReview)
	if err != nil {
		return RecordParams{}, err
	}
	params.Review = change
	return params, nil
}

// reviewChange computes the spaced-repetition side effect of an attempt.
func (s *Service) reviewChange(ctx context.Context, userID, exerciseID uuid.UUID, at time.Time, correct, isReview bool) (*ReviewChange, error) {
	item, err := s.progress.GetReviewItem(ctx, userID, exerciseID)
	known := err == nil

	if !correct {
		if !known {
			out := review.Initial()
			id, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			return &ReviewChange{
				ItemID: id, ExerciseID: exerciseID, Insert: true,
				DueAt:        at.Add(out.DueIn),
				IntervalDays: out.State.IntervalDays,
				StreakAt7d:   out.State.StreakAt7d,
				FailureCount: out.State.FailureCount,
				LastResult:   "failed",
			}, nil
		}
		out := review.Apply(review.State{
			IntervalDays: item.IntervalDays,
			StreakAt7d:   item.StreakAt7d,
			FailureCount: item.FailureCount,
		}, false)
		return &ReviewChange{
			ItemID: item.ItemID, ExerciseID: exerciseID,
			DueAt:        at.Add(out.DueIn),
			IntervalDays: out.State.IntervalDays,
			StreakAt7d:   out.State.StreakAt7d,
			FailureCount: out.State.FailureCount,
			LastResult:   "failed",
		}, nil
	}

	// Correct answers only advance the schedule during review sessions.
	if !isReview || !known {
		return nil, nil
	}
	out := review.Apply(review.State{
		IntervalDays: item.IntervalDays,
		StreakAt7d:   item.StreakAt7d,
		FailureCount: item.FailureCount,
	}, true)
	if out.Removed {
		return &ReviewChange{ItemID: item.ItemID, ExerciseID: exerciseID, Delete: true}, nil
	}
	return &ReviewChange{
		ItemID: item.ItemID, ExerciseID: exerciseID,
		DueAt:        at.Add(out.DueIn),
		IntervalDays: out.State.IntervalDays,
		StreakAt7d:   out.State.StreakAt7d,
		FailureCount: out.State.FailureCount,
		LastResult:   "passed",
	}, nil
}

// BuildRecordForSpeech lets the speech usecase share the streak/review
// pipeline so failed speaking attempts feed the same queue (T058).
func (s *Service) BuildRecordForSpeech(ctx context.Context, u user.User, log LogEntry, correct, isReview bool) (RecordParams, error) {
	return s.buildRecord(ctx, u, log, correct, isReview)
}

// replayResult rebuilds the original response from a persisted log entry.
func replayResult(entry LogEntry) (AttemptResult, error) {
	var d attemptDetail
	if err := json.Unmarshal(entry.Detail, &d); err != nil {
		return AttemptResult{}, fmt.Errorf("corrupt attempt detail: %w", err)
	}
	return AttemptResult{
		Correct:         d.Correct,
		AccuracyScore:   entry.Accuracy,
		ToleratedTypos:  d.ToleratedTypos,
		ExpectedAnswer:  d.ExpectedAnswer,
		LessonCompleted: d.LessonCompleted,
		XPAwarded:       d.XPAwarded,
		Duplicate:       true,
	}, nil
}
