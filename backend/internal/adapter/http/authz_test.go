package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// Cross-user object access tests (OWASP A01 — quickstart V12).

func TestCrossUserAttemptReplayForbidden(t *testing.T) {
	h := newHarness(t)

	alice := h.client(t)
	placeUser(t, h, alice, "alice@example.com", domain.LevelBasic)
	attemptID := uuid.New().String()
	resp := alice.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts",
		map[string]any{"attemptId": attemptID, "answer": "I have one bag to check"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Bob replays Alice's attemptId: must not read her result.
	bob := h.client(t)
	placeUser(t, h, bob, "bob@example.com", domain.LevelBasic)
	resp = bob.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts",
		map[string]any{"attemptId": attemptID, "answer": "I have one bag to check"})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"replaying another user's attempt id must be denied")
}

func TestReviewQueueIsUserScoped(t *testing.T) {
	h := newHarness(t)

	alice := h.client(t)
	aliceID := placeUser(t, h, alice, "alice@example.com", domain.LevelBasic)
	h.progress.SeedReview(uuid.New(), aliceID, h.writingEx.ID, time.Now().Add(-time.Hour), 1)

	bob := h.client(t)
	placeUser(t, h, bob, "bob@example.com", domain.LevelBasic)

	resp := bob.do(http.MethodGet, "/api/v1/review-queue", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, decodeList(t, resp), "bob must never see alice's review items")
}

func TestDashboardIsUserScoped(t *testing.T) {
	h := newHarness(t)

	alice := h.client(t)
	placeUser(t, h, alice, "alice@example.com", domain.LevelBasic)
	resp := alice.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts",
		map[string]any{"attemptId": uuid.New().String(), "answer": "I have one bag to check"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	bob := h.client(t)
	placeUser(t, h, bob, "bob@example.com", domain.LevelBasic)
	resp = bob.do(http.MethodGet, "/api/v1/dashboard", nil)
	defer resp.Body.Close()
	body := decode(t, resp)
	assert.EqualValues(t, 0, body["currentStreak"], "bob's dashboard must not include alice's activity")

	heatmap := body["heatmap"].([]any)
	last := heatmap[89].(map[string]any)
	assert.EqualValues(t, 0, last["interactions"])
}

func TestSessionCookieIsTheOnlyCredential(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	// No session: every protected route denies by default.
	for _, path := range []string{"/api/v1/tracks", "/api/v1/dashboard", "/api/v1/review-queue", "/api/v1/placement/session"} {
		resp := c.do(http.MethodGet, path, nil)
		resp.Body.Close()
		assert.Equalf(t, http.StatusUnauthorized, resp.StatusCode, "GET %s must require a session", path)
	}
}
