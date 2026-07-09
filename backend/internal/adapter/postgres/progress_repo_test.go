//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/testutil"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

// seedExercise creates module→lesson→exercise and returns the exercise id.
func seedExercise(t *testing.T, q *gen.Queries) (lessonID, exerciseID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	mid, _ := uuid.NewV7()
	require.NoError(t, q.InsertModule(ctx, gen.InsertModuleParams{
		ID: mid, Title: "M", ThemeType: gen.ThemeTypeTravel,
		DifficultyLevel: gen.ProficiencyLevelBasic, SequentialOrder: 1,
	}))
	lid, _ := uuid.NewV7()
	require.NoError(t, q.InsertLesson(ctx, gen.InsertLessonParams{
		ID: lid, ModuleID: mid, Title: "L", XpReward: 50, SequentialOrder: 1,
	}))
	eid, _ := uuid.NewV7()
	require.NoError(t, q.InsertExercise(ctx, gen.InsertExerciseParams{
		ID: eid, LessonID: lid, ExerciseType: gen.ExerciseTypeTranslate,
		PromptContext: "p", TargetAnswerText: "t", SequentialOrder: 1,
	}))
	return lid, eid
}

func TestProgressLogsInsertOnlyAndIdempotent(t *testing.T) {
	pool := testutil.NewTestPool(t)
	store := postgres.NewStore(pool)
	users := postgres.NewUserRepo(store)
	repo := postgres.NewProgressRepo(store)
	q := gen.New(pool)
	ctx := context.Background()

	u := newUser("ana@example.com")
	require.NoError(t, users.Create(ctx, u))
	_, eid := seedExercise(t, q)

	logID, _ := uuid.NewV7()
	params := lessons.RecordParams{
		Log: lessons.LogEntry{
			ID: logID, UserID: u.ID, ExerciseID: eid,
			CompletedAt: time.Now(), Accuracy: 1.0, Detail: []byte(`{"kind":"writing"}`),
		},
		CurrentStreak: 1, LongestStreak: 1,
	}

	inserted, err := repo.RecordAttempt(ctx, params)
	require.NoError(t, err)
	assert.True(t, inserted)

	// PK-idempotent replay: second insert with the same id writes nothing.
	inserted, err = repo.RecordAttempt(ctx, params)
	require.NoError(t, err)
	assert.False(t, inserted, "duplicate replay must be a no-op")

	// Streaks landed on the user row in the same transaction.
	got, err := users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.CurrentStreak)

	// INSERT-only enforcement (FR-011): UPDATE and DELETE must raise.
	_, err = pool.Exec(ctx, `UPDATE progress_logs SET accuracy_score = 0 WHERE id = $1`, logID)
	assert.ErrorContains(t, err, "INSERT-only", "UPDATE must be blocked by trigger")
	_, err = pool.Exec(ctx, `DELETE FROM progress_logs WHERE id = $1`, logID)
	assert.ErrorContains(t, err, "INSERT-only", "DELETE must be blocked by trigger")
}

func TestReviewRepoUpsertLifecycle(t *testing.T) {
	pool := testutil.NewTestPool(t)
	store := postgres.NewStore(pool)
	users := postgres.NewUserRepo(store)
	repo := postgres.NewProgressRepo(store)
	dash := postgres.NewDashboardRepo(store)
	q := gen.New(pool)
	ctx := context.Background()

	u := newUser("ana@example.com")
	require.NoError(t, users.Create(ctx, u))
	_, eid := seedExercise(t, q)

	// Failed attempt inserts a review item due in the past (test shortcut).
	logID, _ := uuid.NewV7()
	itemID, _ := uuid.NewV7()
	_, err := repo.RecordAttempt(ctx, lessons.RecordParams{
		Log: lessons.LogEntry{
			ID: logID, UserID: u.ID, ExerciseID: eid,
			CompletedAt: time.Now(), Accuracy: 0, Detail: []byte(`{}`),
		},
		Review: &lessons.ReviewChange{
			ItemID: itemID, ExerciseID: eid, Insert: true,
			DueAt: time.Now().Add(-time.Minute), IntervalDays: 1,
			FailureCount: 1, LastResult: "failed",
		},
	})
	require.NoError(t, err)

	st, err := repo.GetReviewItem(ctx, u.ID, eid)
	require.NoError(t, err)
	assert.Equal(t, 1, st.FailureCount)

	// UNIQUE(user, exercise): a second INSERT for the same pair conflicts.
	dupID, _ := uuid.NewV7()
	logID2, _ := uuid.NewV7()
	_, err = repo.RecordAttempt(ctx, lessons.RecordParams{
		Log: lessons.LogEntry{
			ID: logID2, UserID: u.ID, ExerciseID: eid,
			CompletedAt: time.Now(), Accuracy: 0, Detail: []byte(`{}`),
		},
		Review: &lessons.ReviewChange{
			ItemID: dupID, ExerciseID: eid, Insert: true,
			DueAt: time.Now(), IntervalDays: 1, FailureCount: 2, LastResult: "failed",
		},
	})
	assert.Error(t, err, "duplicate (user, exercise) insert must violate the unique index")

	// Update path advances the schedule in place.
	logID3, _ := uuid.NewV7()
	_, err = repo.RecordAttempt(ctx, lessons.RecordParams{
		Log: lessons.LogEntry{
			ID: logID3, UserID: u.ID, ExerciseID: eid,
			CompletedAt: time.Now(), Accuracy: 1, IsReview: true, Detail: []byte(`{}`),
		},
		Review: &lessons.ReviewChange{
			ItemID: itemID, ExerciseID: eid,
			DueAt: time.Now().Add(-time.Second), IntervalDays: 3,
			FailureCount: 1, LastResult: "passed",
		},
	})
	require.NoError(t, err)
	st, err = repo.GetReviewItem(ctx, u.ID, eid)
	require.NoError(t, err)
	assert.Equal(t, 3, st.IntervalDays)

	// Due listing includes the item with its exercise.
	due, err := dash.ListDueReviews(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, eid, due[0].Exercise.ID)

	// 90-day heatmap aggregation groups by user-timezone day.
	counts, err := dash.HeatmapCounts(ctx, u.ID, "America/Sao_Paulo")
	require.NoError(t, err)
	total := 0
	for _, n := range counts {
		total += n
	}
	// Two logs: the conflicting-review transaction rolled back atomically,
	// taking its progress log with it.
	assert.Equal(t, 2, total)
}
