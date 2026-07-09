package lessons_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
	"github.com/pradella/fluentdev/backend/internal/testutil"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

type fixture struct {
	svc      *lessons.Service
	users    *testutil.MemUserRepo
	progress *testutil.MemProgressRepo
	userID   uuid.UUID
	lesson   content.Lesson
	ex1, ex2 content.Exercise
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	users := testutil.NewMemUserRepo()
	progress := testutil.NewMemProgressRepo()

	userID := uuid.New()
	users.Users[userID] = user.User{
		ID: userID, Email: "dev@example.com", DisplayName: "Dev",
		Level: domain.LevelBasic, Timezone: "America/Sao_Paulo",
	}

	moduleID, lessonID := uuid.New(), uuid.New()
	ex1 := content.Exercise{
		ID: uuid.New(), LessonID: lessonID, Type: content.Translate,
		Prompt: "Traduza", Target: "I have one bag to check", Order: 1,
	}
	ex2 := content.Exercise{
		ID: uuid.New(), LessonID: lessonID, Type: content.ListeningChoice,
		Prompt: "Ouça", Target: "Your passport, please",
		Options: []string{"Your passport, please", "Gate is closed"}, Order: 2,
	}
	contentRepo := &testutil.MemContentRepo{
		Modules:   []content.Module{{ID: moduleID, Title: "M1", Theme: "travel", Difficulty: domain.LevelBasic, Order: 1}},
		Lessons:   []content.Lesson{{ID: lessonID, ModuleID: moduleID, Title: "L1", XP: 50, Order: 1}},
		Exercises: []content.Exercise{ex1, ex2},
		Progress:  progress,
	}
	progress.Content = contentRepo

	return &fixture{
		svc:      lessons.New(contentRepo, progress, users),
		users:    users,
		progress: progress,
		userID:   userID,
		lesson:   contentRepo.Lessons[0],
		ex1:      ex1,
		ex2:      ex2,
	}
}

func TestCorrectAnswerWithTypoTolerated(t *testing.T) {
	f := newFixture(t)
	res, err := f.svc.SubmitAttempt(context.Background(), f.userID, f.ex1.ID, uuid.New(),
		"I have one bag to chek", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res.Correct)
	assert.Equal(t, []string{"chek"}, res.ToleratedTypos)
	assert.Equal(t, 1.0, res.AccuracyScore)
	assert.False(t, res.LessonCompleted, "second exercise still unpassed")
	assert.Zero(t, res.XPAwarded)
}

func TestWrongAnswerShowsExpectedAndQueuesReview(t *testing.T) {
	f := newFixture(t)
	res, err := f.svc.SubmitAttempt(context.Background(), f.userID, f.ex1.ID, uuid.New(),
		"I have one dog to check", time.Time{}, false)
	require.NoError(t, err)
	assert.False(t, res.Correct)
	assert.Equal(t, "I have one bag to check", res.ExpectedAnswer)

	// Failed item entered the review queue.
	_, err = f.progress.GetReviewItem(context.Background(), f.userID, f.ex1.ID)
	assert.NoError(t, err)
}

func TestLessonCompletionAwardsXPOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	res1, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, uuid.New(),
		"I have one bag to check", time.Time{}, false)
	require.NoError(t, err)
	assert.False(t, res1.LessonCompleted)

	res2, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"Your passport, please", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res2.LessonCompleted, "all exercises passed")
	assert.Equal(t, 50, res2.XPAwarded)

	// Re-passing the lesson never awards XP again.
	res3, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"Your passport, please", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res3.LessonCompleted)
	assert.Zero(t, res3.XPAwarded, "XP is a one-time award per lesson")
}

func TestIdempotentReplayByAttemptID(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	attemptID := uuid.New()

	res1, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, attemptID,
		"I have one bag to check", time.Time{}, false)
	require.NoError(t, err)
	assert.False(t, res1.Duplicate)

	// Outbox replays the same attemptId: same result, flagged duplicate,
	// and only one log recorded.
	res2, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, attemptID,
		"I have one bag to check", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res2.Duplicate)
	assert.Equal(t, res1.Correct, res2.Correct)
	assert.Len(t, f.progress.Logs, 1)
}

