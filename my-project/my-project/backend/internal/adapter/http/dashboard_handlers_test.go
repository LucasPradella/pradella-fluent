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

func TestDashboardShape(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	// One completed writing attempt for some activity.
	resp := c.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts",
		map[string]any{"attemptId": uuid.New().String(), "answer": "I have one bag to check"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = c.do(http.MethodGet, "/api/v1/dashboard", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)

	heatmap := body["heatmap"].([]any)
	assert.Len(t, heatmap, 90, "heatmap must contain exactly 90 buckets")
	for _, raw := range heatmap {
		day := raw.(map[string]any)
		lvl := day["level"].(float64)
		assert.GreaterOrEqual(t, lvl, 0.0)
		assert.LessOrEqual(t, lvl, 4.0, "saturation level is bucketed 0–4")
	}
	last := heatmap[89].(map[string]any)
	assert.EqualValues(t, 1, last["interactions"], "today's attempt must appear in the last bucket")
	assert.EqualValues(t, 1, body["currentStreak"])
}

func TestReviewQueueDueOnlyOrdering(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	uid := placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	// One overdue, one older-overdue, one not-yet-due.
	h.progress.SeedReview(uuid.New(), uid, h.writingEx.ID, time.Now().Add(-2*time.Hour), 1)
	h.progress.SeedReview(uuid.New(), uid, h.speakingEx.ID, time.Now().Add(-48*time.Hour), 2)
	h.progress.SeedReview(uuid.New(), uid, h.advancedEx.ID, time.Now().Add(24*time.Hour), 1)

	resp := c.do(http.MethodGet, "/api/v1/review-queue", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	items := decodeList(t, resp)
	require.Len(t, items, 2, "only due items are returned")

	// Oldest first.
	ex0 := items[0]["exercise"].(map[string]any)
	assert.Equal(t, h.speakingEx.ID.String(), ex0["id"])
	assert.EqualValues(t, 2, items[0]["failureCount"])

	// Dashboard counts the same due items.
	resp = c.do(http.MethodGet, "/api/v1/dashboard", nil)
	defer resp.Body.Close()
	assert.EqualValues(t, 2, decode(t, resp)["dueReviews"])
}
