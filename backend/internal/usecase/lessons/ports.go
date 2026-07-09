// Package lessons implements track listing, lesson delivery and attempt
// scoring for written/listening exercises (US2, FR-006..FR-012).
package lessons

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// ContentRepo reads learning content.
type ContentRepo interface {
	ListModules(ctx context.Context) ([]content.Module, error)
	ListLessons(ctx context.Context) ([]content.Lesson, error)
	CompletedLessonIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetLessonWithModule(ctx context.Context, lessonID uuid.UUID) (content.Lesson, content.Module, error)
	ListExercises(ctx context.Context, lessonID uuid.UUID) ([]content.Exercise, error)
	GetExerciseContext(ctx context.Context, exerciseID uuid.UUID) (content.Exercise, content.Lesson, content.Module, error)
}

// LogEntry mirrors one immutable progress_logs row.
type LogEntry struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ExerciseID  uuid.UUID
	CompletedAt time.Time
	Accuracy    float64
	IsReview    bool
	Detail      []byte // JSON
}

// ReviewChange is the review-queue side effect of an attempt, applied in the
// same transaction as the progress log (T058).
type ReviewChange struct {
	Delete       bool // remove the item (mastered)
	ItemID       uuid.UUID
	ExerciseID   uuid.UUID
	Insert       bool // true = new item, false = update existing
	DueAt        time.Time
	IntervalDays int
	StreakAt7d   int
	FailureCount int
	LastResult   string // failed | passed
}

// ReviewItemState is the current scheduling state of a queued exercise.
type ReviewItemState struct {
	ItemID       uuid.UUID
	IntervalDays int
	StreakAt7d   int
	FailureCount int
}

// RecordParams is everything RecordAttempt must persist atomically:
// the immutable log, the recomputed streaks and the review-queue change.
type RecordParams struct {
	Log           LogEntry
	CurrentStreak int
	LongestStreak int
	Review        *ReviewChange // nil = no queue change
}

// ProgressRepo persists attempts and reads activity aggregates.
type ProgressRepo interface {
	// GetLog returns domain.ErrNotFound when the attempt id is unknown.
	GetLog(ctx context.Context, id uuid.UUID) (LogEntry, error)
	ActivityTimestamps(ctx context.Context, userID uuid.UUID) ([]time.Time, error)
	CountUnpassedInLesson(ctx context.Context, lessonID, userID uuid.UUID) (int, error)
	IsExercisePassed(ctx context.Context, userID, exerciseID uuid.UUID) (bool, error)
	HasLessonXPAward(ctx context.Context, userID, lessonID uuid.UUID) (bool, error)
	// RecordAttempt executes atomically; inserted=false means the log id
	// already existed (idempotent outbox replay) and nothing was written.
	RecordAttempt(ctx context.Context, p RecordParams) (inserted bool, err error)
	// GetReviewItem returns domain.ErrNotFound when the exercise is not queued.
	GetReviewItem(ctx context.Context, userID, exerciseID uuid.UUID) (ReviewItemState, error)
}

// UserReader loads the acting user (level, timezone, streaks).
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
}
