package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
)

// DashboardRepo implements dashboard.AggregateRepo.
type DashboardRepo struct{ *Store }

func NewDashboardRepo(s *Store) *DashboardRepo { return &DashboardRepo{s} }

func (r *DashboardRepo) HeatmapCounts(ctx context.Context, userID uuid.UUID, tz string) (map[string]int, error) {
	rows, err := r.q.HeatmapBuckets(ctx, gen.HeatmapBucketsParams{UserID: userID, Tz: tz})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make(map[string]int, len(rows))
	for _, b := range rows {
		out[b.Day.Time.Format("2006-01-02")] = int(b.Interactions)
	}
	return out, nil
}

func (r *DashboardRepo) TotalXP(ctx context.Context, userID uuid.UUID) (int, error) {
	xp, err := r.q.TotalXP(ctx, userID)
	return int(xp), mapErr(err)
}

func (r *DashboardRepo) CountDueReviews(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.CountDueReviewItems(ctx, userID)
	return int(n), mapErr(err)
}

func (r *DashboardRepo) ListDueReviews(ctx context.Context, userID uuid.UUID) ([]dashboard.DueReview, error) {
	rows, err := r.q.ListDueReviewItems(ctx, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]dashboard.DueReview, 0, len(rows))
	for _, row := range rows {
		out = append(out, dashboard.DueReview{
			ID:           row.ReviewQueueItem.ID,
			DueAt:        row.ReviewQueueItem.DueAt.Time,
			FailureCount: int(row.ReviewQueueItem.FailureCount),
			Exercise:     toDomainExercise(row.Exercise),
		})
	}
	return out, nil
}