func TestFutureCompletedAtClamped(t *testing.T) {
	f := newFixture(t)
	future := time.Now().Add(48 * time.Hour)
	attemptID := uuid.New()
	_, err := f.svc.SubmitAttempt(context.Background(), f.userID, f.ex1.ID, attemptID,
		"I have one bag to check", future, false)
	require.NoError(t, err)
	logged := f.progress.Logs[attemptID]
	assert.True(t, logged.CompletedAt.Before(time.Now().Add(time.Minute)),
		"client timestamps in the future must be clamped to server now")
}

func TestLockedTrackForbidden(t *testing.T) {
	f := newFixture(t)
	// Demote module to advanced while the user is basic.
	f.progress.Content.Modules[0].Difficulty = domain.LevelAdvanced
	_, err := f.svc.SubmitAttempt(context.Background(), f.userID, f.ex1.ID, uuid.New(),
		"anything", time.Time{}, false)
	assert.ErrorIs(t, err, domain.ErrForbidden)

	_, err = f.svc.Lesson(context.Background(), f.userID, f.lesson.ID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestListeningChoiceScoring(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	res, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"Your passport, please", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res.Correct)

	res, err = f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"Gate is closed", time.Time{}, false)
	require.NoError(t, err)
	assert.False(t, res.Correct)
	assert.Equal(t, "Your passport, please", res.ExpectedAnswer)
}

func TestListeningOrderScoring(t *testing.T) {
	f := newFixture(t)
	f.progress.Content.Exercises[1].Type = content.ListeningOrder
	f.progress.Content.Exercises[1].Target = "how long will you stay"
	ctx := context.Background()

	res, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"How long will you stay", time.Time{}, false)
	require.NoError(t, err)
	assert.True(t, res.Correct, "word order matches, case/punctuation ignored")

	res, err = f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"how will long you stay", time.Time{}, false)
	require.NoError(t, err)
	assert.False(t, res.Correct, "wrong word order fails")
}

func TestReviewPassAdvancesLadderAndRemovesWhenMastered(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Item passed once at >=7d (streak 1), now at 21d: a review pass
	// removes it from the queue (mastered).
	itemID := uuid.New()
	f.progress.SeedReviewState(itemID, f.userID, f.ex1.ID, time.Now().Add(-time.Hour),
		lessons.ReviewItemState{IntervalDays: 21, StreakAt7d: 1, FailureCount: 1})

	res, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, uuid.New(),
		"I have one bag to check", time.Time{}, true)
	require.NoError(t, err)
	require.True(t, res.Correct)
	assert.False(t, f.progress.HasReviewItemID(itemID), "mastered item must leave the queue")

	// A fresh 1d item passing during review advances to 3d (stays queued).
	item2 := uuid.New()
	f.progress.SeedReviewState(item2, f.userID, f.ex2.ID, time.Now().Add(-time.Hour),
		lessons.ReviewItemState{IntervalDays: 1, FailureCount: 1})
	res, err = f.svc.SubmitAttempt(ctx, f.userID, f.ex2.ID, uuid.New(),
		"Your passport, please", time.Time{}, true)
	require.NoError(t, err)
	require.True(t, res.Correct)
	st, err := f.progress.GetReviewItem(ctx, f.userID, f.ex2.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, st.IntervalDays)
}

func TestRepeatedFailureIncrementsCount(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	_, err := f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, uuid.New(),
		"totally wrong words here now", time.Time{}, false)
	require.NoError(t, err)
	_, err = f.svc.SubmitAttempt(ctx, f.userID, f.ex1.ID, uuid.New(),
		"still very wrong words here", time.Time{}, false)
	require.NoError(t, err)

	st, err := f.progress.GetReviewItem(ctx, f.userID, f.ex1.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, st.FailureCount)
}

func TestSpeakingExerciseRejectedOnWrittenEndpoint(t *testing.T) {
	f := newFixture(t)
	f.progress.Content.Exercises[0].Type = content.Speaking
	_, err := f.svc.SubmitAttempt(context.Background(), f.userID, f.ex1.ID, uuid.New(),
		"anything", time.Time{}, false)
	assert.ErrorIs(t, err, domain.ErrInvalid)
}
