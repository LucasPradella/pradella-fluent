// Package dashboard aggregates streaks, XP, the 90-day heatmap and the
// spaced-review queue (US4, FR-018..FR-020).
package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/progress"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

// AggregateRepo reads the dashboard aggregates.
type AggregateRepo interface {
	// HeatmapCounts returns interactions per day (YYYY-MM-DD in tz) for the
	// last ~90 days.
	HeatmapCounts(ctx context.Context, userID uuid.UUID, tz string) (map[string]int, error)
	TotalXP(ctx context.Context, userID uuid.UUID) (int, error)
	CountDueReviews(ctx context.Context, userID uuid.UUID) (int, error)
	ListDueReviews(ctx context.Context, userID uuid.UUID) ([]DueReview, error)
}

// DueReview is a queue item joined with its exercise.
type DueReview struct {
	ID           uuid.UUID
	DueAt        time.Time
	FailureCount int
	Exercise     content.Exercise
}

// Data is the GET /dashboard aggregate.
type Data struct {
	CurrentStreak int
	LongestStreak int
	TotalXP       int
	Heatmap       []progress.HeatmapDay
	DueReviews    int
}

// ReviewItemDTO is a due review with a client-safe exercise shape.
type ReviewItemDTO struct {
	ID           uuid.UUID
	DueAt        time.Time
	FailureCount int
	Exercise     lessons.ExerciseDTO
}

// Service wires the dashboard use cases.
type Service struct {
	users lessons.UserReader
	repo  AggregateRepo
}

func New(users lessons.UserReader, repo AggregateRepo) *Service {
	return &Service{users: users, repo: repo}
}

// Dashboard builds the aggregate for the acting user, bucketing the heatmap
// in the user's IANA timezone (never UTC).
func (s *Service) Dashboard(ctx context.Context, userID uuid.UUID) (Data, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Data{}, err
	}
	counts, err := s.repo.HeatmapCounts(ctx, userID, u.Timezone)
	if err != nil {
		return Data{}, err
	}
	xp, err := s.repo.TotalXP(ctx, userID)
	if err != nil {
		return Data{}, err
	}
	due, err := s.repo.CountDueReviews(ctx, userID)
	if err != nil {
		return Data{}, err
	}
	return Data{
		CurrentStreak: u.CurrentStreak,
		LongestStreak: u.LongestStreak,
		TotalXP:       xp,
		Heatmap:       progress.BuildHeatmap(counts, u.Location(), time.Now()),
		DueReviews:    due,
	}, nil
}

// ReviewQueue lists due items oldest-first with client-safe exercises
// (correct answers stripped; speaking exposes the target sentence).
func (s *Service) ReviewQueue(ctx context.Context, userID uuid.UUID) ([]ReviewItemDTO, error) {
	items, err := s.repo.ListDueReviews(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]ReviewItemDTO, 0, len(items))
	for _, it := range items {
		ex := lessons.ExerciseDTO{
			ID:       it.Exercise.ID,
			Type:     it.Exercise.Type,
			Prompt:   it.Exercise.Prompt,
			Options:  it.Exercise.Options,
			AudioURL: it.Exercise.AudioURL,
		}
		if it.Exercise.Type == content.Speaking {
			ex.TargetSentence = it.Exercise.Target
		}
		out = append(out, ReviewItemDTO{
			ID:           it.ID,
			DueAt:        it.DueAt,
			FailureCount: it.FailureCount,
			Exercise:     ex,
		})
	}
	return out, nil
}
