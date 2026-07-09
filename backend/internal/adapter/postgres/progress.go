package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

// ProgressRepo implements lessons.ProgressRepo.
type ProgressRepo struct{ *Store }

func NewProgressRepo(s *Store) *ProgressRepo { return &ProgressRepo{s} }

func (r *ProgressRepo) GetLog(ctx context.Context, id uuid.UUID) (lessons.LogEntry, error) {
	row, err := r.q.GetProgressLog(ctx, id)
	if err != nil {
		return lessons.LogEntry{}, mapErr(err)
	}
	return lessons.LogEntry{
		ID:          row.ID,
		UserID:      row.UserID,
		ExerciseID:  row.ExerciseID,
		CompletedAt: row.CompletedAt.Time,
		Accuracy:    row.AccuracyScore,
		IsReview:    row.IsReview,
		Detail:      row.Detail,
	}, nil
}

func (r *ProgressRepo) ActivityTimestamps(ctx context.Context, userID uuid.UUID) ([]time.Time, error) {
	rows, err := r.q.ListActivityTimestamps(ctx, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]time.Time, 0, len(rows))
	for _, t := range rows {
		out = append(out, t.Time)
	}
	return out, nil
}

func (r *ProgressRepo) CountUnpassedInLesson(ctx context.Context, lessonID, userID uuid.UUID) (int, error) {
	n, err := r.q.CountUnpassedExercisesInLesson(ctx, gen.CountUnpassedExercisesInLessonParams{
		LessonID: lessonID,
		UserID:   userID,
	})
	return int(n), mapErr(err)
}

func (r *ProgressRepo) IsExercisePassed(ctx context.Context, userID, exerciseID uuid.UUID) (bool, error) {
	var passed bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM progress_logs WHERE user_id = $1 AND exercise_id = $2 AND accuracy_score >= 0.8)`,
		userID, exerciseID).Scan(&passed)
	return passed, mapErr(err)
}

func (r *ProgressRepo) HasLessonXPAward(ctx context.Context, userID, lessonID uuid.UUID) (bool, error) {
	awarded, err := r.q.HasLessonXPAward(ctx, gen.HasLessonXPAwardParams{
		UserID:   userID,
		LessonID: lessonID.String(),
	})
	return awarded, mapErr(err)
}

// RecordAttempt persists the log, streaks and review change atomically.
// inserted=false means the log id already existed (idempotent replay) and
// the transaction wrote nothing else.
func (r *ProgressRepo) RecordAttempt(ctx context.Context, p lessons.RecordParams) (bool, error) {
	inserted := false
	err := r.inTx(ctx, func(q *gen.Queries) error {
		n, err := q.InsertProgressLog(ctx, gen.InsertProgressLogParams{
			ID:            p.Log.ID,
			UserID:        p.Log.UserID,
			ExerciseID:    p.Log.ExerciseID,
			CompletedAt:   ts(p.Log.CompletedAt),
			AccuracyScore: p.Log.Accuracy,
			IsReview:      p.Log.IsReview,
			Detail:        p.Log.Detail,
		})
		if err != nil {
			return mapErr(err)
		}
		if n == 0 {
			return nil // duplicate replay — leave everything untouched
		}
		inserted = true

		if err := q.UpdateStreaks(ctx, gen.UpdateStreaksParams{
			ID:            p.Log.UserID,
			CurrentStreak: int32(p.CurrentStreak), //nolint:gosec // small counters
			LongestStreak: int32(p.LongestStreak), //nolint:gosec // small counters
		}); err != nil {
			return mapErr(err)
		}

		if p.Review == nil {
			return nil
		}
		rc := p.Review
		switch {
		case rc.Delete:
			return mapErr(q.DeleteReviewItem(ctx, rc.ItemID))
		case rc.Insert:
			return mapErr(q.InsertReviewItem(ctx, gen.InsertReviewItemParams{
				ID:           rc.ItemID,
				UserID:       p.Log.UserID,
				ExerciseID:   rc.ExerciseID,
				DueAt:        ts(rc.DueAt),
				IntervalDays: int32(rc.IntervalDays), //nolint:gosec // 1..21
				StreakAt7d:   int32(rc.StreakAt7d),   //nolint:gosec // 0..2
				FailureCount: int32(rc.FailureCount), //nolint:gosec // small counter
				LastResult:   gen.ReviewResult(rc.LastResult),
			}))
		default:
			return mapErr(q.UpdateReviewItem(ctx, gen.UpdateReviewItemParams{
				ID:           rc.ItemID,
				DueAt:        ts(rc.DueAt),
				IntervalDays: int32(rc.IntervalDays), //nolint:gosec // 1..21
				StreakAt7d:   int32(rc.StreakAt7d),   //nolint:gosec // 0..2
				FailureCount: int32(rc.FailureCount), //nolint:gosec // small counter
				LastResult:   gen.ReviewResult(rc.LastResult),
			}))
		}
	})
	return inserted, err
}

func (r *ProgressRepo) GetReviewItem(ctx context.Context, userID, exerciseID uuid.UUID) (lessons.ReviewItemState, error) {
	row, err := r.q.GetReviewItem(ctx, gen.GetReviewItemParams{UserID: userID, ExerciseID: exerciseID})
	if err != nil {
		return lessons.ReviewItemState{}, mapErr(err)
	}
	return lessons.ReviewItemState{
		ItemID:       row.ID,
		IntervalDays: int(row.IntervalDays),
		StreakAt7d:   int(row.StreakAt7d),
		FailureCount: int(row.FailureCount),
	}, nil
}
