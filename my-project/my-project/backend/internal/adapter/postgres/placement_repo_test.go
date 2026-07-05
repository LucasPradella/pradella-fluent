//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/testutil"
	"github.com/pradella/fluentdev/backend/internal/domain"
	dplacement "github.com/pradella/fluentdev/backend/internal/domain/placement"
)

func TestPlacementRepoConstraints(t *testing.T) {
	pool := testutil.NewTestPool(t)
	store := postgres.NewStore(pool)
	users := postgres.NewUserRepo(store)
	repo := postgres.NewPlacementRepo(store)
	q := gen.New(pool)
	ctx := context.Background()

	u := newUser("ana@example.com")
	require.NoError(t, users.Create(ctx, u))

	// Seed one question.
	qid, _ := uuid.NewV7()
	require.NoError(t, q.InsertPlacementQuestion(ctx, gen.InsertPlacementQuestionParams{
		ID: qid, CefrBand: gen.CefrBandB1, QuestionType: gen.PlacementQuestionTypeChoice,
		Prompt: "test", Options: []byte(`["ok","no"]`), CorrectOption: "ok",
	}))

	sid, _ := uuid.NewV7()
	sess, err := repo.CreateSession(ctx, sid, u.ID)
	require.NoError(t, err)
	assert.Equal(t, dplacement.StartBand, sess.CurrentBand, "session starts at B1")

	// Partial unique: a second ACTIVE session for the same user must fail.
	sid2, _ := uuid.NewV7()
	_, err = repo.CreateSession(ctx, sid2, u.ID)
	assert.ErrorIs(t, err, domain.ErrConflict, "one active session per user")

	// Answers: no repeats per (session, question).
	aid, _ := uuid.NewV7()
	require.NoError(t, repo.InsertAnswer(ctx, aid, sid, qid, 0, true))
	aid2, _ := uuid.NewV7()
	err = repo.InsertAnswer(ctx, aid2, sid, qid, 0, true)
	assert.ErrorIs(t, err, domain.ErrConflict, "question repeats must be rejected")

	// Question already answered is not served again.
	_, err = repo.PickUnservedQuestion(ctx, dplacement.BandB1, sid)
	assert.ErrorIs(t, err, domain.ErrNotFound, "bank exhausted for this session")

	// Complete + assign level atomically; user level lands.
	require.NoError(t, repo.CompleteAndAssignLevel(ctx, sid, u.ID, domain.LevelIntermediate))
	placed, err := users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.LevelIntermediate, placed.Level)

	// After completion, a new session may start (partial index frees up).
	sid3, _ := uuid.NewV7()
	_, err = repo.CreateSession(ctx, sid3, u.ID)
	assert.NoError(t, err)
}
